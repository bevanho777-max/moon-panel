package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Env            string // "production" → gin release mode; otherwise debug
	Port           string
	DataDir        string
	PublicMode     bool
	AdminPassword  string // optional bootstrap; only consulted if no admin exists
	JWTSecret      []byte
	// TokenTTLDays is the default session lifetime (no "remember me"). 7 days
	// is a balance for personal-panel public deployments: long enough that
	// users don't re-login multiple times a day, short enough that a stolen
	// cookie's blast radius is bounded. "Remember me" extends to 30 days.
	TokenTTLDays         int
	TokenRememberTTLDays int
	CookieSecure         bool
	CORSOrigins          []string // empty disables CORS entirely (default for prod same-origin)
	TrustedProxies       []string // CIDR list passed to gin.SetTrustedProxies

	// SSRF defense for "fetch URL" endpoints (icon library / future wallpaper).
	// See memory/feedback_ssrf_protection.md.
	AllowPrivateFetch bool
	AllowedFetchHosts []string
}

func Load() (*Config, error) {
	c := &Config{
		Env:            getEnv("MOON_ENV", "development"),
		Port:           getEnv("MOON_PORT", "3000"),
		DataDir:        getEnv("MOON_DATA_DIR", "./data"),
		PublicMode:     getEnvBool("MOON_PUBLIC_MODE", true),
		AdminPassword:        os.Getenv("MOON_ADMIN_PASSWORD"),
		TokenTTLDays:         getEnvInt("MOON_TOKEN_TTL_DAYS", 7),
		TokenRememberTTLDays: getEnvInt("MOON_TOKEN_REMEMBER_TTL_DAYS", 30),
		CookieSecure:         getEnvBool("MOON_COOKIE_SECURE", false),
		CORSOrigins:    splitCSV(os.Getenv("MOON_CORS_ORIGINS")),
		TrustedProxies: splitCSV(getEnv("MOON_TRUSTED_PROXIES", "127.0.0.1,172.16.0.0/12")),

		AllowPrivateFetch: getEnvBool("MOON_ALLOW_PRIVATE_FETCH", false),
		AllowedFetchHosts: splitCSV(os.Getenv("MOON_ALLOWED_FETCH_HOSTS")),
	}

	if err := os.MkdirAll(c.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	secret, err := loadOrCreateJWTSecret(c.DataDir)
	if err != nil {
		return nil, err
	}
	c.JWTSecret = secret

	return c, nil
}

// loadOrCreateJWTSecret prefers MOON_JWT_SECRET env, otherwise reads/generates
// data/jwt.key. Persisted across restarts so existing sessions survive reboot.
func loadOrCreateJWTSecret(dataDir string) ([]byte, error) {
	if env := os.Getenv("MOON_JWT_SECRET"); env != "" {
		return []byte(env), nil
	}
	path := filepath.Join(dataDir, "jwt.key")
	if data, err := os.ReadFile(path); err == nil && len(data) >= 32 {
		return data, nil
	}
	buf := make([]byte, 64)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate jwt secret: %w", err)
	}
	encoded := []byte(hex.EncodeToString(buf))
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return nil, fmt.Errorf("persist jwt secret: %w", err)
	}
	return encoded, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
