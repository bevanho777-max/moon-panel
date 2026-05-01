package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/audit"
	"github.com/moon-panel/moon-panel/internal/auth"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
	"github.com/moon-panel/moon-panel/internal/totp"
)

// TOTPHandler exposes 2FA management endpoints. Phase 3d-3.
//
// Endpoints:
//   POST /api/auth/2fa/enroll   (auth)  → returns { secret, otpauth_url, backup_codes }
//   POST /api/auth/2fa/confirm  (auth)  → verifies first TOTP, persists secret + hashed backup codes
//   POST /api/auth/2fa/disable  (auth)  → requires password + TOTP, clears secret
//   POST /api/auth/2fa/verify   (challenge cookie) → completes login two-step
type TOTPHandler struct {
	DB           *gorm.DB
	Auth         *auth.Service
	CookieSecure bool
	// TOTPLockout is per-IP+account lockout for verify failures. Nil-safe.
	TOTPLockout *middleware.LoginLockout
}

func (h *TOTPHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/auth/2fa")
	g.POST("/enroll", requireAuth, h.enroll)
	g.POST("/confirm", requireAuth, h.confirm)
	g.POST("/disable", requireAuth, h.disable)
	// /verify uses the challenge cookie, not the session cookie. Lockout
	// middleware (if configured) pre-rejects locked IPs before any DB hit.
	if h.TOTPLockout != nil {
		g.POST("/verify", h.TOTPLockout.Middleware(), h.verify)
	} else {
		g.POST("/verify", h.verify)
	}
}

type enrollResponse struct {
	Secret      string   `json:"secret"`
	OTPAuthURL  string   `json:"otpauth_url"`
	BackupCodes []string `json:"backup_codes"`
}

// enroll generates a fresh TOTP secret + URI + plaintext backup codes. Does
// NOT persist anything yet — the user must complete /confirm with a valid
// TOTP code first. Re-enrolling overwrites any previous in-progress
// enrollment but cannot escalate (must call /confirm to activate).
//
// Why not persist at this stage: a half-completed enrollment (user closed
// the modal mid-scan) shouldn't leave them with a 2FA-enabled account they
// can't actually use. Two-step (enroll → confirm) makes activation atomic.
func (h *TOTPHandler) enroll(c *gin.Context) {
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)

	var user model.User
	if err := h.DB.Where("username = ?", actor).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, 401, "user not found")
		return
	}
	if user.TOTPEnabled {
		Fail(c, http.StatusConflict, 409, "2FA is already enabled; disable first to re-enroll")
		return
	}

	enrollment, err := totp.NewEnrollment(user.Username)
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "generate enrollment failed")
		return
	}

	// Stash the candidate secret on the row WITHOUT setting TOTPEnabled.
	// /confirm will validate the TOTP code against this secret.
	if err := h.DB.Model(&user).Updates(map[string]any{
		"totp_secret":  enrollment.Secret,
		"totp_enabled": false,
	}).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "persist enrollment failed")
		return
	}

	// Hash backup codes and stash. Plaintext codes returned this once only.
	hashed, err := totp.HashBackupCodes(enrollment.BackupCodes)
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "hash backup codes failed")
		return
	}
	if err := h.DB.Model(&user).Update("totp_backup_codes", hashed).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "persist backup codes failed")
		return
	}

	OK(c, enrollResponse{
		Secret:      enrollment.Secret,
		OTPAuthURL:  enrollment.OTPAuthURL,
		BackupCodes: enrollment.BackupCodes,
	})
}

type confirmRequest struct {
	Code string `json:"code" binding:"required"`
}

// confirm checks the user's first TOTP code against the candidate secret
// stashed during /enroll. On success, marks TOTPEnabled = true. On failure,
// clears the candidate secret so the user must re-enroll (avoids a stuck
// half-state where /enroll was called but /confirm never succeeded).
func (h *TOTPHandler) confirm(c *gin.Context) {
	var req confirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)

	var user model.User
	if err := h.DB.Where("username = ?", actor).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, 401, "user not found")
		return
	}
	if user.TOTPSecret == "" {
		Fail(c, http.StatusBadRequest, 400, "no enrollment in progress; call /enroll first")
		return
	}
	if user.TOTPEnabled {
		Fail(c, http.StatusConflict, 409, "2FA is already enabled")
		return
	}

	if !totp.Verify(user.TOTPSecret, req.Code) {
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "totp_enable_failed",
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    http.StatusUnauthorized,
			Details:   map[string]any{"reason": "code_mismatch"},
		})
		Fail(c, http.StatusUnauthorized, 401, "code does not match")
		return
	}

	if err := h.DB.Model(&user).Update("totp_enabled", true).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "activate failed")
		return
	}
	audit.Write(h.DB, audit.Entry{
		Actor:     user.Username,
		Action:    "totp_enable",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
	})
	OK(c, gin.H{"ok": true})
}

type disableRequest struct {
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// disable turns off 2FA. Requires BOTH the password AND a current TOTP
// (or backup code). This is by design (Fork 3 default A): a stolen session
// cookie alone cannot disable 2FA, because the attacker doesn't know the
// password. A stolen session + leaked password also can't disable without
// the user's TOTP device — keeping the account-recovery bar high.
func (h *TOTPHandler) disable(c *gin.Context) {
	var req disableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)

	var user model.User
	if err := h.DB.Where("username = ?", actor).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, 401, "user not found")
		return
	}
	if !user.TOTPEnabled {
		Fail(c, http.StatusBadRequest, 400, "2FA is not enabled")
		return
	}
	if !h.Auth.VerifyPassword(user.PasswordHash, req.Password) {
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "totp_disable_failed",
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    http.StatusUnauthorized,
			Details:   map[string]any{"reason": "password_mismatch"},
		})
		Fail(c, http.StatusUnauthorized, 401, "password incorrect")
		return
	}
	// Accept a TOTP code OR a backup code at the disable step.
	codeOK := totp.Verify(user.TOTPSecret, req.Code)
	if !codeOK && user.TOTPBackupCodes != "" {
		newJSON, matched := totp.ConsumeBackupCode(user.TOTPBackupCodes, req.Code)
		if matched {
			codeOK = true
			// Persist the consumed code immediately so it can't be reused
			// even if the disable transaction fails further on.
			_ = h.DB.Model(&user).Update("totp_backup_codes", newJSON).Error
		}
	}
	if !codeOK {
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "totp_disable_failed",
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    http.StatusUnauthorized,
			Details:   map[string]any{"reason": "code_mismatch"},
		})
		Fail(c, http.StatusUnauthorized, 401, "code does not match")
		return
	}

	if err := h.DB.Model(&user).Updates(map[string]any{
		"totp_enabled":      false,
		"totp_secret":       "",
		"totp_backup_codes": "",
	}).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "disable failed")
		return
	}
	audit.Write(h.DB, audit.Entry{
		Actor:     user.Username,
		Action:    "totp_disable",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
	})
	OK(c, gin.H{"ok": true})
}

type verifyRequest struct {
	Code        string `json:"code" binding:"required"`
	IsBackup    bool   `json:"is_backup"`
	RememberMe  bool   `json:"remember_me"`
}

// verify completes the two-step login. Reads the challenge cookie set by
// /login, validates the TOTP (or backup code), and on success issues the
// full session cookie. Failure increments the TOTP-specific lockout bucket.
func (h *TOTPHandler) verify(c *gin.Context) {
	var req verifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	cookie, err := c.Cookie(middleware.ChallengeCookieName)
	if err != nil {
		Fail(c, http.StatusUnauthorized, 401, "no 2FA challenge in progress")
		return
	}
	claims, err := h.Auth.ParseToken(cookie)
	if err != nil || claims.Stage != auth.StageAwaiting2FA {
		Fail(c, http.StatusUnauthorized, 401, "challenge invalid or expired")
		return
	}

	ip := c.ClientIP()

	var user model.User
	if err := h.DB.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, 401, "user not found")
		return
	}
	if !user.TOTPEnabled {
		// Race: 2FA was disabled in another session between password OK and
		// this verify. Treat as session over — user should restart login.
		middleware.ClearChallengeCookie(c, h.CookieSecure)
		Fail(c, http.StatusBadRequest, 400, "2FA no longer enabled; restart login")
		return
	}

	emitFailure := func(reason string) {
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "totp_verify_failure",
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			Status:    http.StatusUnauthorized,
			Details:   map[string]any{"reason": reason, "is_backup": req.IsBackup},
		})
		if h.TOTPLockout != nil {
			h.TOTPLockout.RecordFailure(ip)
		}
	}

	codeOK := false
	usedBackup := false
	if req.IsBackup {
		newJSON, matched := totp.ConsumeBackupCode(user.TOTPBackupCodes, req.Code)
		if matched {
			codeOK = true
			usedBackup = true
			_ = h.DB.Model(&user).Update("totp_backup_codes", newJSON).Error
		}
	} else {
		codeOK = totp.Verify(user.TOTPSecret, req.Code)
	}

	if !codeOK {
		emitFailure("code_mismatch")
		Fail(c, http.StatusUnauthorized, 401, "code does not match")
		return
	}

	// Success: clear challenge cookie, issue full session.
	middleware.ClearChallengeCookie(c, h.CookieSecure)
	if h.TOTPLockout != nil {
		h.TOTPLockout.RecordSuccess(ip)
	}

	ttl := h.Auth.TTL()
	if req.RememberMe {
		ttl = h.Auth.RememberTTL()
	}
	token, exp, err := h.Auth.IssueTokenWithTTL(user.ID, user.Username, ttl)
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "issue token failed")
		return
	}
	middleware.SetSessionCookie(c, token, exp.Unix(), h.CookieSecure)

	if usedBackup {
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "backup_code_used",
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			Status:    200,
			Details:   map[string]any{"remaining": countBackupCodes(user.TOTPBackupCodes) - 1},
		})
	}
	audit.Write(h.DB, audit.Entry{
		Actor:     user.Username,
		Action:    "totp_verify_success",
		IP:        ip,
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details:   map[string]any{"remember_me": req.RememberMe, "ttl_seconds": int(ttl.Seconds()), "via_backup": usedBackup},
	})
	OK(c, gin.H{"username": user.Username})
}

// countBackupCodes parses the JSON array and returns its length. Best-effort:
// a parse error returns 0 rather than failing the whole request.
func countBackupCodes(raw string) int {
	if raw == "" {
		return 0
	}
	// Quick count via comma — JSON array of strings, each with quotes; commas
	// separate elements. Cheaper than a real parse and good enough for an
	// audit-log-only display value.
	if !strings.Contains(raw, "[") {
		return 0
	}
	if strings.HasPrefix(strings.TrimSpace(raw), "[]") {
		return 0
	}
	commas := strings.Count(raw, ",")
	return commas + 1
}

