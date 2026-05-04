package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

// SiteStatsHandler exposes GET /api/site/stats. v0.2.1: lightweight public
// (no auth) endpoint feeding the bottom status bar that the "risen" theme
// shows. Different from the auth-gated /api/admin/stats — this one is
// 4 fields without the audit count, and visible to anonymous viewers
// because the status bar renders on the public home page too.
type SiteStatsHandler struct {
	DB        *gorm.DB
	StartedAt time.Time
}

func (h *SiteStatsHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/site/stats", h.get)
}

func (h *SiteStatsHandler) get(c *gin.Context) {
	var groups, cards int64
	h.DB.Model(&model.Group{}).Count(&groups)
	h.DB.Model(&model.Card{}).Count(&cards)
	uptime := int64(time.Since(h.StartedAt).Seconds())
	OK(c, gin.H{
		"version":        Version, // shared with internal/api/version.go
		"cards_count":    cards,
		"groups_count":   groups,
		"uptime_seconds": uptime,
	})
}
