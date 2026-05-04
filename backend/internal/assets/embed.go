// Package assets embeds static SVG/image assets that ship inside the binary
// (currently: builtin wallpapers). Anything in this package is part of the
// compiled binary — no /data filesystem dependency.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed wallpapers/*.svg
var wallpaperFS embed.FS

// WallpaperFS returns the embedded wallpapers/ directory rooted at "wallpapers".
// Caller can fs.ReadFile(fs, "night.svg") etc.
func WallpaperFS() (fs.FS, error) {
	return fs.Sub(wallpaperFS, "wallpapers")
}

//go:embed themes/*.svg
var themeFS embed.FS

// ThemeFS returns the embedded themes/ directory. v0.2.1 ships two preview
// thumbnails: moon-preview.svg and risen-preview.svg, used by the admin
// theme picker.
func ThemeFS() (fs.FS, error) {
	return fs.Sub(themeFS, "themes")
}

// IsValidThemeID reports whether id matches a shipped theme preset.
func IsValidThemeID(id string) bool {
	for _, t := range []string{"moon", "risen"} {
		if t == id {
			return true
		}
	}
	return false
}

// BuiltinWallpaperIDs returns the canonical list of available builtin wallpaper
// IDs in display order. Used by the public panel endpoint to advertise choices
// to the frontend. v0.2.0 added 6 new SVGs:
//   galaxy, ocean, sunset, mountain, meadow, forest.
func BuiltinWallpaperIDs() []string {
	return []string{
		"night", "aurora", "graphite",
		"galaxy", "ocean", "sunset", "mountain", "meadow", "forest",
	}
}

// IsValidBuiltinID reports whether id matches a shipped wallpaper. Used by
// backup-restore fallback validation (ui.wallpaper="builtin:foo" must resolve).
func IsValidBuiltinID(id string) bool {
	for _, b := range BuiltinWallpaperIDs() {
		if b == id {
			return true
		}
	}
	return false
}
