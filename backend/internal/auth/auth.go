package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID   uint   `json:"uid"`
	Username string `json:"usr"`
	// Stage distinguishes full sessions ("" or "active") from the short-lived
	// 2FA challenge token issued between password OK and TOTP OK. The verify
	// endpoint checks Stage=="awaiting_2fa" to accept the challenge cookie;
	// admin endpoints reject anything other than "" / "active".
	Stage string `json:"stg,omitempty"`
}

// StageAwaiting2FA marks the short-lived cookie issued after password OK
// but before TOTP OK. Tokens with this stage are accepted ONLY by the
// /auth/2fa/verify endpoint.
const StageAwaiting2FA = "awaiting_2fa"

// ChallengeTTL is the lifespan of the 2FA challenge cookie. Short enough
// to bound exposure if the user walks away mid-login; long enough to type
// a 6-digit code from their phone (NIST recommends 90s for OTP).
const ChallengeTTL = 90 * time.Second

type Service struct {
	secret      []byte
	defaultTTL  time.Duration // used when caller doesn't specify "remember me"
	rememberTTL time.Duration // used when caller asks for long session
}

// New builds an auth service. defaultDays applies to standard logins;
// rememberDays applies when the login request flags "remember me". Both
// must be > 0; sensible defaults applied if not.
func New(secret []byte, defaultDays, rememberDays int) *Service {
	if defaultDays <= 0 {
		defaultDays = 7
	}
	if rememberDays <= 0 {
		rememberDays = 30
	}
	return &Service{
		secret:      secret,
		defaultTTL:  time.Duration(defaultDays) * 24 * time.Hour,
		rememberTTL: time.Duration(rememberDays) * 24 * time.Hour,
	}
}

// TTL returns the default (non-remember) session length.
func (s *Service) TTL() time.Duration { return s.defaultTTL }

// RememberTTL returns the "remember me" session length.
func (s *Service) RememberTTL() time.Duration { return s.rememberTTL }

func (s *Service) HashPassword(plain string) (string, error) {
	// Defense-in-depth: handler-level binding tags enforce min=8 at the API
	// boundary; this keeps the same minimum for any future internal callers
	// (CLI tools, tests, future bootstrap paths) so weak passwords can't slip
	// in through a non-handler code path.
	if len(plain) < 8 {
		return "", errors.New("password too short (min 8 characters)")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func (s *Service) VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// IssueToken issues a token with the default (non-remember) TTL. Kept for
// callers that don't differentiate (token refresh in middleware, password
// change re-issue) so they don't have to plumb a "remember me" flag.
func (s *Service) IssueToken(userID uint, username string) (string, time.Time, error) {
	return s.IssueTokenWithTTL(userID, username, s.defaultTTL)
}

// IssueTokenWithTTL issues a token with a caller-specified lifetime. Login
// uses this to honor "remember me".
func (s *Service) IssueTokenWithTTL(userID uint, username string, ttl time.Duration) (string, time.Time, error) {
	return s.issueTokenInternal(userID, username, "", ttl)
}

// IssueChallengeToken issues the short-lived 2FA-stage cookie. Stage is
// preset to "awaiting_2fa" and TTL is the global ChallengeTTL.
func (s *Service) IssueChallengeToken(userID uint, username string) (string, time.Time, error) {
	return s.issueTokenInternal(userID, username, StageAwaiting2FA, ChallengeTTL)
}

func (s *Service) issueTokenInternal(userID uint, username, stage string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   username,
		},
		UserID:   userID,
		Username: username,
		Stage:    stage,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func (s *Service) ParseToken(raw string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ShouldRefresh returns true if the token is more than refreshAfter old
// relative to its TTL. We refresh once a day to keep cookie age "rolling 30d"
// without doing a Set-Cookie on every request.
func (s *Service) ShouldRefresh(claims *Claims) bool {
	if claims.IssuedAt == nil {
		return false
	}
	age := time.Since(claims.IssuedAt.Time)
	return age >= 24*time.Hour
}
