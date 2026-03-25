package web

import (
	"log"

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
	
	srv := &Server{
		port:    port,
		engine:  engine,
		handler: handler.NewHandler(services),
	}

	srv.handler.RegisterRoutes(engine)

	return srv
}

// Start 启动服务器
func (s *Server) Start() error {
	log.Printf("Web服务器启动，端口: %d", s.port)
	return s.engine.Run(":8082")
}

// Stop 停止服务器
func (s *Server) Stop() {
	log.Println("Web服务器停止")
}