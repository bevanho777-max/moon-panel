package api

import (
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var startedAt = time.Now()

func RegisterHealth(rg *gin.RouterGroup) {
	rg.GET("/health", func(c *gin.Context) {
		OK(c, gin.H{
			"status":     "ok",
			"uptime_sec": int64(time.Since(startedAt).Seconds()),
			"go":         runtime.Version(),
		})
	})
}
