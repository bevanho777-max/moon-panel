package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/audit"
	"github.com/moon-panel/moon-panel/internal/auth"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
)

type AuthHandler struct {
	DB           *gorm.DB
	Auth         *auth.Service
	CookieSecure bool
	// Lockout is optional. If non-nil, login wraps with brute-force protection:
	// pre-check at middleware level, RecordSuccess/RecordFailure in handler.
	Lockout *middleware.LoginLockout
}

func (h *AuthHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/auth")
	g.GET("/me", h.me)
	g.POST("/init", h.initAdmin)
	// Phase 4a: login no longer pre-rejects with 429 in middleware. Soft-unlock
	// requires the handler to see the password attempt before deciding whether
	// to subtract from a lock or extend it. Trusted IPs bypass entirely.
	g.POST("/login", h.login)
	g.POST("/logout", h.logout)
	g.PUT("/password", requireAuth, h.changePassword)
}

type meResponse struct {
	Initialized   bool   `json:"initialized"`
	Authenticated bool   `json:"authenticated"`
	Username      string `json:"username,omitempty"`
	TOTPEnabled   bool   `json:"totp_enabled"`
}

func (h *AuthHandler) me(c *gin.Context) {
	var count int64
	if err := h.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	resp := meResponse{Initialized: count > 0}

	if cookie, err := c.Cookie(middleware.CookieName); err == nil {
		if claims, err := h.Auth.ParseToken(cookie); err == nil && claims.Stage != auth.StageAwaiting2FA {
			resp.Authenticated = true
			resp.Username = claims.Username
			// Look up TOTP status so the admin UI can render the 2FA section
			// state (enabled vs disabled) without a separate request.
			var user model.User
			if err := h.DB.Select("totp_enabled").Where("id = ?", claims.UserID).First(&user).Error; err == nil {
				resp.TOTPEnabled = user.TOTPEnabled
			}
		}
	}
	OK(c, resp)
}

type initRequest struct {
	Password string `json:"password" binding:"required,min=8,max=128"`
}

func (h *AuthHandler) initAdmin(c *gin.Context) {
	var req initRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	var count int64
	if err := h.DB.Model(&model.User{}).Count(&count).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	if count > 0 {
		Fail(c, http.StatusForbidden, 403, "already initialized")
		return
	}
	hash, err := h.Auth.HashPassword(req.Password)
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	user := model.User{Username: "admin", PasswordHash: hash}
	if err := h.DB.Create(&user).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "create user failed")
		return
	}
	h.issueAndSetCookieWithTTL(c, user, h.Auth.TTL())
	audit.Write(h.DB, audit.Entry{
		Actor:     user.Username,
		Action:    "init_admin",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details:   map[string]any{"username": user.Username},
	})
	OK(c, gin.H{"username": user.Username})
}

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"`
}

func (h *AuthHandler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "admin"
	}
	ip := c.ClientIP()

	// Phase 4a: soft-lock-aware lockout flow.
	//   1. Trusted IP → bypass all lockout logic (still audit failures).
	//   2. Locked IP → process credentials anyway:
	//      - wrong: extend lock, return 429 + remaining
	//      - right: subtract softUnlockSubtract; if cleared → proceed,
	//        else return 429 + new remaining (real user can keep trying)
	//   3. Not locked, not trusted → existing flow.
	trusted := h.Lockout != nil && h.Lockout.IsTrusted(ip)
	wasLocked := false
	var lockRemaining time.Duration
	if h.Lockout != nil && !trusted {
		wasLocked, lockRemaining = h.Lockout.Status(ip)
	}

	emitFailure := func(reason string) {
		details := map[string]any{"username_tried": username, "reason": reason}
		if trusted {
			details["trusted_ip"] = true
		}
		if wasLocked {
			details["was_locked"] = true
		}
		audit.Write(h.DB, audit.Entry{
			Actor:     username,
			Action:    "login_failure",
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			Status:    http.StatusUnauthorized,
			Details:   details,
		})
		if h.Lockout != nil && !trusted {
			h.Lockout.RecordFailure(ip)
		}
	}

	var user model.User
	err := h.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			emitFailure("user_not_found")
			h.respondLoginFailure(c, wasLocked, trusted)
			return
		}
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	passwordOK := h.Auth.VerifyPassword(user.PasswordHash, req.Password)

	if !passwordOK {
		emitFailure("password_mismatch")
		h.respondLoginFailure(c, wasLocked, trusted)
		return
	}

	// Password OK. If locked (and not trusted), apply soft-unlock subtract.
	if wasLocked && h.Lockout != nil && !trusted {
		fullyUnlocked, newRemaining := h.Lockout.TrySoftUnlock(ip)
		if !fullyUnlocked {
			audit.Write(h.DB, audit.Entry{
				Actor:     user.Username,
				Action:    "login_soft_unlock_progress",
				IP:        ip,
				UserAgent: c.Request.UserAgent(),
				Status:    http.StatusTooManyRequests,
				Details: map[string]any{
					"remaining_seconds":     int(newRemaining.Seconds()),
					"original_lock_seconds": int(lockRemaining.Seconds()),
				},
			})
			c.Header("Retry-After", newRemaining.Truncate(time.Second).String())
			Fail(c, http.StatusTooManyRequests, 429,
				"密码正确但仍在软锁期内，再试 "+newRemaining.Truncate(time.Second).String()+" 后解除")
			return
		}
		// Fully unlocked — fall through to normal login flow below.
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "login_soft_unlock_cleared",
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			Status:    200,
		})
	} else if h.Lockout != nil && !trusted {
		h.Lockout.RecordSuccess(ip)
	}

	if user.TOTPEnabled {
		// Issue short-lived challenge cookie. Client follows up with /2fa/verify.
		token, exp, err := h.Auth.IssueChallengeToken(user.ID, user.Username)
		if err != nil {
			Fail(c, http.StatusInternalServerError, 500, "issue challenge failed")
			return
		}
		middleware.SetChallengeCookie(c, token, exp.Unix(), h.CookieSecure)
		audit.Write(h.DB, audit.Entry{
			Actor:     user.Username,
			Action:    "login_password_ok_awaiting_2fa",
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			Status:    200,
			Details:   map[string]any{"remember_me": req.RememberMe},
		})
		// Pass remember_me preference forward — the verify endpoint reads it
		// from the body so the user doesn't have to re-tick.
		OK(c, gin.H{"needs_2fa": true})
		return
	}

	// No 2FA — issue full session immediately (legacy single-step flow).
	ttl := h.Auth.TTL()
	if req.RememberMe {
		ttl = h.Auth.RememberTTL()
	}
	h.issueAndSetCookieWithTTL(c, user, ttl)
	audit.Write(h.DB, audit.Entry{
		Actor:     user.Username,
		Action:    "login_success",
		IP:        ip,
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details:   map[string]any{"remember_me": req.RememberMe, "ttl_seconds": int(ttl.Seconds())},
	})
	OK(c, gin.H{"username": user.Username, "needs_2fa": false})
}

// respondLoginFailure tailors the response based on lockout state. A locked
// IP gets a 429 (so the client knows to slow down); an unlocked IP gets the
// usual 401 (so it learns the credentials were wrong, but no lock yet).
// Trusted IPs always get 401 — no rate-limit signal because they're free
// to retry as fast as they want.
func (h *AuthHandler) respondLoginFailure(c *gin.Context, wasLocked, trusted bool) {
	if !wasLocked || trusted {
		Fail(c, http.StatusUnauthorized, 401, "invalid credentials")
		return
	}
	// Was locked AND just got another wrong attempt → already extended in
	// emitFailure. Report the new remaining.
	if h.Lockout != nil {
		_, remaining := h.Lockout.Status(c.ClientIP())
		c.Header("Retry-After", remaining.Truncate(time.Second).String())
		Fail(c, http.StatusTooManyRequests, 429,
			"too many failed login attempts; locked for "+remaining.Truncate(time.Second).String())
		return
	}
	Fail(c, http.StatusUnauthorized, 401, "invalid credentials")
}

func (h *AuthHandler) logout(c *gin.Context) {
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	middleware.ClearSessionCookie(c, h.CookieSecure)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "logout",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
	})
	OK(c, gin.H{"ok": true})
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

func (h *AuthHandler) changePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	var user model.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil {
		Fail(c, http.StatusUnauthorized, 401, "user not found")
		return
	}
	if !h.Auth.VerifyPassword(user.PasswordHash, req.OldPassword) {
		Fail(c, http.StatusUnauthorized, 401, "old password incorrect")
		return
	}
	hash, err := h.Auth.HashPassword(req.NewPassword)
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if err := h.DB.Model(&user).Update("password_hash", hash).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	// Public-deployment safety: changing the password invalidates the current
	// session so the user must re-authenticate. JWT can't be revoked server-side
	// without a denylist, so we settle for clearing the cookie — any leaked
	// token still works until its TTL, but the legitimate user is forced to
	// prove they know the new password.
	middleware.ClearSessionCookie(c, h.CookieSecure)
	audit.Write(h.DB, audit.Entry{
		Actor:     user.Username,
		Action:    "password_change",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
	})
	OK(c, gin.H{"ok": true})
}

func (h *AuthHandler) issueAndSetCookieWithTTL(c *gin.Context, user model.User, ttl time.Duration) {
	token, exp, err := h.Auth.IssueTokenWithTTL(user.ID, user.Username, ttl)
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "issue token failed")
		return
	}
	middleware.SetSessionCookie(c, token, exp.Unix(), h.CookieSecure)
}
