package audit

import (
	"encoding/json"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

// Retention policy constants. Cleanup deletes whichever cap hits first:
//   - rows older than RetentionDays (90 days)
//   - all rows except the most recent RetentionMaxRows (1000) by id
const (
	RetentionDays    = 90
	RetentionMaxRows = 1000
	// Cleanup runs probabilistically once per ~CleanupOdds inserts. With
	// odds=100 and a typical admin session (10-30 actions/day), cleanup runs
	// every few days under normal load — frequent enough to keep the table
	// bounded, infrequent enough to never block a write noticeably.
	CleanupOdds = 100
)

// Entry is the input shape for Write. Mirrors model.AuditLog but excludes
// auto-managed fields (ID / CreatedAt). Lets callers build entries without
// importing gorm-tagged types.
type Entry struct {
	Actor      string
	Action     string
	TargetType string
	TargetID   string
	IP         string
	UserAgent  string
	Status     int
	Details    map[string]any // marshaled to JSON; nil → "{}"
}

// Write persists one entry synchronously. SQLite WAL mode handles single-row
// inserts in ~100μs–1ms — synchronous keeps the implementation simple and
// avoids goroutine/channel/shutdown coordination. If profiling later shows
// a real bottleneck (it won't, for this scale), switch to a batched async
// flusher then.
//
// Errors are intentionally swallowed and logged: the audit log failing should
// NEVER block or break the user-facing operation. The cost of a missed audit
// entry is a degraded log; the cost of a 500 because audit insertion failed
// is much worse — the user can't even change their password.
func Write(db *gorm.DB, e Entry) {
	var details []byte
	if e.Details != nil {
		details, _ = json.Marshal(e.Details)
	}
	if len(details) == 0 {
		details = []byte("{}")
	}
	now := time.Now()
	row := model.AuditLog{
		Timestamp:  now,
		Actor:      truncate(e.Actor, 64),
		Action:     truncate(e.Action, 128),
		TargetType: truncate(e.TargetType, 32),
		TargetID:   truncate(e.TargetID, 64),
		IP:         truncate(e.IP, 64),
		UserAgent:  truncate(e.UserAgent, 200),
		Status:     e.Status,
		Details:    string(details),
	}
	// db.Create logs internally if it fails (gorm Logger.Warn). We don't
	// surface to the caller — see comment above.
	_ = db.Create(&row).Error

	// Probabilistic cleanup: avoids a background goroutine and aligns with
	// the project pattern (see middleware/ratelimit.go opportunistic eviction).
	if rand.Intn(CleanupOdds) == 0 {
		Cleanup(db)
	}
}

// Cleanup deletes audit log rows older than RetentionDays, then trims the
// table to RetentionMaxRows newest entries. Idempotent. Safe to call from
// startup (one-shot) or from Write (probabilistic). Returns the count of
// deleted rows for logging convenience.
func Cleanup(db *gorm.DB) int64 {
	cutoff := time.Now().Add(-RetentionDays * 24 * time.Hour)
	res := db.Where("timestamp < ?", cutoff).Delete(&model.AuditLog{})
	deleted := res.RowsAffected

	// Cap by row count: delete rows with id NOT in (most recent N by id).
	// Single statement, idempotent. Uses id rather than timestamp for the
	// "newest" definition since id is the primary key (cheap index scan).
	res2 := db.Exec(
		`DELETE FROM audit_logs WHERE id NOT IN (
			SELECT id FROM audit_logs ORDER BY id DESC LIMIT ?
		)`,
		RetentionMaxRows,
	)
	deleted += res2.RowsAffected
	return deleted
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
