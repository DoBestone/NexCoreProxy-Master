package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	var port int
	var dbHost, dbPort, dbUser, dbPass, dbName string

	flag.IntVar(&port, "port", 8080, "web server port")
	flag.StringVar(&dbHost, "db-host", "localhost", "database host")
	flag.StringVar(&dbPort, "db-port", "3306", "database port")
	flag.StringVar(&dbUser, "db-user", "root", "database user")
	flag.StringVar(&dbPass, "db-pass", "", "database password")
	flag.StringVar(&dbName, "db-name", "nexcore_proxy", "database name")
	flag.Parse()

	log.Println("Starting...")

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)
	
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	log.Println("数据库连接成功")

	// 创建路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 静态文件
	r.Static("/assets", "./web/dist/assets")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	// API
	api := r.Group("/api")
	{
		api.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"success": true, "msg": "OK"})
		})
	}

	log.Printf("服务启动，端口: %d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatal("启动失败:", err)
	}
}