package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

// StatsHandler exposes GET /api/admin/stats — counters powering the admin
// Overview page. Auth-gated since the numbers expose deployment scale (a
// public endpoint would help an attacker decide whether the panel's worth
// spending more time on).
type StatsHandler struct {
	DB *gorm.DB
}

func (h *StatsHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin")
	g.Use(requireAuth)
	g.GET("/stats", h.get)
}

func (h *StatsHandler) get(c *gin.Context) {
	var groups, cards, engines, audit int64
	h.DB.Model(&model.Group{}).Count(&groups)
	h.DB.Model(&model.Card{}).Count(&cards)
	h.DB.Model(&model.SearchEngine{}).Count(&engines)
	// Audit log: rows in the last 7 days. Column is "timestamp" (not
	// created_at) — see internal/audit/store.go retention sweep.
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	h.DB.Model(&model.AuditLog{}).Where("timestamp >= ?", sevenDaysAgo).Count(&audit)

	OK(c, gin.H{
		"groups_count":  groups,
		"cards_count":   cards,
		"engines_count": engines,
		"audit_count":   audit,
	})
}
