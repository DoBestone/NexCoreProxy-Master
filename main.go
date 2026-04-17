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
	version   = "2.0.1"
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
	var masterURL, agentBinaryURL, xrayVersion, agentTargetVer string
	var dropLegacy bool

	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.IntVar(&port, "port", 9310, "web server port")
	flag.StringVar(&dbHost, "db-host", "localhost", "database host")
	flag.StringVar(&dbPort, "db-port", "3306", "database port")
	flag.StringVar(&dbUser, "db-user", "root", "database user")
	flag.StringVar(&dbPass, "db-pass", "", "database password")
	flag.StringVar(&dbName, "db-name", "nexcore_proxy", "database name")
	flag.StringVar(&masterURL, "master-url", "", "Master 公网地址，agent 用它回连（必填，用于一键部署）")
	flag.StringVar(&agentBinaryURL, "agent-binary-url", "", "ncp-agent 二进制下载 URL，含 ${ARCH} 占位")
	flag.StringVar(&xrayVersion, "xray-version", "1.8.24", "目标 xray-core 版本")
	flag.StringVar(&agentTargetVer, "agent-version", "0.1.0", "目标 ncp-agent 版本")
	flag.BoolVar(&dropLegacy, "drop-legacy", false, "DROP legacy tables (user_nodes/inbound_templates/traffic_logs/relay_rules) and exit")
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

	// 一次性 DROP 老表
	if dropLegacy {
		if err := model.DropLegacyTables(); err != nil {
			log.Fatal("DROP legacy 失败:", err)
		}
		log.Println("Legacy tables dropped")
		return
	}

	// 注入运行时配置（agent 部署/升级流程要读）
	service.SetRuntimeConfig(masterURL, agentBinaryURL, xrayVersion, agentTargetVer)

	// 初始化服务
	services := service.NewServices()
	services.Backup.AttachDBConfig(dbHost, dbPort, dbUser, dbPass, dbName)

	// 初始化管理员账户
	if err := services.User.InitAdmin(); err != nil {
		log.Println("初始化管理员账户:", err)
	}

	// 为历史用户补齐持久订阅令牌（幂等）
	if err := services.User.BackfillSubscribeTokens(); err != nil {
		log.Println("补齐订阅令牌:", err)
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
