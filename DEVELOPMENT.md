# NexCore代理主机 开发文档

## 项目信息

- **项目名称**: NexCore代理主机
- **项目路径**: `/root/projects/NexCoreProxy-Master` (主控面板)
- **Agent仓库**: https://github.com/DoBestone/NexCoreProxy-Agent (公开)
- **技术栈**: Go + Gin + GORM + Vue 3 + Vite + Ant Design Vue 4

## 一键安装

**Agent (节点服务端):**
```bash
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Agent/main/install.sh) -u admin -pass YourPassword
```

**Master (主控面板):**
```bash
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/install.sh) -pass YourPassword
```

## 数据库配置

- **主机**: 127.0.0.1:3307 (Docker)
- **数据库**: nexcore_proxy
- **用户**: nexcore_proxy / NexCore@2026!
- ⚠️ **注意**: 任何时候都不要使用 `nexcore` 数据库，不要使用 `root` 连接

## 端口配置

| 服务 | 端口 |
|------|------|
| Master 服务 | 8084 |
| ncp-api (Agent) | 54322 |
| MySQL | 3307 |

## 启动方式

```bash
cd /root/projects/NexCoreProxy-Master
./start.sh   # 启动（自动关闭旧进程）
./stop.sh    # 停止
```

---

## 已完成功能

### 后端功能

1. ✅ 数据模型设计 (User, Node, Package, Order, Ticket, InboundTemplate, UserNode, Announcement, EmailConfig)
2. ✅ JWT Token 认证
3. ✅ 后端 API 实现
   - 登录、注册
   - 节点管理（增删改查、SSH安装、状态同步）
   - 用户管理
   - 套餐管理
   - 订单管理
   - 工单系统（创建、回复、关闭）
   - 公告系统
   - 邮件通知
4. ✅ SSH 自动安装节点 (x-ui)
5. ✅ 订阅链接生成 (vmess/vless/trojan/shadowsocks)
6. ✅ 用户购买套餐自动分配节点
7. ✅ Cloudflare Turnstile 人机验证
8. ✅ x-ui 面板反向代理 (通过密钥访问)
9. ✅ Agent API HTTP 客户端 (`agent_api.go`)
10. ✅ 用户端回复工单 API

### 前端功能

#### 管理端
1. ✅ 仪表盘（统计概览）
2. ✅ 节点管理
   - 节点列表（卡片展示、状态、资源监控）
   - SSH 安装
   - API Token 显示与复制
   - 面板访问链接
   - 自动刷新
3. ✅ 用户管理
4. ✅ 套餐管理
5. ✅ 订单管理
6. ✅ 工单管理
7. ✅ 节点模板
8. ✅ 公告管理
9. ✅ 系统设置（邮件配置）

#### 用户端
1. ✅ 我的节点（订阅链接）
2. ✅ 购买套餐
3. ✅ 我的订单
4. ✅ 流量统计（ECharts 图表）
5. ✅ 我的工单（回复功能）
6. ✅ 账户设置（修改密码）

#### 前端优化
1. ✅ 侧边栏固定定位 + 滚动支持
2. ✅ 头部固定 + 毛玻璃效果
3. ✅ 侧边栏去掉圆角，选中左边框高亮
4. ✅ 移动端菜单按钮移到头部
5. ✅ 移除悬浮 FAB 按钮
6. ✅ 根路径跳转用户端登录页

---

## 数据库表结构

### users 用户表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| username | varchar(50) | 用户名 |
| password | varchar(255) | 密码 |
| email | varchar(100) | 邮箱 |
| role | varchar(20) | 角色 (admin/user) |
| balance | float | 余额 |
| traffic_limit | bigint | 流量限制 |
| traffic_used | bigint | 已用流量 |
| expire_at | datetime | 到期时间 |
| enable | bool | 是否启用 |
| invite_code | varchar(20) | 邀请码 |
| invited_by | bigint | 邀请人ID |

### nodes 节点表
| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 主键 |
| name | varchar(100) | 节点名称 |
| ip | varchar(50) | IP地址 |
| port | bigint | 面板端口 |
| username | varchar(50) | 面板用户名 |
| password | varchar(255) | 面板密码 |
| ssh_port | bigint | SSH端口 |
| ssh_user | varchar(50) | SSH用户 |
| ssh_password | varchar(255) | SSH密码 |
| agent_key | varchar(64) | Agent连接密钥 |
| api_token | varchar(255) | ncp-api Token |
| api_port | bigint | ncp-api端口 |
| status | varchar(20) | 状态 |
| cpu | float | CPU使用率 |
| memory | float | 内存使用率 |
| disk | float | 磁盘使用率 |
| xray_version | varchar(20) | Xray版本 |

---

## 开发规范

### 认证系统
- **管理端**: `/admin/login` → `admin_token`
- **用户端**: `/user/login` → `user_token`
- 两端 token 独立存储，可同时登录

### API 路径
- 管理端 API: `/admin/*` (需要 AdminMiddleware)
- 用户端 API: `/my/*` (需要 AuthMiddleware)
- 公开 API: `/api/login`, `/api/register`, `/api/packages`

### 字段命名
- 数据库字段: `api_token`, `api_port`, `memory`
- JSON 返回: `apiToken`, `apiPort`, `memory`
- ⚠️ 不要使用 `mem` 字段名

---

## 待完善功能

1. 节点状态实时同步优化
2. 支付功能集成（支付宝/微信）
3. 流量统计实时更新
4. 节点模板批量应用

---

## 访问地址

- **管理端**: https://ncp.nice07.com/admin/login
- **用户端**: https://ncp.nice07.com/user/login
- **默认账号**: admin / admin123

---

## Cloudflare 缓存

使用 cloudflare-dns skill 清除缓存:

```bash
CF=~/.openclaw/workspace/skills/cloudflare-dns/scripts/cf-dns
$CF -d nice07.com cache-purge
```

---

## 更新记录

### 2026-03-27
- 修复 `api_token` 字段长度问题 (VARCHAR 255)
- 修复 `memory` 字段名问题 (原为 `mem`)
- 修复启动脚本自动关闭旧进程
- 添加编辑节点显示 API Token
- 根路径跳转用户端登录页

### 2026-03-26
- 项目初始化
- 核心功能实现
- 前端页面搭建