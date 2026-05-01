package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/audit"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
	"github.com/moon-panel/moon-panel/internal/security"
)

// SecurityHandler exposes admin endpoints for the Phase 4a security UX:
//   - List + add + remove entries in the trusted-IP CIDR allowlist.
//   - View currently-locked IPs and manually unlock them.
//
// The trusted list is persisted as a single Setting key
// (security.trusted_ips, JSON-encoded array of {cidr, note}). Updates
// re-parse and atomically swap the in-memory matcher used by LoginLockout
// — no container restart needed.
type SecurityHandler struct {
	DB           *gorm.DB
	LoginLockout *middleware.LoginLockout
	TOTPLockout  *middleware.LoginLockout
}

const trustedIPsSettingKey = "security.trusted_ips"

type trustedEntry struct {
	CIDR    string    `json:"cidr"`
	Note    string    `json:"note,omitempty"`
	AddedAt time.Time `json:"added_at"`
}

func (h *SecurityHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/security", requireAuth)
	g.GET("/trusted-ips", h.listTrusted)
	g.POST("/trusted-ips", h.addTrusted)
	g.DELETE("/trusted-ips/:cidr", h.deleteTrusted)
	g.GET("/locked-ips", h.listLocked)
	g.POST("/unlock", h.unlock)
}

// loadTrusted parses the persisted setting into a slice. Missing key →
// empty list. Malformed JSON → error (caller decides to 500 or reset).
func (h *SecurityHandler) loadTrusted() ([]trustedEntry, error) {
	var s model.Setting
	if err := h.DB.Where("key = ?", trustedIPsSettingKey).First(&s).Error; err != nil {
		return []trustedEntry{}, nil
	}
	if s.Value == "" {
		return []trustedEntry{}, nil
	}
	var entries []trustedEntry
	if err := json.Unmarshal([]byte(s.Value), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// saveTrusted persists the slice and re-parses + swaps the matcher used
// by LoginLockout / TOTPLockout. Errors at parse stage are propagated so
// the caller can return 400 — defends against persisting an invalid list
// that would silently fail to take effect.
func (h *SecurityHandler) saveTrusted(entries []trustedEntry) error {
	cidrs := make([]string, len(entries))
	for i, e := range entries {
		cidrs[i] = e.CIDR
	}
	matcher, err := security.ParseTrustedCIDRs(cidrs)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if err := h.DB.Save(&model.Setting{Key: trustedIPsSettingKey, Value: string(raw)}).Error; err != nil {
		return err
	}
	if h.LoginLockout != nil {
		h.LoginLockout.SetTrustedMatcher(matcher)
	}
	if h.TOTPLockout != nil {
		h.TOTPLockout.SetTrustedMatcher(matcher)
	}
	return nil
}

func (h *SecurityHandler) listTrusted(c *gin.Context) {
	entries, err := h.loadTrusted()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "load failed: "+err.Error())
		return
	}
	OK(c, gin.H{"items": entries})
}

type addTrustedRequest struct {
	CIDR string `json:"cidr" binding:"required"`
	Note string `json:"note"`
}

func (h *SecurityHandler) addTrusted(c *gin.Context) {
	var req addTrustedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	cidr := strings.TrimSpace(req.CIDR)
	if cidr == "" {
		Fail(c, http.StatusBadRequest, 400, "CIDR is required")
		return
	}
	// Validate the CIDR alone first so we can report a precise error.
	if _, err := security.ParseTrustedCIDRs([]string{cidr}); err != nil {
		Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	entries, err := h.loadTrusted()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "load failed")
		return
	}
	for _, e := range entries {
		if e.CIDR == cidr {
			Fail(c, http.StatusConflict, 409, "CIDR already in the trusted list")
			return
		}
	}
	entries = append(entries, trustedEntry{
		CIDR:    cidr,
		Note:    strings.TrimSpace(req.Note),
		AddedAt: time.Now(),
	})
	if err := h.saveTrusted(entries); err != nil {
		Fail(c, http.StatusInternalServerError, 500, "save failed: "+err.Error())
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "trusted_ip_add",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details:   map[string]any{"cidr": cidr, "note": req.Note},
	})
	OK(c, gin.H{"ok": true})
}

func (h *SecurityHandler) deleteTrusted(c *gin.Context) {
	target := c.Param("cidr")
	entries, err := h.loadTrusted()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "load failed")
		return
	}
	filtered := make([]trustedEntry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.CIDR == target {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		Fail(c, http.StatusNotFound, 404, "CIDR not in trusted list")
		return
	}
	if err := h.saveTrusted(filtered); err != nil {
		Fail(c, http.StatusInternalServerError, 500, "save failed: "+err.Error())
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "trusted_ip_remove",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details:   map[string]any{"cidr": target},
	})
	OK(c, gin.H{"ok": true})
}

// listLocked merges the in-memory snapshots from password + TOTP lockouts
// into a single dashboard view. Each entry is tagged with which lockout
// produced it so the unlock endpoint knows where to release.
type lockedIPItem struct {
	IP             string `json:"ip"`
	Source         string `json:"source"` // "login" or "totp"
	Failures       int    `json:"failures"`
	RemainingSec   int    `json:"remaining_seconds"`
	LockedUntilISO string `json:"locked_until"`
}

func (h *SecurityHandler) listLocked(c *gin.Context) {
	out := make([]lockedIPItem, 0)
	if h.LoginLockout != nil {
		for _, s := range h.LoginLockout.SnapshotLocked() {
			out = append(out, lockedIPItem{
				IP:             s.IP,
				Source:         "login",
				Failures:       s.Failures,
				RemainingSec:   int(s.Remaining.Seconds()),
				LockedUntilISO: s.LockedUntil.UTC().Format(time.RFC3339),
			})
		}
	}
	if h.TOTPLockout != nil {
		for _, s := range h.TOTPLockout.SnapshotLocked() {
			out = append(out, lockedIPItem{
				IP:             s.IP,
				Source:         "totp",
				Failures:       s.Failures,
				RemainingSec:   int(s.Remaining.Seconds()),
				LockedUntilISO: s.LockedUntil.UTC().Format(time.RFC3339),
			})
		}
	}
	OK(c, gin.H{"items": out})
}

type unlockRequest struct {
	IP     string `json:"ip" binding:"required"`
	Source string `json:"source" binding:"required"` // "login" or "totp"
}

func (h *SecurityHandler) unlock(c *gin.Context) {
	var req unlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	released := false
	switch req.Source {
	case "login":
		if h.LoginLockout != nil {
			released = h.LoginLockout.ManualUnlock(req.IP)
		}
	case "totp":
		if h.TOTPLockout != nil {
			released = h.TOTPLockout.ManualUnlock(req.IP)
		}
	default:
		Fail(c, http.StatusBadRequest, 400, "source must be 'login' or 'totp'")
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "manual_unlock",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details:   map[string]any{"target_ip": req.IP, "source": req.Source, "released": released},
	})
	OK(c, gin.H{"ok": true, "released": released})
}
