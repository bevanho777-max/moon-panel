package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

type CardHandler struct {
	DB *gorm.DB
}

func (h *CardHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/cards", requireAuth)
	g.GET("", h.list)
	g.GET("/:id", h.getOne)
	g.POST("", h.create)
	g.PUT("/reorder", h.reorder) // must come before /:id to avoid id-binding conflict
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

type reorderEntry struct {
	ID      uint  `json:"id" binding:"required"`
	Sort    int   `json:"sort"`
	GroupID *uint `json:"group_id"` // optional cross-group move (cards only)
}

type reorderRequest struct {
	Items []reorderEntry `json:"items" binding:"required"`
}

// reorder accepts a batch of {id, sort, group_id?} pairs and applies them
// in a single transaction. Used by the drag-drop UI in admin Cards.
//
// Why one batch endpoint vs N individual PUTs:
//   - Atomicity: a partial reorder leaves the list in a half-sorted state.
//     A transaction rolls back cleanly on any failure.
//   - Audit log gets ONE entry summarizing the change rather than N entries
//     that look like routine edits — easier to scan after a sort op.
//   - Network economy: 30-card sort is one request not 30.
func (h *CardHandler) reorder(c *gin.Context) {
	var req reorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	if len(req.Items) == 0 {
		Fail(c, http.StatusBadRequest, 400, "items must not be empty")
		return
	}
	if len(req.Items) > 500 {
		Fail(c, http.StatusBadRequest, 400, "too many items (max 500 per reorder)")
		return
	}

	// Capture pre-state for audit diff. A keyed lookup is O(1) per id;
	// 500 cards is microseconds.
	ids := make([]uint, len(req.Items))
	for i, it := range req.Items {
		ids[i] = it.ID
	}
	var existing []model.Card
	if err := h.DB.Where("id IN ?", ids).Find(&existing).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	if len(existing) != len(req.Items) {
		Fail(c, http.StatusBadRequest, 400, "one or more card ids not found")
		return
	}

	type beforeAfter struct {
		ID         uint `json:"id"`
		BeforeSort int  `json:"before_sort"`
		AfterSort  int  `json:"after_sort"`
		BeforeGrp  uint `json:"before_group,omitempty"`
		AfterGrp   uint `json:"after_group,omitempty"`
	}
	prior := make(map[uint]model.Card, len(existing))
	for _, c := range existing {
		prior[c.ID] = c
	}
	diffs := make([]beforeAfter, 0, len(req.Items))

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			updates := map[string]any{"sort": it.Sort}
			diff := beforeAfter{ID: it.ID, BeforeSort: prior[it.ID].Sort, AfterSort: it.Sort}
			if it.GroupID != nil && *it.GroupID != 0 && *it.GroupID != prior[it.ID].GroupID {
				if !h.groupExists(*it.GroupID) {
					return errors.New("target group does not exist")
				}
				updates["group_id"] = *it.GroupID
				diff.BeforeGrp = prior[it.ID].GroupID
				diff.AfterGrp = *it.GroupID
			}
			diffs = append(diffs, diff)
			if err := tx.Model(&model.Card{}).Where("id = ?", it.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "reorder failed: "+err.Error())
		return
	}

	// Audit middleware will log the raw request, but a semantic action with
	// the explicit diff is more useful for admins reviewing reorder ops.
	// The handler-emitted version uses action=cards_reorder for clean UI
	// translation.
	OK(c, gin.H{"updated": len(req.Items), "diffs": diffs})
}

// getOne returns a single card by id. Used by the admin editor to refetch full
// state before opening the edit modal — avoids stale list-cache data.
func (h *CardHandler) getOne(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	var card model.Card
	if err := h.DB.First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, http.StatusNotFound, 404, "card not found")
			return
		}
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	OK(c, card)
}

// list returns a flat array of cards. Admin UI uses this for editing/filtering;
// the public homepage uses the nested shape via /api/public/panel.
// Optional ?group_id=N filters to one group; nonexistent group → empty array.
func (h *CardHandler) list(c *gin.Context) {
	var cards []model.Card
	q := h.DB.Order("group_id ASC, sort ASC, id ASC")
	if gid := c.Query("group_id"); gid != "" {
		n, err := strconv.ParseUint(gid, 10, 32)
		if err != nil || n == 0 {
			Fail(c, http.StatusBadRequest, 400, "invalid group_id")
			return
		}
		q = q.Where("group_id = ?", n)
	}
	if err := q.Find(&cards).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	OK(c, cards)
}

type cardWriteRequest struct {
	GroupID      *uint  `json:"group_id"`        // pointer: 0 vs missing
	Title        string `json:"title"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	URLInternal  string `json:"url_internal"`
	URLExternal  string `json:"url_external"`
	URLDefault   string `json:"url_default"`     // "" | "internal" | "external"
	OpenInNewTab *bool  `json:"open_in_new_tab"` // pointer: false vs missing
	Sort         *int   `json:"sort"`            // pointer: 0 vs missing
}

func (h *CardHandler) create(c *gin.Context) {
	var req cardWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	if req.GroupID == nil || *req.GroupID == 0 {
		Fail(c, http.StatusBadRequest, 400, "group_id required")
		return
	}
	if msg := validateCardWrite(&req, true); msg != "" {
		Fail(c, http.StatusBadRequest, 400, msg)
		return
	}
	if !h.groupExists(*req.GroupID) {
		Fail(c, http.StatusNotFound, 404, "group not found")
		return
	}

	card := model.Card{
		GroupID:     *req.GroupID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Icon:        req.Icon,
		URLInternal: req.URLInternal,
		URLExternal: req.URLExternal,
		URLDefault:  req.URLDefault,
	}
	if req.OpenInNewTab != nil {
		card.OpenInNewTab = *req.OpenInNewTab
	} else {
		card.OpenInNewTab = true
	}

	// v0.2.19: sort=0 视为"未提供" → 走 max+10 fallback (新卡放底部, Bevan
	// daily UX 反馈 "新建卡片放最后一行"). frontend emptyForm 默认 sort:0 短路
	// 原 nil check 导致新卡顶部, 加 *req.Sort > 0 短路修复.
	if req.Sort != nil && *req.Sort > 0 {
		card.Sort = *req.Sort
	} else {
		card.Sort = h.nextSortInGroup(*req.GroupID)
	}

	if err := h.DB.Create(&card).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "create failed")
		return
	}
	OK(c, card)
}

func (h *CardHandler) update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	var req cardWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}

	var card model.Card
	if err := h.DB.First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, http.StatusNotFound, 404, "card not found")
			return
		}
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	if msg := validateCardWrite(&req, false); msg != "" {
		Fail(c, http.StatusBadRequest, 400, msg)
		return
	}

	updates := map[string]any{
		// PUT semantics: client sends full state. Empty strings are explicit clears.
		"description":  req.Description,
		"icon":         req.Icon,
		"url_internal": req.URLInternal,
		"url_external": req.URLExternal,
	}

	if title := strings.TrimSpace(req.Title); title != "" {
		updates["title"] = title
	}
	if req.URLDefault != "" {
		updates["url_default"] = req.URLDefault
	}
	if req.OpenInNewTab != nil {
		updates["open_in_new_tab"] = *req.OpenInNewTab
	}

	// Group move: validate target exists, recalc sort to end-of-new-group unless caller pinned it.
	movingGroup := req.GroupID != nil && *req.GroupID != 0 && *req.GroupID != card.GroupID
	if movingGroup {
		if !h.groupExists(*req.GroupID) {
			Fail(c, http.StatusNotFound, 404, "target group not found")
			return
		}
		updates["group_id"] = *req.GroupID
		if req.Sort == nil {
			updates["sort"] = h.nextSortInGroup(*req.GroupID)
		}
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}

	if err := h.DB.Model(&card).Updates(updates).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "update failed")
		return
	}
	h.DB.First(&card, id)
	OK(c, card)
}

func (h *CardHandler) delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	res := h.DB.Delete(&model.Card{}, id)
	if res.Error != nil {
		Fail(c, http.StatusInternalServerError, 500, "delete failed")
		return
	}
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, 404, "card not found")
		return
	}
	OK(c, gin.H{"deleted": id})
}

// validateCardWrite checks fields shared by create + update.
// requireURL: create always needs at least one URL; update too (PUT replaces).
func validateCardWrite(req *cardWriteRequest, isCreate bool) string {
	if isCreate {
		if strings.TrimSpace(req.Title) == "" {
			return "title required"
		}
	}
	if title := strings.TrimSpace(req.Title); title != "" && len(title) > 128 {
		return "title too long"
	}
	if len(req.Description) > 512 {
		return "description too long"
	}
	if strings.TrimSpace(req.URLInternal) == "" && strings.TrimSpace(req.URLExternal) == "" {
		return "url_internal and url_external cannot both be empty"
	}
	if req.URLDefault != "" && req.URLDefault != "internal" && req.URLDefault != "external" {
		return "url_default must be 'internal' or 'external'"
	}
	return ""
}

func (h *CardHandler) groupExists(id uint) bool {
	var count int64
	h.DB.Model(&model.Group{}).Where("id = ?", id).Count(&count)
	return count > 0
}

// nextSortInGroup returns max(sort)+10 for the given group, or 10 if empty.
func (h *CardHandler) nextSortInGroup(groupID uint) int {
	var maxSorts []int
	h.DB.Model(&model.Card{}).Where("group_id = ?", groupID).Order("sort DESC").Limit(1).Pluck("sort", &maxSorts)
	if len(maxSorts) > 0 {
		return maxSorts[0] + 10
	}
	return 10
}
