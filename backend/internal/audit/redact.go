// Package audit provides building blocks for the admin action audit log:
// recursive redaction of sensitive fields, cleanup helpers, and explicit
// emit functions called by handlers.
package audit

import (
	"encoding/json"
	"strings"
)

// SensitiveKeys are field names whose values must NEVER be persisted to the
// audit log. Lower-cased, exact match (after lowercasing). Includes Phase 3d-3
// 2FA fields pre-emptively so adding TOTP later doesn't require revisiting
// this list — easier to add now than to discover a leak later.
var SensitiveKeys = map[string]struct{}{
	"password":     {},
	"old_password": {},
	"new_password": {},
	"totp_code":    {},
	"totp_secret":  {},
	"backup_code":  {},
	// HTTP auth identifiers — even if they shouldn't be in a body, defense in
	// depth catches misuse (e.g. someone POSTs a token in a JSON field).
	"authorization": {},
	"cookie":        {},
	"token":         {},
}

const redactedMarker = "[REDACTED]"

// RedactJSON parses raw JSON bytes, walks the structure recursively, replaces
// values for any key in SensitiveKeys with "[REDACTED]", and re-marshals. On
// parse failure (non-JSON body, e.g. multipart form bytes) returns a synthetic
// JSON object indicating the body was opaque — never returns the raw bytes.
//
// Why parse-then-walk rather than regex: a JSON-aware approach handles nested
// objects, arrays of objects, and case variations correctly. Regex on raw
// bytes would miss `{"data":{"password":"..."}}` patterns or accidentally
// redact non-sensitive fields containing the substring "password" (e.g.
// "passwordless"). Parse cost is microseconds for typical request bodies.
func RedactJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		// Not JSON — could be form data, multipart, binary upload, etc.
		// Don't store the raw bytes. Replace with size hint only.
		opaque := map[string]any{"_opaque_body_bytes": len(raw)}
		out, _ := json.Marshal(opaque)
		return out
	}
	walked := walk(v)
	out, err := json.Marshal(walked)
	if err != nil {
		return []byte(`{"_redact_error":"marshal failed"}`)
	}
	return out
}

// walk recurses through arbitrary JSON-decoded structures (maps / slices /
// scalars). Maps with sensitive keys have their values overwritten with the
// redaction marker. Returns the cleaned structure (mutates in place for
// efficiency, but exported as a fresh value to avoid surprising callers).
func walk(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if _, sensitive := SensitiveKeys[strings.ToLower(k)]; sensitive {
				t[k] = redactedMarker
				continue
			}
			t[k] = walk(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = walk(val)
		}
		return t
	default:
		return v
	}
}
