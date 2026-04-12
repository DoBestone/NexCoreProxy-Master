package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"nexcoreproxy-master/internal/handler"
	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"
	"nexcoreproxy-master/web"
)

var (
	version   = "1.1.0"
	buildTime = "unknown"
)

func init() {
	// 同步版本号到 handler 包（供在线更新使用）
	handler.AppVersion = version
}

func main() {
	var showVersion bool
	var port int
	var dbHost, dbPort, dbUser, dbPass, dbName string

	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.IntVar(&port, "port", 8080, "web server port")
	flag.StringVar(&dbHost, "db-host", "localhost", "database host")
	flag.StringVar(&dbPort, "db-port", "3306", "database port")
	flag.StringVar(&dbUser, "db-user", "root", "database user")
	flag.StringVar(&dbPass, "db-pass", "", "database password")
	flag.StringVar(&dbName, "db-name", "nexcore", "database name")
	flag.Parse()

	if showVersion {
		fmt.Printf("NexCoreProxy Master v%s (built: %s)\n", version, buildTime)
		return
	}

	log.Println("Starting NexCoreProxy Master...")

	// MySQL DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	log.Printf("Connecting to database: %s@%s:%s/%s", dbUser, dbHost, dbPort, dbName)

	// 初始化数据库
	if err := model.InitDB(dsn); err != nil {
		log.Fatal("初始化数据库失败:", err)
	}
	log.Println("Database connected")

	// 自动迁移
	if err := model.AutoMigrate(); err != nil {
		log.Fatal("数据库迁移失败:", err)
	}
	log.Println("Database migrated")

	// 初始化服务
	services := service.NewServices()

	// 初始化管理员账户
	if err := services.User.InitAdmin(); err != nil {
		log.Println("初始化管理员账户:", err)
	}

	// 启动定时任务
	services.StartCron()

	// 启动 Web 服务
	server := web.NewServer(port, services)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatal("启动 Web 服务失败:", err)
		}
	}()

	log.Printf("NexCoreProxy Master v%s 启动成功，端口: %d", version, port)

	<-sigCh
	log.Println("正在关闭服务...")
	server.Stop()
	services.StopCron()
	log.Println("服务已关闭")
}
