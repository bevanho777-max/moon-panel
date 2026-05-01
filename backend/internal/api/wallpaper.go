package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WallpaperHandler manages the user-uploaded wallpaper library that lives
// alongside builtin SVGs (see internal/assets/wallpapers). Frontend stores a
// reference like "upload:public/wallpapers/<hash>.webp" or "builtin:night" in
// the ui.wallpaper setting; this handler is only concerned with the upload
// lifecycle.
type WallpaperHandler struct {
	DB      *gorm.DB
	DataDir string
}

// Wallpapers are larger than icons — 5 MiB cap per Phase 2.5c spec, which
// allows full-resolution photos pre-compression. Frontend canvas downscales
// to 1920×1080 webp ~85% quality before upload (typical < 250 KB), so this
// ceiling is mostly a defense-in-depth against malformed uploads, not the
// expected payload size.
const maxWallpaperBytes = 5 << 20 // 5 MiB

// allowedWallpaperTypes — png, jpeg, webp accepted. SVG remains excluded
// even for wallpapers (XSS via embedded scripts inside an <img src> tag is
// theoretical but cheap to avoid). Builtin SVGs ship inside the binary and
// bypass this path entirely.
var allowedWallpaperTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
}

const wallpaperSubdir = "public/wallpapers"

func (h *WallpaperHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/wallpapers", requireAuth)
	g.POST("/upload", h.upload)
	g.GET("", h.list)
	g.DELETE("/:hash", h.delete)
}

// upload accepts a multipart `file` field (already canvas-downscaled to
// 1920×1080 webp on the client side per Phase 2.5c spec), validates size +
// MIME, hashes content, and stores under /data/uploads/public/wallpapers/.
//
// Dedup: same content (by hash) returns the existing path with deduped:true,
// no duplicate write. Lets users re-pick the same wallpaper without bloating
// disk.
func (h *WallpaperHandler) upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "file field required")
		return
	}
	if file.Size <= 0 {
		Fail(c, http.StatusBadRequest, 400, "empty file")
		return
	}
	if file.Size > maxWallpaperBytes {
		Fail(c, http.StatusBadRequest, 400, fmt.Sprintf("file too large (%d bytes, max %d)", file.Size, maxWallpaperBytes))
		return
	}

	src, err := file.Open()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "open upload failed")
		return
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, maxWallpaperBytes+1))
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "read upload failed")
		return
	}
	if int64(len(data)) > maxWallpaperBytes {
		Fail(c, http.StatusBadRequest, 400, "file too large")
		return
	}

	result, serr := saveImageBytes(h.DataDir, wallpaperSubdir, allowedWallpaperTypes, data)
	if serr != nil {
		Fail(c, serr.status, serr.code, serr.msg)
		return
	}
	OK(c, gin.H{
		"path":      result.Path,
		"url":       "/uploads/" + result.Path,
		"wallpaper": "upload:" + result.Path,
		"deduped":   result.Deduped,
		"size":      len(data),
		"type":      result.Type,
	})
}

// list returns the catalog of uploaded wallpapers (filename + size + url).
// Builtin wallpapers are NOT included — frontend gets that list from the
// /api/public/panel response (assets.BuiltinWallpaperIDs). The split keeps
// "what's hardcoded vs what's user data" clear in the API surface.
func (h *WallpaperHandler) list(c *gin.Context) {
	dir := filepath.Join(h.DataDir, "uploads", filepath.FromSlash(wallpaperSubdir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Missing dir = no uploads yet. Return [] not 500.
		if os.IsNotExist(err) {
			OK(c, gin.H{"items": []any{}})
			return
		}
		Fail(c, http.StatusInternalServerError, 500, "read wallpaper dir")
		return
	}
	type item struct {
		Hash      string `json:"hash"`
		Path      string `json:"path"`
		URL       string `json:"url"`
		Wallpaper string `json:"wallpaper"`
		Size      int64  `json:"size"`
	}
	out := make([]item, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip tmp- prefixed atomic-write leftovers (saveImageBytes only renames
		// on success; orphans should be rare but possible after crashes).
		if strings.HasPrefix(name, "tmp-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		rel := wallpaperSubdir + "/" + name
		// Strip extension for the "hash" field — frontend uses it as React-style key.
		hash := name
		if dot := strings.LastIndex(name, "."); dot > 0 {
			hash = name[:dot]
		}
		out = append(out, item{
			Hash:      hash,
			Path:      rel,
			URL:       "/uploads/" + rel,
			Wallpaper: "upload:" + rel,
			Size:      info.Size(),
		})
	}
	OK(c, gin.H{"items": out})
}

// delete removes a single wallpaper file by hash. Frontend confirms before
// calling. NOTE: does NOT clear the ui.wallpaper setting if it currently
// references this file — that's the frontend's responsibility (it should
// either prompt the user or auto-fallback to builtin/null after delete).
// Server keeps the surface narrow: file ops only.
func (h *WallpaperHandler) delete(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" || strings.ContainsAny(hash, "/\\.") {
		// Reject "..", "/", "." — only bare hash + ext (e.g. "abc123.webp").
		// Restrict ext set: the file MUST exist, so a malformed name fails Stat.
		Fail(c, http.StatusBadRequest, 400, "invalid hash")
		return
	}
	dir := filepath.Join(h.DataDir, "uploads", filepath.FromSlash(wallpaperSubdir))
	// Walk dir to find the file with the requested hash prefix (the file on
	// disk has an extension; client only knows the hash sans ext).
	entries, err := os.ReadDir(dir)
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "read dir")
		return
	}
	for _, e := range entries {
		name := e.Name()
		base := name
		if dot := strings.LastIndex(name, "."); dot > 0 {
			base = name[:dot]
		}
		if base == hash {
			path := filepath.Join(dir, name)
			if err := os.Remove(path); err != nil {
				Fail(c, http.StatusInternalServerError, 500, "delete failed")
				return
			}
			OK(c, gin.H{"deleted": name})
			return
		}
	}
	Fail(c, http.StatusNotFound, 404, "not found")
}
