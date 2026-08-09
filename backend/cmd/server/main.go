package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/api"
	"github.com/moon-panel/moon-panel/internal/assets"
	"github.com/moon-panel/moon-panel/internal/audit"
	"github.com/moon-panel/moon-panel/internal/auth"
	"github.com/moon-panel/moon-panel/internal/config"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
	"github.com/moon-panel/moon-panel/internal/security"
	"github.com/moon-panel/moon-panel/internal/store"
	"github.com/moon-panel/moon-panel/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := store.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	authSvc := auth.New(cfg.JWTSecret, cfg.TokenTTLDays, cfg.TokenRememberTTLDays)

	if cfg.AdminPassword != "" {
		if err := bootstrapAdmin(db, authSvc, cfg.AdminPassword); err != nil {
			log.Fatalf("bootstrap admin: %v", err)
		}
	}

	if err := bootstrapDefaultEngines(db); err != nil {
		log.Fatalf("bootstrap engines: %v", err)
	}
	if err := migrateBootstrapIconURLs(db); err != nil {
		log.Fatalf("migrate icons: %v", err)
	}
	if _, err := store.MigrateEngineCategories(db); err != nil {
		log.Fatalf("migrate engine categories: %v", err)
	}
	// v0.2.28 (A.5 R1): backfill owner_id on existing Group/Card rows. Runs
	// after bootstrapAdmin so the admin user is guaranteed to exist when we
	// look it up by username. Idempotent — re-runs on every boot are no-ops
	// once the rows are stamped.
	if err := store.MigrateOwnerID(db); err != nil {
		log.Fatalf("migrate owner ids: %v", err)
	}
	if err := bootstrapWidgetSettings(db); err != nil {
		log.Fatalf("bootstrap widgets: %v", err)
	}
	if err := bootstrapNetworkSettings(db); err != nil {
		log.Fatalf("bootstrap network: %v", err)
	}
	if err := bootstrapSessionFloor(db); err != nil {
		log.Fatalf("bootstrap session floor: %v", err)
	}
	if err := bootstrapTrustedIPs(db); err != nil {
		log.Fatalf("bootstrap trusted ips: %v", err)
	}
	// One-shot audit log retention sweep on startup. Subsequent cleanups run
	// opportunistically inside audit.Write (1/100 odds per insert).
	if deleted := audit.Cleanup(db); deleted > 0 {
		log.Printf("audit retention: pruned %d old entries on startup", deleted)
	}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("trusted proxies: %v", err)
	}
	if len(cfg.CORSOrigins) > 0 {
		r.Use(corsMiddleware(cfg.CORSOrigins))
	}

	requireAuth := middleware.RequireAuth(authSvc, cfg.CookieSecure)

	// Login lockout: 5 failures within 15 min → IP locked for 30 min.
	// Numbers chosen to deter brute-force enumeration on public deployments
	// while leaving room for legitimate-user typos. State is in-memory; a
	// container restart resets all locks (acceptable trade-off — see
	// middleware/login_lockout.go for rationale).
	loginLockout := middleware.NewLoginLockout(5, 15*time.Minute, 30*time.Minute)
	// Phase 4a: load persisted trusted-IP allowlist into the lockout instances.
	// Falls back to empty matcher on parse error — broken setting shouldn't
	// crash boot; admin can re-edit via UI.
	if matcher := loadTrustedMatcher(db); matcher != nil {
		loginLockout.SetTrustedMatcher(matcher)
	}
	// TOTP lockout: 7 failures within 10 min → IP locked for 15 min.
	// Looser than password lockout because TOTP code drift / fat-finger
	// happen more often (small font, time skew, scroll-past-window). 6-digit
	// brute-force has ~1M space per code; 7 attempts still gives a 0.0007%
	// chance per cycle, which is negligible. See Phase 4 backlog for the
	// trusted_ips whitelist that will let home networks bypass this entirely.
	totpLockout := middleware.NewLoginLockout(7, 10*time.Minute, 15*time.Minute)

	apiGroup := r.Group("/api")
	// Audit middleware: records admin mutations to the audit_logs table. Path
	// scoping (only /api/admin/*) is inside the middleware itself so non-admin
	// requests pass through cheaply.
	apiGroup.Use(middleware.AuditLog(db))

	api.RegisterHealth(apiGroup)
	api.RegisterVersion(apiGroup)
	(&api.SiteStatsHandler{DB: db, StartedAt: time.Now()}).Register(apiGroup)
	(&api.AuthHandler{DB: db, Auth: authSvc, CookieSecure: cfg.CookieSecure, Lockout: loginLockout}).Register(apiGroup, requireAuth)
	(&api.TOTPHandler{DB: db, Auth: authSvc, CookieSecure: cfg.CookieSecure, TOTPLockout: totpLockout}).Register(apiGroup, requireAuth)
	// Apply same trusted-IP matcher to TOTP lockout so home networks bypass
	// both password and 2FA lockouts.
	if matcher := loadTrustedMatcher(db); matcher != nil {
		totpLockout.SetTrustedMatcher(matcher)
	}
	(&api.SecurityHandler{DB: db, LoginLockout: loginLockout, TOTPLockout: totpLockout}).Register(apiGroup, requireAuth)
	(&api.BackupHandler{DB: db, DataDir: cfg.DataDir}).Register(apiGroup, requireAuth)
	(&api.PublicHandler{DB: db, PublicMode: cfg.PublicMode}).Register(apiGroup, requireAuth)
	(&api.GroupHandler{DB: db}).Register(apiGroup, requireAuth)
	(&api.CardHandler{DB: db}).Register(apiGroup, requireAuth)
	(&api.IconHandler{
		DataDir: cfg.DataDir,
		SSRFConfig: security.Config{
			AllowPrivate: cfg.AllowPrivateFetch,
			AllowedHosts: cfg.AllowedFetchHosts,
		},
	}).Register(apiGroup, requireAuth)
	(&api.WallpaperHandler{DB: db, DataDir: cfg.DataDir}).Register(apiGroup, requireAuth)
	(&api.SearchEngineHandler{DB: db}).Register(apiGroup, requireAuth)
	(&api.SettingHandler{DB: db}).Register(apiGroup, requireAuth)
	(&api.AuditLogHandler{DB: db}).Register(apiGroup, requireAuth)
	(&api.StatsHandler{DB: db}).Register(apiGroup, requireAuth)

	// Weather endpoint is public (homepage widget needs it without login),
	// rate-limited per IP to prevent amplification attacks against Open-Meteo.
	weatherLimit := middleware.NewIPRateLimit(10, time.Minute)
	weatherHandler := &api.WeatherHandler{}
	apiGroup.GET("/public/weather", weatherLimit.Middleware(), weatherHandler.GetHandler())

	// Static file serving for uploaded assets. Public (no auth) — icons render
	// on the public homepage. Future Phase can gate /uploads/private/*.
	//
	// IMPORTANT: r.Static() can NOT be used here. Gin's createStaticHandler
	// routes missing files to NoRoute handlers (the SPA fallback), which would
	// return index.html with Content-Type: text/html for any missing icon URL.
	// We need a clean 404 so <img onerror> can fire correctly.
	// See memory/feedback_gin_static_route_order.md.
	uploadsDir := filepath.Join(cfg.DataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		log.Fatalf("create uploads dir: %v", err)
	}
	uploadsHandler := staticUploadsHandler(uploadsDir)
	r.GET("/uploads/*filepath", uploadsHandler)
	r.HEAD("/uploads/*filepath", uploadsHandler)

	// Builtin wallpaper SVGs — embedded in the binary, no /data dependency.
	// Public (no auth): public Home page must load wallpaper before login.
	// Long Cache-Control: contents are immutable across binary versions
	// (file changes only ship via new release), so a 1-year cache is safe.
	wallpaperFS, err := assets.WallpaperFS()
	if err != nil {
		log.Fatalf("wallpaper embed: %v", err)
	}
	r.GET("/assets/wallpapers/:name", builtinWallpaperHandler(wallpaperFS))
	r.HEAD("/assets/wallpapers/:name", builtinWallpaperHandler(wallpaperFS))

	// v0.2.1: theme preview thumbnails (admin theme picker). Same handler
	// pattern as wallpapers — strict ID validation + long Cache-Control.
	themeFS, err := assets.ThemeFS()
	if err != nil {
		log.Fatalf("theme embed: %v", err)
	}
	r.GET("/assets/themes/:name", themePreviewHandler(themeFS))
	r.HEAD("/assets/themes/:name", themePreviewHandler(themeFS))

	fsys, err := web.Sub()
	if err != nil {
		log.Fatalf("embed fs: %v", err)
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "api not found"})
			return
		}
		web.SPAHandler(fsys)(c)
	})

	addr := ":" + cfg.Port
	log.Printf("moon-panel listening on %s (env=%s public_mode=%v cors=%v trusted_proxies=%v)",
		addr, cfg.Env, cfg.PublicMode, cfg.CORSOrigins, cfg.TrustedProxies)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// builtinWallpaperHandler serves embedded wallpaper SVGs at
// /assets/wallpapers/:name. Strict: only IDs in assets.BuiltinWallpaperIDs()
// resolve; anything else → 404 (defends against directory enumeration even
// though embed.FS doesn't allow path traversal).
func builtinWallpaperHandler(fsys fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		// Strip ".svg" extension if present so callers can use either form.
		id := strings.TrimSuffix(name, ".svg")
		if !assets.IsValidBuiltinID(id) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}
		data, err := fs.ReadFile(fsys, id+".svg")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}
		c.Header("Content-Type", "image/svg+xml")
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(http.StatusOK, "image/svg+xml", data)
	}
}

// themePreviewHandler serves embedded theme preview SVGs at
// /assets/themes/:name. v0.2.1: only "moon" / "risen" resolve. URL convention
// is `/assets/themes/moon-preview.svg` — handler accepts the bare id
// ("moon", "risen") or the full filename ("moon-preview.svg") for caller
// flexibility. Same long-cache headers as wallpapers.
func themePreviewHandler(fsys fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		// Accept both "moon" and "moon-preview.svg" forms.
		stem := strings.TrimSuffix(name, ".svg")
		stem = strings.TrimSuffix(stem, "-preview")
		if !assets.IsValidThemeID(stem) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}
		data, err := fs.ReadFile(fsys, stem+"-preview.svg")
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}
		c.Header("Content-Type", "image/svg+xml")
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(http.StatusOK, "image/svg+xml", data)
	}
}

// staticUploadsHandler serves files under uploadsDir at /uploads/*filepath
// with explicit 404 on missing files (NOT SPA fallback). Defends against
// path traversal by rejecting ".." segments and verifying the resolved path
// stays inside uploadsDir.
func staticUploadsHandler(uploadsDir string) gin.HandlerFunc {
	absRoot, err := filepath.Abs(uploadsDir)
	if err != nil {
		log.Fatalf("resolve uploads dir: %v", err)
	}
	return func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		if rel == "" || strings.Contains(rel, "..") || filepath.IsAbs(rel) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid path"})
			return
		}
		absPath := filepath.Join(absRoot, filepath.FromSlash(rel))
		// Defense in depth: ensure the resolved path is rooted under absRoot.
		if absPath != absRoot && !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid path"})
			return
		}
		info, err := os.Stat(absPath)
		if err != nil || info.IsDir() {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}
		c.File(absPath)
	}
}

// bootstrapDefaultEngines seeds the builtin search engines on first start (when
// the search_engines table is empty). Idempotent — once seeded, subsequent
// starts are no-op even if the user has deleted some/all engines (we only check
// "is the table empty?", not "are these specific entries present?"). The
// admin-triggered POST /admin/search-engines/restore-builtins covers re-adding
// individually deleted ones.
//
// v0.2.29: the list itself lives in api.BuiltinSearchEngines() so seeding and
// restore share one source of truth. Icons use jsdelivr CDN
// (walkxcode/dashboard-icons) — direct upstream favicons (google.com /
// bing.com / etc) aren't reliably reachable from mainland China. See
// migrateBootstrapIconURLs for the one-time migration that updates existing
// 3b-1-era deployments.
func bootstrapDefaultEngines(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.SearchEngine{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	engines := api.BuiltinSearchEngines()
	if err := db.Create(&engines).Error; err != nil {
		return err
	}
	log.Printf("bootstrapped %d default search engines", len(engines))
	return nil
}

// bootstrapWidgetSettings seeds widget.cities (default 3) and widget.temp_unit
// (default °C) on first start. Idempotent — only inserts a key if it doesn't
// already exist (so user changes on subsequent restarts aren't reverted).
func bootstrapWidgetSettings(db *gorm.DB) error {
	defaults := []model.Setting{
		{Key: "widget.cities", Value: `[{"name_cn":"北京","name_en":"Beijing","tz":"Asia/Shanghai","lat":39.9042,"lon":116.4074},{"name_cn":"纽约","name_en":"New York","tz":"America/New_York","lat":40.7128,"lon":-74.006},{"name_cn":"东京","name_en":"Tokyo","tz":"Asia/Tokyo","lat":35.6762,"lon":139.6503}]`},
		{Key: "widget.temp_unit", Value: "C"},
	}
	for _, s := range defaults {
		var existing model.Setting
		err := db.Where("key = ?", s.Key).First(&existing).Error
		if err == nil {
			continue // already set, leave alone
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&s).Error; err != nil {
			return err
		}
		log.Printf("bootstrapped setting %s", s.Key)
	}
	return nil
}

// bootstrapNetworkSettings seeds network.probe_url (default empty string) on
// first start. v0.2.23: empty means "auto-sample from card internal URLs";
// an explicit value enables LAN/WAN detection against a user-known endpoint.
// Idempotent — same pattern as bootstrapWidgetSettings.
func bootstrapNetworkSettings(db *gorm.DB) error {
	defaults := []model.Setting{
		{Key: "network.probe_url", Value: ""},
	}
	for _, s := range defaults {
		var existing model.Setting
		err := db.Where("key = ?", s.Key).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&s).Error; err != nil {
			return err
		}
		log.Printf("bootstrapped setting %s", s.Key)
	}
	return nil
}

// migrateBootstrapIconURLs is a one-time idempotent migration: replaces the
// 3b-1-era direct favicon URLs (google.com/favicon.ico etc) with jsdelivr CDN
// versions. Targeted WHERE icon = '<exact-old-URL>' so:
//   - First start after upgrade: 4 rows updated, log emitted
//   - Subsequent starts: 0 rows match (already migrated), log silent
//   - Users who manually edited an icon: untouched (their custom icon doesn't
//     match the old URL exactly)
//
// New deployments don't trigger this — bootstrapDefaultEngines already inserts
// jsdelivr URLs.
func migrateBootstrapIconURLs(db *gorm.DB) error {
	const cdnPrefix = "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/"
	updates := []struct{ oldIcon, newIcon string }{
		{"https://www.google.com/favicon.ico", cdnPrefix + "google.png"},
		{"https://www.bing.com/favicon.ico", cdnPrefix + "bing.png"},
		{"https://duckduckgo.com/favicon.ico", cdnPrefix + "duckduckgo.png"},
		{"https://www.baidu.com/favicon.ico", cdnPrefix + "baidu.png"},
	}
	var total int64
	for _, u := range updates {
		res := db.Model(&model.SearchEngine{}).Where("icon = ?", u.oldIcon).Update("icon", u.newIcon)
		if res.Error != nil {
			return res.Error
		}
		total += res.RowsAffected
	}
	if total > 0 {
		log.Printf("migrated %d bootstrap search engine icon URLs to jsdelivr CDN (one-time)", total)
	}
	return nil
}

// bootstrapSessionFloor sets a global "minimum issued-at" cutoff for session
// tokens. On first boot of the 3d-2 build (no existing setting), writes
// floor = NOW(). All sessions issued before this moment fail validation —
// the deploy-time effect: every existing user's cookie is invalidated and
// they must re-login once.
//
// Idempotent: only writes when the setting key is missing. Future boots read
// the persisted floor and apply it to the in-memory atomic. Future "logout
// all sessions" admin endpoint can update this setting to NOW() to force
// a global re-login without restarting the container.
func bootstrapSessionFloor(db *gorm.DB) error {
	const key = "auth.session_floor"
	var s model.Setting
	err := db.Where("key = ?", key).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// First boot: set floor to now → invalidates pre-3d-2 sessions.
		now := time.Now().Unix()
		s = model.Setting{Key: key, Value: strconv.FormatInt(now, 10)}
		if err := db.Create(&s).Error; err != nil {
			return err
		}
		middleware.SetSessionFloor(now)
		log.Printf("session floor initialized to %d (existing sessions invalidated)", now)
		return nil
	}
	if err != nil {
		return err
	}
	floor, parseErr := strconv.ParseInt(s.Value, 10, 64)
	if parseErr != nil {
		return parseErr
	}
	middleware.SetSessionFloor(floor)
	return nil
}

// bootstrapTrustedIPs ensures the security.trusted_ips key exists with
// an empty array as default. Admin populates it later via UI. Idempotent —
// only writes when key is missing, never overwrites user changes.
func bootstrapTrustedIPs(db *gorm.DB) error {
	const key = "security.trusted_ips"
	var s model.Setting
	err := db.Where("key = ?", key).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&model.Setting{Key: key, Value: "[]"}).Error
	}
	return err
}

// loadTrustedMatcher reads security.trusted_ips and returns a parsed matcher.
// Returns nil on missing key, malformed JSON, or invalid CIDR — callers should
// treat nil as "no trusted IPs configured" (safe default). Errors are logged
// for visibility but not fatal: a corrupt setting shouldn't prevent boot,
// admin can fix via UI after login.
func loadTrustedMatcher(db *gorm.DB) *security.TrustedIPMatcher {
	var s model.Setting
	if err := db.Where("key = ?", "security.trusted_ips").First(&s).Error; err != nil {
		return nil
	}
	type entry struct {
		CIDR string `json:"cidr"`
	}
	var entries []entry
	if err := json.Unmarshal([]byte(s.Value), &entries); err != nil {
		log.Printf("trusted_ips: malformed setting JSON, treating as empty: %v", err)
		return nil
	}
	if len(entries) == 0 {
		return nil
	}
	cidrs := make([]string, 0, len(entries))
	for _, e := range entries {
		cidrs = append(cidrs, e.CIDR)
	}
	matcher, err := security.ParseTrustedCIDRs(cidrs)
	if err != nil {
		log.Printf("trusted_ips: invalid CIDR in setting, treating as empty: %v", err)
		return nil
	}
	if matcher.Size() > 0 {
		log.Printf("trusted_ips: loaded %d entries", matcher.Size())
	}
	return matcher
}

// bootstrapAdmin creates the admin user from MOON_ADMIN_PASSWORD if no user
// exists yet. If a user already exists, the env var is silently ignored.
func bootstrapAdmin(db *gorm.DB, svc *auth.Service, password string) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := svc.HashPassword(password)
	if err != nil {
		return err
	}
	user := model.User{Username: "admin", PasswordHash: hash}
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	log.Printf("bootstrapped admin user from MOON_ADMIN_PASSWORD")
	return nil
}

// corsMiddleware echoes the request Origin if it's in the allowlist.
// Credentials (cookies) require an exact origin match — wildcards are rejected.
func corsMiddleware(allowed []string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allowedSet[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowedSet[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

