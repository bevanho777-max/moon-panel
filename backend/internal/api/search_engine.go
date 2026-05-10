package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/model"
)

type SearchEngineHandler struct {
	DB *gorm.DB
}

func (h *SearchEngineHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/admin/search-engines", requireAuth)
	g.GET("", h.list)
	g.POST("", h.create)
	g.PUT("/reorder", h.reorder) // before /:id to avoid id binding conflict
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

// v0.2.16 P0 b: search engines batch reorder for inline drag UX.
// Mirror GroupHandler.reorder pattern (atomic tx update; max items low because
// search engines list is typically <20).
type searchEngineReorderEntry struct {
	ID   uint `json:"id" binding:"required"`
	Sort int  `json:"sort"`
}

type searchEngineReorderRequest struct {
	Items []searchEngineReorderEntry `json:"items" binding:"required"`
}

func (h *SearchEngineHandler) reorder(c *gin.Context) {
	var req searchEngineReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	if len(req.Items) == 0 {
		Fail(c, http.StatusBadRequest, 400, "items must not be empty")
		return
	}
	if len(req.Items) > 100 {
		Fail(c, http.StatusBadRequest, 400, "too many items (max 100 per reorder)")
		return
	}

	ids := make([]uint, len(req.Items))
	for i, it := range req.Items {
		ids[i] = it.ID
	}
	var existing []model.SearchEngine
	if err := h.DB.Where("id IN ?", ids).Find(&existing).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	if len(existing) != len(req.Items) {
		Fail(c, http.StatusBadRequest, 400, "one or more search engine ids not found")
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			if err := tx.Model(&model.SearchEngine{}).Where("id = ?", it.ID).Update("sort", it.Sort).Error; err != nil {
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

func (h *SearchEngineHandler) list(c *gin.Context) {
	var engines []model.SearchEngine
	if err := h.DB.Order("sort ASC, id ASC").Find(&engines).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	OK(c, engines)
}

type searchEngineWriteRequest struct {
	Name        string `json:"name"`
	URLTemplate string `json:"url_template"`
	Icon        string `json:"icon"`
	IsDefault   *bool  `json:"is_default"` // pointer: false vs missing
	Sort        *int   `json:"sort"`
}

// Templates must contain at least one of {q} or {query}. Frontend substitutes
// both placeholders at search time, so admins can use either.
var placeholderPattern = regexp.MustCompile(`\{q(?:uery)?\}`)

func validateSearchEngineWrite(req *searchEngineWriteRequest, isCreate bool) string {
	if isCreate {
		if strings.TrimSpace(req.Name) == "" {
			return "name required"
		}
		if strings.TrimSpace(req.URLTemplate) == "" {
			return "url_template required"
		}
	}
	if name := strings.TrimSpace(req.Name); name != "" && len(name) > 64 {
		return "name too long (max 64)"
	}
	if t := strings.TrimSpace(req.URLTemplate); t != "" {
		if len(t) > 1024 {
			return "url_template too long (max 1024)"
		}
		if !strings.HasPrefix(strings.ToLower(t), "http://") && !strings.HasPrefix(strings.ToLower(t), "https://") {
			return "url_template must start with http:// or https://"
		}
		if !placeholderPattern.MatchString(t) {
			return `url_template must contain a query placeholder: {q} or {query}`
		}
	}
	if len(req.Icon) > 512 {
		return "icon too long (max 512)"
	}
	return ""
}

func (h *SearchEngineHandler) create(c *gin.Context) {
	var req searchEngineWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}
	if msg := validateSearchEngineWrite(&req, true); msg != "" {
		Fail(c, http.StatusBadRequest, 400, msg)
		return
	}
	engine := model.SearchEngine{
		Name:        strings.TrimSpace(req.Name),
		URLTemplate: strings.TrimSpace(req.URLTemplate),
		Icon:        req.Icon,
	}
	if req.IsDefault != nil {
		engine.IsDefault = *req.IsDefault
	}
	// v0.2.19: sort=0 视为"未提供" → 走 max+10 fallback (新搜索引擎放底部, Bevan
	// daily UX 反馈一致, 跟 card.go + group.go 同模式). 只 createHandler 改, 不
	// 动 updateHandler 行 221 (update 时 sort=0 是用户明确意图, 应尊重).
	if req.Sort != nil && *req.Sort > 0 {
		engine.Sort = *req.Sort
	} else {
		var maxSorts []int
		h.DB.Model(&model.SearchEngine{}).Order("sort DESC").Limit(1).Pluck("sort", &maxSorts)
		engine.Sort = 10
		if len(maxSorts) > 0 {
			engine.Sort = maxSorts[0] + 10
		}
	}

	// If this engine is set as default, clear is_default on all others
	// (transactional to avoid two defaults coexisting).
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&engine).Error; err != nil {
			return err
		}
		if engine.IsDefault {
			if err := tx.Model(&model.SearchEngine{}).
				Where("id != ?", engine.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		Fail(c, http.StatusInternalServerError, 500, "create failed")
		return
	}
	OK(c, engine)
}

func (h *SearchEngineHandler) update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	var req searchEngineWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid request")
		return
	}

	var engine model.SearchEngine
	if err := h.DB.First(&engine, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(c, http.StatusNotFound, 404, "search engine not found")
			return
		}
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	if msg := validateSearchEngineWrite(&req, false); msg != "" {
		Fail(c, http.StatusBadRequest, 400, msg)
		return
	}

	updates := map[string]any{}
	if name := strings.TrimSpace(req.Name); name != "" {
		updates["name"] = name
	}
	if t := strings.TrimSpace(req.URLTemplate); t != "" {
		updates["url_template"] = t
	}
	// Icon allows empty string to clear. Always update on PUT.
	updates["icon"] = req.Icon

	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}

	wantDefault := req.IsDefault != nil && *req.IsDefault
	clearDefault := req.IsDefault != nil && !*req.IsDefault

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&engine).Updates(updates).Error; err != nil {
				return err
			}
		}
		if wantDefault {
			// Set this one default, clear others
			if err := tx.Model(&engine).Update("is_default", true).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.SearchEngine{}).
				Where("id != ?", engine.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		} else if clearDefault {
			if err := tx.Model(&engine).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		Fail(c, http.StatusInternalServerError, 500, "update failed")
		return
	}
	h.DB.First(&engine, id)
	OK(c, engine)
}

func (h *SearchEngineHandler) delete(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		Fail(c, http.StatusBadRequest, 400, "invalid id")
		return
	}
	res := h.DB.Delete(&model.SearchEngine{}, id)
	if res.Error != nil {
		Fail(c, http.StatusInternalServerError, 500, "delete failed")
		return
	}
	if res.RowsAffected == 0 {
		Fail(c, http.StatusNotFound, 404, "search engine not found")
		return
	}
	OK(c, gin.H{"deleted": id})
}
