package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/moon-panel/moon-panel/internal/security"
)

const (
	maxIconBytes = 1 << 20 // 1 MiB
	hashLen      = 16      // SHA-256 truncated to 16 hex chars (64-bit space, collisions negligible at our scale)

	// Subdir under <DataDir>/uploads/ for icons. Wallpapers use a sibling subdir
	// (see wallpaper.go). Public so /uploads/public/icons/<hash>.<ext> resolves
	// without auth — homepage cards render before login.
	iconSubdir = "public/icons"
)

// allowedContentTypes maps sniffed Content-Type → file extension. SVG is
// intentionally excluded — it can embed <script> and is an XSS vector unless
// sanitized. Defer SVG support to Phase 3 with proper sanitization.
var allowedContentTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type IconHandler struct {
	DataDir    string
	SSRFConfig security.Config
}

func (h *IconHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/icons", requireAuth)
	g.POST("/upload", h.upload)
	g.POST("/fetch", h.fetch)
}

// upload accepts multipart `file` field, validates size + content-type,
// SHA-256 hashes content, stores at /data/uploads/public/icons/<hash>.<ext>,
// and returns the upload reference plus public URL.
//
// Dedup: if the same content (by hash) was uploaded before, the existing
// file is reused — no duplicate write, response includes "deduped: true".
func (h *IconHandler) upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "file field required")
		return
	}
	if file.Size <= 0 {
		Fail(c, http.StatusBadRequest, 400, "empty file")
		return
	}
	if file.Size > maxIconBytes {
		Fail(c, http.StatusBadRequest, 400, fmt.Sprintf("file too large (%d bytes, max %d)", file.Size, maxIconBytes))
		return
	}

	src, err := file.Open()
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "open upload failed")
		return
	}
	defer src.Close()

	// Read full content (bounded by maxIconBytes+1 to detect truncation attacks
	// claiming a smaller Size than actual content).
	data, err := io.ReadAll(io.LimitReader(src, maxIconBytes+1))
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "read upload failed")
		return
	}
	if int64(len(data)) > maxIconBytes {
		Fail(c, http.StatusBadRequest, 400, "file too large")
		return
	}

	result, serr := saveImageBytes(h.DataDir, iconSubdir, allowedContentTypes, data)
	if serr != nil {
		Fail(c, serr.status, serr.code, serr.msg)
		return
	}
	OK(c, gin.H{
		"path":    result.Path,
		"url":     "/uploads/" + result.Path,
		"icon":    "upload:" + result.Path,
		"deduped": result.Deduped,
		"size":    len(data),
		"type":    result.Type,
	})
}

type fetchRequest struct {
	URL string `json:"url"`
}

// fetch downloads an external image URL and stores it locally with the same
// hash/dedup pipeline as upload. SSRF-safe (see internal/security/ssrf.go).
//
// On success the response payload is identical shape to /upload, plus
// "source_url" so the client knows the original URL was processed.
func (h *IconHandler) fetch(c *gin.Context) {
	var req fetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	if req.URL == "" {
		Fail(c, http.StatusBadRequest, 400, "url required")
		return
	}

	target, err := security.ValidateURL(req.URL, h.SSRFConfig)
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, err.Error())
		return
	}

	client := security.BuildSafeClient(target, 10*time.Second)
	httpReq, err := http.NewRequest("GET", target.URL, nil)
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "build request failed")
		return
	}
	httpReq.Header.Set("User-Agent", "moon-panel/icon-fetch")
	httpReq.Header.Set("Accept", "image/*")

	resp, err := client.Do(httpReq)
	if err != nil {
		Fail(c, http.StatusBadGateway, 502, fmt.Sprintf("fetch failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		Fail(c, http.StatusBadGateway, 502, "URL returned a redirect — please paste the final URL after following redirects")
		return
	}
	if resp.StatusCode != http.StatusOK {
		Fail(c, http.StatusBadGateway, 502, fmt.Sprintf("URL returned HTTP %d", resp.StatusCode))
		return
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIconBytes+1))
	if err != nil {
		Fail(c, http.StatusBadGateway, 502, fmt.Sprintf("read response failed: %v", err))
		return
	}
	if int64(len(data)) > maxIconBytes {
		Fail(c, http.StatusBadRequest, 400, fmt.Sprintf("remote file too large (max %d bytes)", maxIconBytes))
		return
	}
	if len(data) == 0 {
		Fail(c, http.StatusBadGateway, 502, "remote returned empty body")
		return
	}

	result, ferr := saveImageBytes(h.DataDir, iconSubdir, allowedContentTypes, data)
	if ferr != nil {
		Fail(c, ferr.status, ferr.code, ferr.msg)
		return
	}
	OK(c, gin.H{
		"path":       result.Path,
		"url":        "/uploads/" + result.Path,
		"icon":       "upload:" + result.Path,
		"deduped":    result.Deduped,
		"size":       len(data),
		"type":       result.Type,
		"source_url": req.URL,
	})
}

// saveResult is the shared output of upload and fetch. Path is the relative
// /uploads/* path component (no leading slash).
type saveResult struct {
	Path    string
	Deduped bool
	Type    string
}

type saveError struct {
	status int
	code   int
	msg    string
}

// saveImageBytes sniffs content-type, validates extension, computes hash,
// dedups against existing file, atomically writes if new. Shared by icon
// upload/fetch and wallpaper upload — caller passes the relative subdir
// (e.g. "public/icons" or "public/wallpapers") and the allowed-MIME map
// (icons accept png/jpeg/webp/gif; wallpapers accept jpeg/webp/png; SVG
// remains intentionally excluded everywhere — see allowedContentTypes for
// rationale).
func saveImageBytes(dataDir, subdir string, allowed map[string]string, data []byte) (*saveResult, *saveError) {
	contentType := http.DetectContentType(data)
	ext, ok := allowed[contentType]
	if !ok {
		return nil, &saveError{
			status: http.StatusBadRequest,
			code:   400,
			msg:    fmt.Sprintf("unsupported content type %q", contentType),
		}
	}

	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])[:hashLen]
	filename := hash + ext
	relPath := filepath.ToSlash(filepath.Join(subdir, filename))
	absDir := filepath.Join(dataDir, "uploads", filepath.FromSlash(subdir))
	absPath := filepath.Join(absDir, filename)

	if _, err := os.Stat(absPath); err == nil {
		return &saveResult{Path: relPath, Deduped: true, Type: contentType}, nil
	}

	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, &saveError{http.StatusInternalServerError, 500, "mkdir failed"}
	}
	tmp, err := os.CreateTemp(absDir, "tmp-*"+ext)
	if err != nil {
		return nil, &saveError{http.StatusInternalServerError, 500, "create temp failed"}
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, &saveError{http.StatusInternalServerError, 500, "write failed"}
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return nil, &saveError{http.StatusInternalServerError, 500, "close failed"}
	}
	if err := os.Rename(tmp.Name(), absPath); err != nil {
		os.Remove(tmp.Name())
		return nil, &saveError{http.StatusInternalServerError, 500, "rename failed"}
	}
	return &saveResult{Path: relPath, Deduped: false, Type: contentType}, nil
}
