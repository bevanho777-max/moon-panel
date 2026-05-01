package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

type SettingHandler struct {
	DB *gorm.DB
}

func (h *SettingHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/settings", requireAuth)
	g.GET("", h.list)
	g.PUT("", h.upsert)
}

// list returns all settings as a flat object: { key: value }.
// Phase 3b-1 only seeds nothing — future phases populate (wallpaper, theme,
// site title, etc). Empty result is fine.
func (h *SettingHandler) list(c *gin.Context) {
	var rows []model.Setting
	if err := h.DB.Order("key ASC").Find(&rows).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Key] = r.Value
	}
	OK(c, out)
}

// upsert accepts a flat object: { key1: value1, key2: value2 }.
// Each entry is INSERT ... ON CONFLICT(key) DO UPDATE.
// To delete a key, send empty string as value (Phase 3b-1 doesn't expose
// delete; settings table grows monotonically until Phase 4 adds cleanup).
func (h *SettingHandler) upsert(c *gin.Context) {
	var body map[string]string
	if err := c.ShouldBindJSON(&body); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request (expected flat key→value object)")
		return
	}
	if len(body) == 0 {
		OK(c, gin.H{"updated": 0})
		return
	}
	for k := range body {
		if k == "" {
			Fail(c, http.StatusBadRequest, 400, "empty key not allowed")
			return
		}
		if len(k) > 64 {
			Fail(c, http.StatusBadRequest, 400, "key too long (max 64): "+k)
			return
		}
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		for k, v := range body {
			s := model.Setting{Key: k, Value: v}
			// GORM's Save does insert-or-update on primary key (key).
			if err := tx.Save(&s).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		Fail(c, http.StatusInternalServerError, 500, "upsert failed")
		return
	}
	OK(c, gin.H{"updated": len(body)})
}
