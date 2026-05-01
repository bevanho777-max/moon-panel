package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// Sub returns the embedded frontend filesystem rooted at dist/.
func Sub() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// SPAHandler serves the embedded frontend with SPA fallback to index.html.
// Returns a Gin NoRoute handler — install with router.NoRoute(SPAHandler(fsys)).
func SPAHandler(fsys fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(fsys, path); err != nil {
			// SPA fallback — let the router handle the route client-side.
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
