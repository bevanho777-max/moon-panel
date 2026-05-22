package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/auth"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
)

type GroupHandler struct {
	DB *gorm.DB
}

func (h *GroupHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/groups", requireAuth)
	g.GET("", h.list)
	g.POST("", h.create)
	g.PUT("/reorder", h.reorder) // before /:id to avoid id binding conflict
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

type groupReorderEntry struct {
	ID   uint `json:"id" binding:"required"`
	Sort int  `json:"sort"`
}

type groupReorderRequest struct {
	Items []groupReorderEntry `json:"items" binding:"required"`
}

// reorder applies a batch of {id, sort} updates atomically. See the
// equivalent CardHandler.reorder for the design rationale (atomicity,
// audit clarity, network economy).
func (h *GroupHandler) reorder(c *gin.Context) {
	var req groupReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	if len(req.Items) == 0 {
		Fail(c, http.StatusBadRequest, 400, "items must not be empty")
		return
	}
	if len(req.Items) > 200 {
		Fail(c, http.StatusBadRequest, 400, "too many items (max 200 per reorder)")
		return
	}

	ids := make([]uint, len(req.Items))
	for i, it := range req.Items {
		ids[i] = it.ID
	}
	var existing []model.Group
	if err := h.DB.Where("id IN ?", ids).Find(&existing).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	if len(existing) != len(req.Items) {
		Fail(c, http.StatusBadRequest, 400, "one or more group ids not found")
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			if err := tx.Model(&model.Group{}).Where("id = ?", it.ID).Update("sort", it.Sort).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "reorder failed: "+err.Error())
		return
	}
	OK(c, gin.H{"updated": len(req.Items)})
}

func (h *GroupHandler) list(c *gin.Context) {
	var groups []model.Group
	if err := h.DB.Order("sort ASC, id ASC").Find(&groups).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	OK(c, groups)
}

type groupWriteRequest struct {
	Name string  `json:"name"`
	Icon string  `json:"icon"`
	Sort *int    `json:"sort"` // pointer so we can distinguish "omitted" from "0"
}

func (h *GroupHandler) create(c *gin.Context) {
	var req groupWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		Fail(c, http.StatusBadRequest, 400, "name required")
		return
	}
	if len(name) > 128 {
		Fail(c, http.StatusBadRequest, 400, "name too long")
		return
	}

	// v0.2.28 R1: stamp owner_id from the authed session. See the equivalent
	// note in card.go's create handler for the R2/R3 rationale.
	claims := c.MustGet(middleware.ContextClaimsKey).(*auth.Claims)
	g := model.Group{
		OwnerID: claims.UserID,
		Name:    name,
		Icon:    req.Icon,
	}
	// v0.2.19: sort=0 视为"未提供" → 走 max+10 fallback (新分组放底部, Bevan
	// daily UX 反馈一致, 跟 card.go 同模式).
	if req.Sort != nil && *req.Sort > 0 {
		g.Sort = *req.Sort
	} else {
		// Default to (max existing sort) + 10 so manual reordering has gaps
		// to slot into. Empty table → start at 10.
		var maxSorts []int
		h.DB.Model(&model.Group{}).Order("sort DESC").Limit(1).Pluck("sort", &maxSorts)
		g.Sort = 10
		if len(maxSorts) > 0 {
			g.Sort = maxSorts[0] + 10
		}
	}

	if err := h.DB.Create(&g).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "create failed")
		return
	}
	OK(c, g)
}

func (h *GroupHandler) update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	var req groupWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}

	var g model.Group
	if err := h.DB.First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, http.StatusNotFound, 404, "group not found")
			return
		}
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	updates := map[string]any{}
	if name := strings.TrimSpace(req.Name); name != "" {
		if len(name) > 128 {
			Fail(c, http.StatusBadRequest, 400, "name too long")
			return
		}
		updates["name"] = name
	}
	// Icon allows empty string to clear it; only update if the field is present.
	// Since groupWriteRequest doesn't distinguish missing from empty for Icon,
	// always update it when the request body parses successfully.
	updates["icon"] = req.Icon
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}

	if len(updates) == 0 {
		OK(c, g)
		return
	}
	if err := h.DB.Model(&g).Updates(updates).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "update failed")
		return
	}
	// Reload to get fresh timestamps.
	h.DB.First(&g, id)
	OK(c, g)
}

func (h *GroupHandler) delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	// Cards are removed via the GORM constraint OnDelete:CASCADE on the
	// Group→Cards association, which translates to ON DELETE CASCADE in SQLite.
	res := h.DB.Delete(&model.Group{}, id)
	if res.Error != nil {
		Fail(c, http.StatusInternalServerError, 500, "delete failed")
		return
	}
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, 404, "group not found")
		return
	}
	OK(c, gin.H{"deleted": id})
}

func parseID(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, errors.New("zero id")
	}
	return uint(n), nil
}

