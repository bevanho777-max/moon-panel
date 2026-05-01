package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

// AuditLogHandler exposes the audit trail to the admin UI. Read-only — entries
// are written only by the audit middleware and explicit handler emits.
type AuditLogHandler struct {
	DB *gorm.DB
}

func (h *AuditLogHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/audit-logs", requireAuth)
	g.GET("", h.list)
}

type auditLogListResponse struct {
	Items []model.AuditLog `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}

// list returns a paginated, optionally filtered slice of audit log entries.
// Query params:
//   page (default 1)
//   size (default 20, capped at 100 — protects against memory blow-up)
//   action (substring match on action column; allows e.g. "login" to find
//           login_success and login_failure both)
//   actor (exact match on actor)
//   from / to (ISO 8601 timestamps; inclusive)
func (h *AuditLogHandler) list(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	q := h.DB.Model(&model.AuditLog{})
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		q = q.Where("action LIKE ?", "%"+action+"%")
	}
	if actor := strings.TrimSpace(c.Query("actor")); actor != "" {
		q = q.Where("actor = ?", actor)
	}
	if from := strings.TrimSpace(c.Query("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("timestamp >= ?", t)
		}
	}
	if to := strings.TrimSpace(c.Query("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("timestamp <= ?", t)
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	var items []model.AuditLog
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	OK(c, auditLogListResponse{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	})
}
