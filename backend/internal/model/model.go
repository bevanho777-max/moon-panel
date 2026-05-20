package model

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string `gorm:"not null" json:"-"`
	// TOTP fields, populated when 2FA is enrolled. TOTPSecret is stored in
	// plaintext base32 (same trust posture as data/jwt.key — at-rest disk
	// access already implies full panel compromise). TOTPBackupCodes is JSON
	// array of bcrypt hashes; consumed once each on use.
	TOTPSecret      string    `gorm:"size:64" json:"-"`
	TOTPEnabled     bool      `gorm:"default:false" json:"totp_enabled"`
	TOTPBackupCodes string    `gorm:"type:text" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Group struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Icon      string    `gorm:"size:512" json:"icon"`
	Sort      int       `gorm:"index;default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Cards     []Card    `gorm:"foreignKey:GroupID;constraint:OnDelete:CASCADE" json:"cards,omitempty"`
}

type Card struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	GroupID       uint      `gorm:"index;not null" json:"group_id"`
	Title         string    `gorm:"size:128;not null" json:"title"`
	Description   string    `gorm:"size:512" json:"description"`
	Icon          string    `gorm:"size:512" json:"icon"`
	// Deprecated: 改用 icon 前缀编码（lucide:/upload:/http(s)://）。
	// 保留列只为避免 migration，Phase 6 文档整理时统一处理。
	IconType      string    `gorm:"size:16;default:url" json:"icon_type"`
	URLInternal   string    `gorm:"size:1024" json:"url_internal"`
	URLExternal   string    `gorm:"size:1024" json:"url_external"`
	URLDefault    string    `gorm:"size:16" json:"url_default"` // "" | internal | external
	OpenInNewTab  bool      `gorm:"default:true" json:"open_in_new_tab"`
	Sort          int       `gorm:"index;default:0" json:"sort"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SearchEngine struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	URLTemplate string    `gorm:"size:1024;not null" json:"url_template"`
	Icon        string    `gorm:"size:512" json:"icon"`
	IsDefault   bool      `gorm:"default:false" json:"is_default"`
	Sort        int       `gorm:"index;default:0" json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Setting struct {
	Key       string    `gorm:"primaryKey;size:64" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditLog is one entry in the admin audit trail. Phase 3d-2.
//
// Schema choices:
//   - Action is "verb noun" — for middleware-emitted entries we use
//     "<METHOD> <path>" (e.g. "PUT /api/admin/cards/123"); for explicit
//     handler emits we use a stable name (e.g. "login_success",
//     "password_change", "bootstrap"). UI translates both forms.
//   - Details is opaque JSON: request body (with sensitive keys redacted),
//     response status, optional context. Keep flexible — schema can evolve.
//   - We index Timestamp DESC for the typical "recent first" listing query;
//     id is the primary key so pagination by id works as a tiebreaker for
//     entries within the same second.
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Timestamp  time.Time `gorm:"index;not null" json:"timestamp"`
	Actor      string    `gorm:"size:64;index" json:"actor"`
	Action     string    `gorm:"size:128;not null;index" json:"action"`
	TargetType string    `gorm:"size:32;index" json:"target_type"`
	TargetID   string    `gorm:"size:64" json:"target_id"`
	IP         string    `gorm:"size:64" json:"ip"`
	UserAgent  string    `gorm:"size:200" json:"user_agent"`
	Status     int       `json:"status"`
	Details    string    `gorm:"type:text" json:"details"`
	CreatedAt  time.Time `json:"created_at"`
}

func All() []any {
	return []any{
		&User{},
		&Group{},
		&Card{},
		&SearchEngine{},
		&Setting{},
		&AuditLog{},
	}
}
