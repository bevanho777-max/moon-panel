package api

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/assets"
	"github.com/moon-panel/moon-panel/internal/audit"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
	"github.com/moon-panel/moon-panel/internal/store"
)

// BackupHandler exposes JSON metadata export + restore. Phase 4c.
//
// Format spec (BackupV1):
//   - version: "moon-panel-backup-v1"
//   - exported_at: RFC3339 timestamp
//   - groups, cards, search_engines: full row arrays
//   - settings: map of safe keys (auth.* are stripped on export)
//
// Sensitive data NOT included:
//   - users.password_hash / totp_secret / totp_backup_codes (auth credentials
//     never leave the host; that's a feature, not a limitation)
//   - audit_logs (privacy + size — typically the largest table)
//   - auth.session_floor (per-instance; restoring it would invalidate all
//     fresh sessions issued by the new container)
//   - jwt.key (lives on filesystem at /data/jwt.key, not in DB)
//
// Why not include uploads/* in the metadata JSON: file blobs would balloon
// the export. The optional zip variant (?include_uploads=true) bundles them
// alongside metadata.json for full-restore portability.
//
// Restore semantics: REPLACE — existing groups/cards/engines/settings are
// wiped and rebuilt from the backup. Audit_logs and users are preserved
// (you keep your password and 2FA after restore). Restore happens in one
// transaction for atomicity.
type BackupHandler struct {
	DB      *gorm.DB
	DataDir string // for resolving uploads/ on zip export and zip restore
}

const backupVersion = "moon-panel-backup-v1"

func (h *BackupHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/backup", requireAuth)
	g.GET("", h.exportJSON)
	g.GET("/zip", h.exportZip)
	g.POST("/restore", h.restore)
}

type backupV1 struct {
	Version       string                `json:"version"`
	ExportedAt    string                `json:"exported_at"`
	Groups        []model.Group         `json:"groups"`
	Cards         []model.Card          `json:"cards"`
	SearchEngines []model.SearchEngine  `json:"search_engines"`
	Settings      map[string]string     `json:"settings"`
}

// safeSettingKey returns true for settings keys safe to export. Auth-related
// keys are per-instance (session_floor) or sensitive, so we omit them.
func safeSettingKey(key string) bool {
	if strings.HasPrefix(key, "auth.") {
		return false
	}
	return true
}

func (h *BackupHandler) loadBackup() (*backupV1, error) {
	out := &backupV1{
		Version:    backupVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Settings:   make(map[string]string),
	}
	if err := h.DB.Order("sort ASC, id ASC").Find(&out.Groups).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Order("group_id ASC, sort ASC, id ASC").Find(&out.Cards).Error; err != nil {
		return nil, err
	}
	if err := h.DB.Order("sort ASC, id ASC").Find(&out.SearchEngines).Error; err != nil {
		return nil, err
	}
	var settings []model.Setting
	if err := h.DB.Find(&settings).Error; err != nil {
		return nil, err
	}
	for _, s := range settings {
		if safeSettingKey(s.Key) {
			out.Settings[s.Key] = s.Value
		}
	}
	return out, nil
}

func (h *BackupHandler) exportJSON(c *gin.Context) {
	bk, err := h.loadBackup()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "load backup: "+err.Error())
		return
	}
	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "backup_export_json",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details: map[string]any{
			"groups":   len(bk.Groups),
			"cards":    len(bk.Cards),
			"engines":  len(bk.SearchEngines),
			"settings": len(bk.Settings),
		},
	})
	filename := fmt.Sprintf("moon-panel-backup-%s.json", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.JSON(http.StatusOK, bk)
}

func (h *BackupHandler) exportZip(c *gin.Context) {
	bk, err := h.loadBackup()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "load backup: "+err.Error())
		return
	}
	bkBytes, err := json.MarshalIndent(bk, "", "  ")
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "marshal backup: "+err.Error())
		return
	}

	filename := fmt.Sprintf("moon-panel-backup-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Type", "application/zip")

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	// metadata.json
	mw, err := zw.Create("metadata.json")
	if err != nil {
		// Headers already sent; can't change status. Best we can do is log + abort.
		c.Abort()
		return
	}
	if _, err := mw.Write(bkBytes); err != nil {
		c.Abort()
		return
	}

	// uploads/* — walk the dir, add each file as zip entry. Errors are
	// logged but don't abort the whole export (a single corrupt file
	// shouldn't lose the whole backup).
	uploadsDir := filepath.Join(h.DataDir, "uploads")
	uploadCount := 0
	uploadBytes := int64(0)
	if info, err := os.Stat(uploadsDir); err == nil && info.IsDir() {
		err = filepath.Walk(uploadsDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil // skip unreadable files
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(uploadsDir, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()
			ew, err := zw.Create("uploads/" + rel)
			if err != nil {
				return nil
			}
			n, err := io.Copy(ew, f)
			if err != nil {
				return nil
			}
			uploadCount++
			uploadBytes += n
			return nil
		})
		if err != nil {
			// non-fatal; some uploads might be missing in zip
		}
	}

	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "backup_export_zip",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details: map[string]any{
			"groups":        len(bk.Groups),
			"cards":         len(bk.Cards),
			"engines":       len(bk.SearchEngines),
			"settings":      len(bk.Settings),
			"uploads_files": uploadCount,
			"uploads_bytes": uploadBytes,
		},
	})
}

// restore replaces all existing groups/cards/engines/settings with the
// backup contents in a single transaction. Accepts either:
//   - application/json body (BackupV1 shape)
//   - multipart form with file field "backup" containing a JSON file or zip
//
// Auth/audit data is preserved (users table untouched). On success returns
// counts of imported items; on failure rolls back entirely.
func (h *BackupHandler) restore(c *gin.Context) {
	bk, uploadsDirInZip, err := h.parseRestoreBody(c)
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}
	if bk.Version != backupVersion {
		Fail(c, http.StatusBadRequest, 400, "unsupported backup version: "+bk.Version)
		return
	}

	// One transaction: wipe + reinsert. ID values are preserved from the
	// backup so URLs/audit references remain stable. Cards have FK on
	// group_id — delete order: cards → groups → re-insert groups → cards.
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&model.Card{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&model.Group{}).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&model.SearchEngine{}).Error; err != nil {
			return err
		}
		// Settings: delete only safe keys (auth.* preserved so session_floor
		// etc. survive restore — fresh container must keep its own auth state).
		if err := tx.Where("key NOT LIKE ?", "auth.%").Delete(&model.Setting{}).Error; err != nil {
			return err
		}

		if len(bk.Groups) > 0 {
			if err := tx.Create(&bk.Groups).Error; err != nil {
				return err
			}
		}
		if len(bk.Cards) > 0 {
			if err := tx.Create(&bk.Cards).Error; err != nil {
				return err
			}
		}
		if len(bk.SearchEngines) > 0 {
			if err := tx.Create(&bk.SearchEngines).Error; err != nil {
				return err
			}
		}
		for k, v := range bk.Settings {
			if !safeSettingKey(k) {
				continue
			}
			if err := tx.Create(&model.Setting{Key: k, Value: v}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "restore failed: "+err.Error())
		return
	}

	// v0.2.28 R1: backups produced before v0.2.28 don't carry owner_id, so the
	// rows just re-inserted land with owner_id=0. The boot migration already
	// ran once and won't re-run on its own, so call it here to stamp the
	// freshly-restored data with the current admin owner. No-op for backups
	// produced on v0.2.28+.
	if err := store.MigrateOwnerID(h.DB); err != nil {
		// Don't fail the whole restore — the data is in, just unowned.
		// Admin sees a warning in the log and can investigate.
		log.Printf("backup restore: owner_id backfill failed (rows are restored but unowned): %v", err)
	}

	// Restore uploads/ if the zip path provided one. Done OUTSIDE the DB
	// transaction since file operations aren't transactional anyway.
	uploadsRestored := 0
	if uploadsDirInZip != "" {
		uploadsRestored, _ = h.restoreUploadsFromZipDir(uploadsDirInZip)
	}

	// Wallpaper orphan check: a JSON-only backup (no uploads/) restored on a
	// fresh instance can leave ui.wallpaper pointing at "upload:public/
	// wallpapers/<hash>.webp" with no actual file on disk. Detect that case
	// and fall back to null (= no wallpaper, default dark theme) so the home
	// page doesn't render a broken-image background. Builtin refs and null
	// pass through unchanged. Logged for visibility — admin sees they need
	// to re-upload or pick a builtin from SiteSettings.
	wallpaperFallback := h.fallbackOrphanWallpaper()

	username, _ := c.Get(middleware.ContextUsernameKey)
	actor, _ := username.(string)
	audit.Write(h.DB, audit.Entry{
		Actor:     actor,
		Action:    "backup_restore",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    200,
		Details: map[string]any{
			"groups":             len(bk.Groups),
			"cards":              len(bk.Cards),
			"engines":            len(bk.SearchEngines),
			"settings":           len(bk.Settings),
			"uploads_restored":   uploadsRestored,
			"wallpaper_fallback": wallpaperFallback,
		},
	})
	OK(c, gin.H{
		"groups":             len(bk.Groups),
		"cards":              len(bk.Cards),
		"engines":            len(bk.SearchEngines),
		"settings":           len(bk.Settings),
		"uploads_restored":   uploadsRestored,
		"wallpaper_fallback": wallpaperFallback,
	})
}

// fallbackOrphanWallpaper inspects the just-restored ui.wallpaper setting:
//   - "builtin:<id>" with a valid id → keep
//   - "builtin:<id>" with an unknown id → clear (binary version mismatch)
//   - "upload:<path>" pointing at an existing file → keep
//   - "upload:<path>" with no matching file on disk → clear + log
//   - empty/null/unrecognized → keep (no-op)
//
// "Clear" means deleting the row entirely so loadUISettings falls back to
// nil. Returns true when a fallback was applied (audit visibility).
func (h *BackupHandler) fallbackOrphanWallpaper() bool {
	var s model.Setting
	if err := h.DB.Where("key = ?", "ui.wallpaper").First(&s).Error; err != nil {
		return false
	}
	val := s.Value
	if val == "" {
		return false
	}
	switch {
	case strings.HasPrefix(val, "builtin:"):
		id := strings.TrimPrefix(val, "builtin:")
		if assets.IsValidBuiltinID(id) {
			return false
		}
		log.Printf("backup restore: ui.wallpaper=%q references unknown builtin id, clearing", val)
	case strings.HasPrefix(val, "upload:"):
		rel := strings.TrimPrefix(val, "upload:")
		// Defense: reject path-traversal-shaped refs even though saveImageBytes
		// only writes hashed names. Stat resolves to absolute path under DataDir.
		if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
			log.Printf("backup restore: ui.wallpaper=%q has suspicious path, clearing", val)
		} else {
			abs := filepath.Join(h.DataDir, "uploads", filepath.FromSlash(rel))
			if _, err := os.Stat(abs); err == nil {
				return false // file exists, keep
			}
			log.Printf("backup restore: ui.wallpaper=%q points to missing upload, clearing", val)
		}
	default:
		// Unknown prefix — leave alone. Future schema migrations might add
		// new prefixes; better to keep than guess.
		return false
	}
	if err := h.DB.Delete(&s).Error; err != nil {
		log.Printf("backup restore: failed to clear orphan ui.wallpaper: %v", err)
		return false
	}
	return true
}

// parseRestoreBody handles JSON body OR multipart upload with .json or .zip
// file. For zip, returns the temp-extracted uploads/ directory path so the
// caller can restore files after the DB transaction. For JSON, returns "".
func (h *BackupHandler) parseRestoreBody(c *gin.Context) (*backupV1, string, error) {
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		var bk backupV1
		if err := c.ShouldBindJSON(&bk); err != nil {
			return nil, "", errors.New("invalid JSON: " + err.Error())
		}
		return &bk, "", nil
	}
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return nil, "", errors.New("Content-Type must be application/json or multipart/form-data")
	}

	file, hdr, err := c.Request.FormFile("backup")
	if err != nil {
		return nil, "", errors.New("missing 'backup' file field")
	}
	defer file.Close()

	if strings.HasSuffix(strings.ToLower(hdr.Filename), ".json") {
		var bk backupV1
		if err := json.NewDecoder(file).Decode(&bk); err != nil {
			return nil, "", errors.New("invalid backup JSON: " + err.Error())
		}
		return &bk, "", nil
	}
	if !strings.HasSuffix(strings.ToLower(hdr.Filename), ".zip") {
		return nil, "", errors.New("file must be .json or .zip")
	}

	// Read zip into memory (capped at 50MiB — sane for personal panel).
	const maxZipBytes = 50 << 20
	buf, err := io.ReadAll(io.LimitReader(file, maxZipBytes))
	if err != nil {
		return nil, "", errors.New("read zip: " + err.Error())
	}
	if int64(len(buf)) >= maxZipBytes {
		return nil, "", errors.New("zip larger than 50MiB; refusing")
	}

	zr, err := zip.NewReader(strings.NewReader(string(buf)), int64(len(buf)))
	if err != nil {
		return nil, "", errors.New("not a valid zip: " + err.Error())
	}

	// Stage uploads to a temp dir; metadata.json into memory.
	tmpDir, err := os.MkdirTemp("", "moon-restore-*")
	if err != nil {
		return nil, "", errors.New("temp dir: " + err.Error())
	}
	uploadsTmp := filepath.Join(tmpDir, "uploads")
	if err := os.MkdirAll(uploadsTmp, 0o755); err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", errors.New("temp uploads dir: " + err.Error())
	}

	var bk *backupV1
	for _, f := range zr.File {
		name := f.Name
		if name == "metadata.json" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			body, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				continue
			}
			var parsed backupV1
			if err := json.Unmarshal(body, &parsed); err != nil {
				os.RemoveAll(tmpDir)
				return nil, "", errors.New("metadata.json invalid: " + err.Error())
			}
			bk = &parsed
			continue
		}
		if !strings.HasPrefix(name, "uploads/") || strings.HasSuffix(name, "/") {
			continue
		}
		// Path traversal guard: reject any "..", absolute paths, etc.
		clean := filepath.Clean(name)
		if strings.Contains(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		dest := filepath.Join(tmpDir, clean)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		w, err := os.Create(dest)
		if err != nil {
			rc.Close()
			continue
		}
		// 50MiB cap per file too — same defense against zip bombs.
		_, _ = io.Copy(w, io.LimitReader(rc, maxZipBytes))
		w.Close()
		rc.Close()
	}
	if bk == nil {
		os.RemoveAll(tmpDir)
		return nil, "", errors.New("zip missing metadata.json")
	}
	return bk, uploadsTmp, nil
}

// restoreUploadsFromZipDir copies files from the staging tmp dir into the
// real uploads dir. Existing files are overwritten — caller already
// confirmed REPLACE semantics. Returns count of restored files.
func (h *BackupHandler) restoreUploadsFromZipDir(stageDir string) (int, error) {
	defer os.RemoveAll(filepath.Dir(stageDir)) // remove the parent tmp dir too

	dest := filepath.Join(h.DataDir, "uploads")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return 0, err
	}

	count := 0
	walkErr := filepath.Walk(stageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(stageDir, path)
		if err != nil {
			return nil
		}
		target := filepath.Join(dest, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil
		}
		src, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer src.Close()
		out, err := os.Create(target)
		if err != nil {
			return nil
		}
		defer out.Close()
		if _, err := io.Copy(out, src); err == nil {
			count++
		}
		return nil
	})
	return count, walkErr
}
