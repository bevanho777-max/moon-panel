package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moon-panel/moon-panel/internal/auth"
)

const (
	CookieName          = "moon_session"
	ChallengeCookieName = "moon_2fa_challenge"
	ContextClaimsKey    = "auth.claims"
	ContextUsernameKey  = "auth.username"
)

// sessionFloorUnix is the minimum acceptable IssuedAt for any session token.
// Tokens with iat < floor are rejected even if their signature and expiry are
// valid. Used to invalidate all existing sessions atomically (e.g. on the
// 3d-2 deploy that rotates default TTL semantics).
//
// atomic.Int64 lets us update the floor at runtime (future "logout all
// sessions" admin button) without restarting. Read on every request → must
// be lock-free; atomic load is one mov instruction on amd64/arm64.
var sessionFloorUnix atomic.Int64

// SetSessionFloor sets the cutoff. Call once at boot from bootstrap; can be
// called later (e.g. handler that lets admin force-logout all sessions).
func SetSessionFloor(unixSec int64) {
	sessionFloorUnix.Store(unixSec)
}

// GetSessionFloor returns the current cutoff (0 = no floor enforced).
func GetSessionFloor() int64 {
	return sessionFloorUnix.Load()
}

// RequireAuth aborts with 401 if no valid session cookie is present.
// On valid cookie, stashes claims in context and slides the cookie if it's
// older than the refresh threshold.
func RequireAuth(svc *auth.Service, cookieSecure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := claimsFromRequest(c, svc)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "unauthorized",
			})
			return
		}
		c.Set(ContextClaimsKey, claims)
		c.Set(ContextUsernameKey, claims.Username)

		if svc.ShouldRefresh(claims) {
			if newToken, exp, err := svc.IssueToken(claims.UserID, claims.Username); err == nil {
				SetSessionCookie(c, newToken, exp.Unix(), cookieSecure)
			}
		}
		c.Next()
	}
}

// OptionalAuth attaches claims if a valid cookie is present, but never aborts.
// Used by the public panel endpoint when PUBLIC_MODE=true.
func OptionalAuth(svc *auth.Service, cookieSecure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if claims, err := claimsFromRequest(c, svc); err == nil {
			c.Set(ContextClaimsKey, claims)
			c.Set(ContextUsernameKey, claims.Username)
			if svc.ShouldRefresh(claims) {
				if newToken, exp, err := svc.IssueToken(claims.UserID, claims.Username); err == nil {
					SetSessionCookie(c, newToken, exp.Unix(), cookieSecure)
				}
			}
		}
		c.Next()
	}
}

func claimsFromRequest(c *gin.Context, svc *auth.Service) (*auth.Claims, error) {
	cookie, err := c.Cookie(CookieName)
	if err != nil {
		return nil, err
	}
	claims, err := svc.ParseToken(cookie)
	if err != nil {
		return nil, err
	}
	// Session floor: token's iat must be >= floor. Floor=0 disables the check
	// (default for fresh installs that never set it).
	if floor := sessionFloorUnix.Load(); floor > 0 && claims.IssuedAt != nil {
		if claims.IssuedAt.Time.Unix() < floor {
			return nil, errSessionFloorViolated
		}
	}
	// Reject 2FA challenge tokens at admin endpoints. Challenge tokens are
	// only valid at /api/auth/2fa/verify, which uses ParseChallengeToken
	// directly rather than this middleware. A challenge token reaching
	// RequireAuth means an attacker is trying to use the half-completed
	// login as a full session — must fail.
	if claims.Stage == auth.StageAwaiting2FA {
		return nil, errChallengeStageOnAdmin
	}
	return claims, nil
}

var errChallengeStageOnAdmin = newAuthError("challenge token rejected on admin route")

var errSessionFloorViolated = newAuthError("session predates floor")

type authError struct{ msg string }

func newAuthError(m string) error               { return &authError{msg: m} }
func (e *authError) Error() string              { return e.msg }

func SetSessionCookie(c *gin.Context, token string, expUnix int64, secure bool) {
	maxAge := int(expUnix - time.Now().Unix())
	if maxAge < 0 {
		maxAge = 0
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetChallengeCookie writes the short-lived 2FA challenge token. Path is
// scoped to /api/auth/ so the cookie isn't sent on unrelated requests.
func SetChallengeCookie(c *gin.Context, token string, expUnix int64, secure bool) {
	maxAge := int(expUnix - time.Now().Unix())
	if maxAge < 0 {
		maxAge = 0
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     ChallengeCookieName,
		Value:    token,
		Path:     "/api/auth",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearChallengeCookie removes the 2FA challenge cookie. Called after
// successful TOTP verification (challenge consumed) and after explicit
// abort (e.g. the user closed the modal — no harm clearing).
func ClearChallengeCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     ChallengeCookieName,
		Value:    "",
		Path:     "/api/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

