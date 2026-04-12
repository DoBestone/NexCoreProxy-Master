package web

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nexcoreproxy-master/internal/handler"
	"nexcoreproxy-master/internal/service"
)

// Server Web服务器
type Server struct {
	port    int
	engine  *gin.Engine
	handler *handler.Handler
}

// NewServer 创建Web服务器
func NewServer(port int, services *service.Services) *Server {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.Default()

	// CORS 中间件
	engine.Use(corsMiddleware())

	srv := &Server{
		port:    port,
		engine:  engine,
		handler: handler.NewHandler(services),
	}

	srv.handler.RegisterRoutes(engine)

	return srv
}

// corsMiddleware CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := os.Getenv("CORS_ORIGINS") // 逗号分隔的允许来源列表
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// 检查是否允许该来源
		allowed := false
		if allowedOrigins == "" || allowedOrigins == "*" {
			allowed = true // 开发模式允许所有来源
		} else {
			for _, o := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(o) == origin {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", int((12*time.Hour).Seconds())))
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("Web服务器启动，端口: %d", s.port)
	return s.engine.Run(fmt.Sprintf(":%d", s.port))
}

// Stop 停止服务器
func (s *Server) Stop() {
	log.Println("Web服务器停止")
}
