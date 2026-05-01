package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/moon-panel/moon-panel/internal/assets"
	"github.com/moon-panel/moon-panel/internal/middleware"
	"github.com/moon-panel/moon-panel/internal/model"
)

type PublicHandler struct {
	DB         *gorm.DB
	PublicMode bool
}

// Register mounts the public panel endpoint. When PublicMode is false, the
// endpoint requires a valid session — used for "fully private" deployments.
func (h *PublicHandler) Register(rg *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	g := rg.Group("/public")
	if !h.PublicMode {
		g.Use(requireAuth)
	}
	g.GET("/panel", h.getPanel)
}

func (h *PublicHandler) getPanel(c *gin.Context) {
	if !h.PublicMode {
		if _, ok := c.Get(middleware.ContextClaimsKey); !ok {
			Fail(c, http.StatusUnauthorized, 401, "unauthorized")
			return
		}
	}

	var groups []model.Group
	if err := h.DB.Preload("Cards", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort ASC, id ASC")
	}).Order("sort ASC, id ASC").Find(&groups).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	var engines []model.SearchEngine
	if err := h.DB.Order("sort ASC, id ASC").Find(&engines).Error; err != nil {
		Fail(c, http.StatusInternalServerError, 500, "db error")
		return
	}

	cities := loadCitiesSetting(h.DB)
	tempUnit := loadTempUnitSetting(h.DB)
	ui := loadUISettings(h.DB)

	OK(c, gin.H{
		"site": gin.H{
			"public_mode": h.PublicMode,
			"cities":      cities,
			"temp_unit":   tempUnit,
			"ui":          ui,
		},
		"groups":         groups,
		"search_engines": engines,
	})
}

// loadCitiesSetting reads widget.cities, parses the JSON array, and returns
// either the parsed slice or [] (never nil — keeps frontend type stable).
func loadCitiesSetting(db *gorm.DB) []map[string]any {
	var s model.Setting
	if err := db.Where("key = ?", "widget.cities").First(&s).Error; err != nil {
		return []map[string]any{}
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(s.Value), &out); err != nil {
		return []map[string]any{}
	}
	return out
}

// loadTempUnitSetting returns "C" or "F"; defaults to "C" on missing/invalid.
func loadTempUnitSetting(db *gorm.DB) string {
	var s model.Setting
	if err := db.Where("key = ?", "widget.temp_unit").First(&s).Error; err != nil {
		return "C"
	}
	if s.Value != "C" && s.Value != "F" {
		return "C"
	}
	return s.Value
}

// loadUISettings reads the three Phase 2.5c keys (ui.wallpaper /
// ui.wallpaper_blur / ui.theme_primary) and bundles them with the static
// list of builtin wallpaper IDs. The "builtins" field is what lets the
// frontend render the catalog grid without a separate API call — the IDs
// are stable across binary versions and constant per build, so shipping
// them in the panel payload is essentially free.
//
// Defaults: wallpaper=null, blur=0, theme_primary=null. A null primary
// means "use NaiveUI default blue" — frontend interprets accordingly.
// Validation lives here (not in the setting handler) so a future kerneled
// "settings UI" can write any value while consumers still see safe ones.
func loadUISettings(db *gorm.DB) gin.H {
	out := gin.H{
		"wallpaper":      nil,
		"wallpaper_blur": 0,
		"theme_primary":  nil,
		"builtins":       assets.BuiltinWallpaperIDs(),
	}
	var rows []model.Setting
	if err := db.Where("key IN ?", []string{
		"ui.wallpaper",
		"ui.wallpaper_blur",
		"ui.theme_primary",
	}).Find(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		switch r.Key {
		case "ui.wallpaper":
			if r.Value != "" {
				out["wallpaper"] = r.Value
			}
		case "ui.wallpaper_blur":
			n, err := strconv.Atoi(r.Value)
			if err != nil {
				continue
			}
			if n < 0 {
				n = 0
			} else if n > 20 {
				n = 20
			}
			out["wallpaper_blur"] = n
		case "ui.theme_primary":
			if r.Value != "" {
				out["theme_primary"] = r.Value
			}
		}
	}
	return out
}
