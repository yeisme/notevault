package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/configs"
)

// CORSMiddleware CORS中间件.
func CORSMiddleware(cfg configs.ServerConfig) gin.HandlerFunc {
	config := cors.DefaultConfig()

	if cfg.Debug {
		config.AllowAllOrigins = true
	} else {
		config.AllowOrigins = []string{"*"}
	}

	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}

	return cors.New(config)
}
