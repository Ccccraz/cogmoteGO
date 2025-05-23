package logger

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func GinMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.Info("[GIN]",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
			"latency", time.Since(start),
		)
	}
}
