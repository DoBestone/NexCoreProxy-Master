# NexCore代理主机

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

基于 x-ui 的多节点网络代理管理系统。

## 一键安装

```bash
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/install.sh)
```

## 核心功能

### 节点管理
- **SSH自动安装**: 添加服务器后一键SSH安装x-ui
- **API管理**: 通过x-ui API管理所有节点
- **状态同步**: 自动同步节点CPU、内存、流量等状态

### 订阅系统
- 用户购买套餐后获得订阅链接
- 订阅链接包含所有节点配置（vmess/vless/trojan/shadowsocks）
- 支持主流客户端（v2rayN、Clash等）

### 套餐销售
- 创建不同套餐（流量、时长、价格）
- 用户购买后自动分配节点
- 支持余额充值

## 功能特性

### 管理端
- 仪表盘 - 系统状态总览
- 节点管理 - 添加、SSH安装、管理多个节点服务器
- 用户管理 - 管理用户账户、余额、流量
- 套餐管理 - 创建销售套餐
- 订单管理 - 查看和处理订单
- 工单管理 - 处理用户工单
- 节点模板 - 配置入站模板
- 系统设置 - 系统配置

### 用户端
- 我的节点 - 查看分配的节点和订阅链接
- 购买套餐 - 浏览和购买套餐
- 我的订单 - 查看订单记录
- 流量统计 - 查看流量使用情况
- 我的工单 - 提交和查看工单

## 技术栈

- **后端**: Go + Gin + GORM + MySQL
- **前端**: Vue 3 + Vite + Ant Design Vue 4
- **架构**: Master-Agent 分布式架构
- **SSH**: golang.org/x/crypto/ssh

## 环境要求

- MySQL 5.7+ 或 MySQL 8.0+
- Go 1.22+ (编译时)

## 手动安装

### 数据库配置

创建数据库：
```sql
CREATE DATABASE nexcore CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 编译

```bash
git clone https://github.com/DoBestone/NexCoreProxy-Master.git
cd NexCoreProxy-Master
go mod tidy
go build -o nexcore .
```

### 配置

创建 `.env` 文件：
```bash
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASS=your_password
DB_NAME=nexcore
PORT=8082
```

### 启动

```bash
./start.sh
```

## 使用流程

### 管理员操作

1. 添加节点服务器
   - 填写服务器IP、SSH信息
   - 点击"SSH安装"自动安装x-ui
   - 安装完成后节点自动上线

2. 配置节点入站
   - 在节点上配置vmess/vless/trojan等协议
   - 创建节点模板方便批量配置

3. 创建套餐
   - 设置流量、时长、价格
   - 启用套餐供用户购买

### 用户操作

1. 注册登录账户
2. 购买套餐
3. 获取订阅链接
4. 导入客户端使用

## 项目结构

```
NexCoreProxy-Master/
├── main.go                 # 后端入口
├── nexcore                 # 编译后的二进制文件
├── install.sh              # 一键安装脚本
├── start.sh                # 启动脚本
├── stop.sh                 # 停止脚本
├── internal/
│   ├── model/              # 数据模型
│   ├── service/            # 业务逻辑
│   │   ├── node.go         # 节点服务（SSH安装、API管理、订阅生成）
│   │   └── user.go         # 用户服务
│   └── handler/            # API处理器
├── web/                    # 前端项目
│   ├── src/                # 源码
│   └── dist/               # 构建产物
└── data/                   # 数据目录
```

## API接口

### 公开接口
- `GET /api/packages` - 套餐列表
- `GET /api/sub/:token` - 订阅链接

### 认证接口
- `POST /api/login` - 登录
- `GET /api/my/subscribe` - 我的订阅

### 管理员接口
- `GET /api/nodes` - 节点列表
- `POST /api/nodes/:id/install` - SSH安装节点
- `POST /api/nodes/:id/sync` - 同步节点状态

## 关联项目

- **NexCoreProxy-Host**: 节点服务端 (基于 x-ui)

## License

MIT