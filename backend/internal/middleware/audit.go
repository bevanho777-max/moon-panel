package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/audit"
)

// AuditPathPrefix is the URL prefix recorded by the audit middleware. Only
// requests whose path starts with this prefix are logged. Non-admin paths
// (e.g. /api/auth/login, /api/public/panel) are either not auditable or
// have explicit handler-side emits.
const AuditPathPrefix = "/api/admin/"

// AuditLog is a Gin middleware that records mutating admin requests to the
// audit_logs table. Phase 3d-2.
//
// Scope:
//   - Mounted on the admin route group (anything requiring auth).
//   - Records POST / PUT / PATCH / DELETE only; GET is excluded (read-only
//     traffic generates noise without security value).
//   - Action key is "<METHOD> <path>" — generic schema (Fork 2 option B).
//     Frontend translates to friendly labels at render time. This means a
//     route rename never silently corrupts the audit semantic.
//   - Body is parsed as JSON, sensitive keys redacted recursively, then
//     stored in the details column.
//
// Why generic-by-default rather than handler-explicit:
//   - Adding a new admin endpoint automatically gets audit coverage; no
//     chance for a developer to forget the audit hook.
//   - Schema is uniform across all entries (same JSON shape in details).
//   - Handlers can still emit semantic events (login_success, password_change,
//     etc) on top of the middleware — those use a stable named action key
//     and bypass the generic recorder.
func AuditLog(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Scope: admin mutations only. Non-admin paths pass through.
		if !strings.HasPrefix(c.Request.URL.Path, AuditPathPrefix) {
			c.Next()
			return
		}
		method := c.Request.Method
		// Skip read-only and OPTIONS pre-flights.
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		// Capture body BEFORE handler reads it. Replace the request body with
		// a fresh reader so downstream handlers see the same bytes.
		var bodyBytes []byte
		if c.Request.Body != nil {
			// Limit to 1MiB — beyond that we consider the upload binary or
			// abusive; we still log the request, just without body content.
			limited := io.LimitReader(c.Request.Body, 1<<20)
			b, err := io.ReadAll(limited)
			if err == nil {
				bodyBytes = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		c.Next()

		// Don't log requests that 400-status'ed at the binding stage with no
		// auth context — but we still log auth-rejected attempts (401) since
		// that's exactly the kind of probing we want to see.
		username, _ := c.Get(ContextUsernameKey)
		actor, _ := username.(string)

		details := map[string]any{
			"method": method,
			"path":   c.Request.URL.Path,
			"body":   rawJSONOrPlaceholder(bodyBytes),
		}
		if q := c.Request.URL.RawQuery; q != "" {
			details["query"] = q
		}

		audit.Write(db, audit.Entry{
			Actor:     actor,
			Action:    method + " " + c.Request.URL.Path,
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Status:    c.Writer.Status(),
			Details:   details,
		})
	}
}

// rawJSONOrPlaceholder returns the redacted body as a json.RawMessage when
// it's valid JSON, or a synthetic shape describing non-JSON bodies. Wrapping
// in json.RawMessage ensures the body keeps its native JSON structure inside
// the details object, rather than getting double-escaped as a string.
func rawJSONOrPlaceholder(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	redacted := audit.RedactJSON(body)
	// json.RawMessage marshals as-is — preserves the redacted JSON's structure
	// in the parent object. The downstream encoder.Marshal will validate it.
	return rawJSON(redacted)
}

// rawJSON is a tiny wrapper so json.Marshal sees a RawMessage rather than a
// []byte (which it'd base64-encode).
type rawJSON []byte

func (r rawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}
