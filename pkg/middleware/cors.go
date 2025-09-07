package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/configs"
)

// CORSMiddleware CORS中间件.
func CORSMiddleware(cfg configs.ServerConfig) gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}

	config.AllowWebSockets = true
	config.AllowFiles = true

	if cfg.Debug {
		config.AllowAllOrigins = true
	}

	return cors.New(config)
}
