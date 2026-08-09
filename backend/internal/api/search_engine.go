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
	g.POST("/restore-builtins", h.restoreBuiltins) // before /:id (see reorder note)
	g.PUT("/reorder", h.reorder)                   // before /:id to avoid id binding conflict
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)
}

// builtinIconPrefix hosts the search engine icons. Direct upstream favicons
// (google.com / bing.com / …) aren't reliably reachable from mainland China,
// so we pin walkxcode/dashboard-icons on jsdelivr instead.
const builtinIconPrefix = "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/"

// Search engine categories. The frontend mirrors this set (and its display
// order / labels) in frontend/src/utils/searchCategories.ts — adding a category
// means touching both sides.
const (
	CategoryWeb   = "web"
	CategoryImage = "image"
	CategoryMusic = "music"
	CategoryVideo = "video"
)

// SearchEngineCategories returns the valid category values in canonical order.
func SearchEngineCategories() []string {
	return []string{CategoryWeb, CategoryImage, CategoryMusic, CategoryVideo}
}

// IsValidSearchEngineCategory reports whether c is one of the four known
// categories. Empty is NOT valid here — callers decide whether to default it.
func IsValidSearchEngineCategory(c string) bool {
	switch c {
	case CategoryWeb, CategoryImage, CategoryMusic, CategoryVideo:
		return true
	}
	return false
}

// BuiltinSearchEngines is the single source of truth for the engines the panel
// ships with. Used both by first-start seeding (cmd/server bootstrap) and by
// the restore-builtins endpoint, so the two can never drift apart.
//
// Sort values are the canonical display order (web 10-70, image 110+, music
// 210+, video 310+ — leaving room to slot entries in without renumbering);
// restoreBuiltins re-bases them onto max(sort) so re-added engines land at the
// bottom instead of colliding with whatever the user has already arranged.
//
// v0.2.31: unsplash.png / imdb.png / themoviedb.png are absent from the
// walkxcode CDN (404), so those three fall back to their category's lucide
// glyph rather than shipping a broken <img>.
func BuiltinSearchEngines() []model.SearchEngine {
	return []model.SearchEngine{
		// web
		{Name: "Google", URLTemplate: "https://www.google.com/search?q={query}", Icon: builtinIconPrefix + "google.png", Category: CategoryWeb, IsDefault: true, Sort: 10},
		{Name: "Bing", URLTemplate: "https://www.bing.com/search?q={query}", Icon: builtinIconPrefix + "bing.png", Category: CategoryWeb, IsDefault: false, Sort: 20},
		{Name: "DuckDuckGo", URLTemplate: "https://duckduckgo.com/?q={query}", Icon: builtinIconPrefix + "duckduckgo.png", Category: CategoryWeb, IsDefault: false, Sort: 30},
		{Name: "Brave", URLTemplate: "https://search.brave.com/search?q={query}", Icon: builtinIconPrefix + "brave.png", Category: CategoryWeb, IsDefault: false, Sort: 40},
		{Name: "Startpage", URLTemplate: "https://www.startpage.com/sp/search?query={query}", Icon: builtinIconPrefix + "startpage.png", Category: CategoryWeb, IsDefault: false, Sort: 50},
		{Name: "Yandex", URLTemplate: "https://yandex.com/search/?text={query}", Icon: builtinIconPrefix + "yandex.png", Category: CategoryWeb, IsDefault: false, Sort: 60},
		{Name: "百度", URLTemplate: "https://www.baidu.com/s?wd={query}", Icon: builtinIconPrefix + "baidu.png", Category: CategoryWeb, IsDefault: false, Sort: 70},

		// image
		{Name: "Google 图片", URLTemplate: "https://www.google.com/search?tbm=isch&q={query}", Icon: builtinIconPrefix + "google.png", Category: CategoryImage, IsDefault: false, Sort: 110},
		{Name: "Bing 图片", URLTemplate: "https://www.bing.com/images/search?q={query}", Icon: builtinIconPrefix + "bing.png", Category: CategoryImage, IsDefault: false, Sort: 120},
		{Name: "Unsplash", URLTemplate: "https://unsplash.com/s/photos/{query}", Icon: "lucide:image", Category: CategoryImage, IsDefault: false, Sort: 130},
		{Name: "Pinterest", URLTemplate: "https://www.pinterest.com/search/pins/?q={query}", Icon: builtinIconPrefix + "pinterest.png", Category: CategoryImage, IsDefault: false, Sort: 140},

		// music
		{Name: "YouTube Music", URLTemplate: "https://music.youtube.com/search?q={query}", Icon: builtinIconPrefix + "youtube-music.png", Category: CategoryMusic, IsDefault: false, Sort: 210},
		{Name: "Spotify", URLTemplate: "https://open.spotify.com/search/{query}", Icon: builtinIconPrefix + "spotify.png", Category: CategoryMusic, IsDefault: false, Sort: 220},
		{Name: "SoundCloud", URLTemplate: "https://soundcloud.com/search?q={query}", Icon: builtinIconPrefix + "soundcloud.png", Category: CategoryMusic, IsDefault: false, Sort: 230},

		// video
		{Name: "YouTube", URLTemplate: "https://www.youtube.com/results?search_query={query}", Icon: builtinIconPrefix + "youtube.png", Category: CategoryVideo, IsDefault: false, Sort: 310},
		{Name: "IMDb", URLTemplate: "https://www.imdb.com/find/?q={query}", Icon: "lucide:clapperboard", Category: CategoryVideo, IsDefault: false, Sort: 320},
		{Name: "TMDB", URLTemplate: "https://www.themoviedb.org/search?query={query}", Icon: "lucide:clapperboard", Category: CategoryVideo, IsDefault: false, Sort: 330},
	}
}

// restoreBuiltins is additive and non-destructive: it inserts only the builtin
// engines whose Name is missing from the table, and never touches rows that
// already exist (a user who retitled or re-pointed "Google" keeps their edit —
// name match is the identity here, matching what the admin UI shows).
//
// Inserted rows are always IsDefault=false so restoring can't produce two
// defaults. Only if the whole table ends up with no default at all (e.g. the
// user deleted their default earlier) do we promote the lowest-sort row.
func (h *SearchEngineHandler) restoreBuiltins(c *gin.Context) {
	added := []string{}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var existing []model.SearchEngine
		if err := tx.Find(&existing).Error; err != nil {
			return err
		}
		byName := make(map[string]struct{}, len(existing))
		maxSort := 0
		for _, e := range existing {
			byName[e.Name] = struct{}{}
			if e.Sort > maxSort {
				maxSort = e.Sort
			}
		}

		step := 0
		for _, b := range BuiltinSearchEngines() {
			if _, ok := byName[b.Name]; ok {
				continue
			}
			step += 10
			engine := model.SearchEngine{
				Name:        b.Name,
				URLTemplate: b.URLTemplate,
				Icon:        b.Icon,
				Category:    b.Category,
				IsDefault:   false,
				Sort:        maxSort + step,
			}
			if err := tx.Create(&engine).Error; err != nil {
				return err
			}
			added = append(added, engine.Name)
		}

		var defaultCount int64
		if err := tx.Model(&model.SearchEngine{}).Where("is_default = ?", true).Count(&defaultCount).Error; err != nil {
			return err
		}
		if defaultCount == 0 {
			var first model.SearchEngine
			if err := tx.Order("sort ASC, id ASC").First(&first).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil // empty table: nothing to promote
				}
				return err
			}
			if err := tx.Model(&model.SearchEngine{}).Where("id = ?", first.ID).Update("is_default", true).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		Fail(c, http.StatusInternalServerError, 500, "restore failed: "+err.Error())
		return
	}

	var engines []model.SearchEngine
	if err := h.DB.Order("sort ASC, id ASC").Find(&engines).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}
	OK(c, gin.H{"added": added, "engines": engines})
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
	Category    string `json:"category"` // "" = leave alone (update) / default web (create)
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
	// v0.2.31: category is optional on the wire so pre-0.2.31 API clients keep
	// working — create defaults it to "web", update leaves it untouched. A
	// value that IS supplied must be one of the known four.
	if c := strings.TrimSpace(req.Category); c != "" && !IsValidSearchEngineCategory(c) {
		return "category must be one of: " + strings.Join(SearchEngineCategories(), ", ")
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
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = CategoryWeb
	}
	engine := model.SearchEngine{
		Name:        strings.TrimSpace(req.Name),
		URLTemplate: strings.TrimSpace(req.URLTemplate),
		Icon:        req.Icon,
		Category:    category,
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

	// Category has no "clear" semantics — an engine always belongs somewhere —
	// so an omitted/blank value means "leave as is".
	if c := strings.TrimSpace(req.Category); c != "" {
		updates["category"] = c
	}

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
