package api

import "github.com/gin-gonic/gin"

// Build-time injected version metadata. The Dockerfile + release.yml pass
// the tag name, build timestamp, and commit SHA via go build -ldflags
// -X overrides. Local `go build` keeps the dev defaults so a binary built
// outside the release pipeline reports itself as such (UI shows "vdev").
var (
	Version   = "dev"
	BuildDate = "unknown"
	Commit    = "unknown"
)

// RegisterVersion mounts GET /api/version. Public (no auth) — the version
// badge UI on the home and admin layouts fetches this once per page load
// to render the current build, with no auth gate so the public home page
// can show it before login.
func RegisterVersion(rg *gin.RouterGroup) {
	rg.GET("/version", func(c *gin.Context) {
		commit := Commit
		// Truncate to 7-char short SHA for display. We keep the full SHA
		// in the LDFLAGS-injected variable in case ops needs it via a
		// debug endpoint, but the public API stays compact.
		if len(commit) > 7 {
			commit = commit[:7]
		}
		OK(c, gin.H{
			"version":    Version,
			"build_date": BuildDate,
			"commit":     commit,
		})
	})
}
