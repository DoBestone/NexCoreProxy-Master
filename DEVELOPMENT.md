# NexCoreProxy Master 开发指南

> v2.0 自研 agent 架构。3x-ui 已完全替代，节点端只跑 xray + ncp-agent。

## 仓库结构

```
NexCoreProxy-Master/                  # 本仓库（DoBestone/NexCoreProxy-Master，公开）
├── main.go                           # Master 后端入口
├── internal/
│   ├── model/                        # 数据模型
│   │   ├── model.go                  # User/Node/Package/Order...
│   │   ├── inbound.go                # Inbound/Relay/RelayBinding/PackageInbound
│   │   ├── agent.go                  # NodeConfigVersion/UserTraffic/NodeOnlineIP
│   │   ├── cert.go                   # AcmeAccount/Certificate
│   │   └── etag.go                   # BumpEtag / GetEtag
│   ├── service/
│   │   ├── agent_protocol.go         # /server/config 渲染
│   │   ├── agent_push.go             # /server/push 流量入账 + kicks
│   │   ├── inbound.go                # Inbound CRUD + bump etag
│   │   ├── relay_syncer.go           # RelayBinding → Relay 自动展开
│   │   ├── relay_health.go           # TCP 探测
│   │   ├── subscription.go           # 用户 → ProxyNode 解析
│   │   ├── subscription_render.go    # v2rayN/Clash/sing-box 渲染
│   │   ├── cert.go                   # ACME (lego, Cloudflare)
│   │   ├── provisioner.go            # 默认入站集
│   │   ├── install_agent.go          # SSH 部署 ncp-agent
│   │   ├── install_agent.sh          # 内嵌的节点安装脚本副本
│   │   ├── backup.go / alert.go      # DB 备份 / 节点离线告警
│   │   └── business_cron.go          # 过期/月重置/在线 IP 清理
│   └── handler/                      # Gin 路由 + handler
├── web/                              # Vue 3 前端
│   └── src/views/admin/
│       ├── Inbounds.vue              # 入站管理
│       ├── RelayBindings.vue         # 中转绑定
│       └── Certs.vue                 # 证书管理
├── Agent/                            # 节点端 agent（独立 go module）
│   ├── cmd/ncp-agent/main.go
│   ├── internal/{config,protocol,puller,pusher,statsclient,xrayrender,xraymgr,firewall,upgrade}
│   ├── install-agent.sh              # 节点安装脚本（含 3x-ui 自动卸载）
│   └── systemd/ncp-agent.service
├── build-agent-binaries.sh           # 编 ncp-agent amd64+arm64
├── install.sh                        # Master 一键安装
├── update.sh                         # Master 自动更新（同步下发新 agent 二进制）
├── start.sh / stop.sh                # 本地启停
└── .github/workflows/release.yml     # CI: tag 触发，编 master+agent+前端
```

---

## 端口约定

| 服务 | 默认端口 | 说明 |
|------|----------|------|
| 前端 dev server | **9300** | `web/vite.config.js`，`/api` 反代到 9310 |
| 后端 API | **9310** | `main.go` 默认 |
| MySQL | 127.0.0.1:3306 | DB `nexcore_proxy` |
| 节点 xray API (内部) | 127.0.0.1:10085 | gRPC stats |

---

## 本地开发

### 后端
```bash
cd Master
go run . --db-pass <YourPassword> \
  --master-url http://localhost:9310 \
  --agent-binary-url 'http://localhost:9310/api/binaries/ncp-agent-linux-${ARCH}'
```

### 前端
```bash
cd Master/web
npm install
npm run dev   # http://localhost:9300
```

### 节点 agent（本地调试）
```bash
cd Master/Agent
go run ./cmd/ncp-agent --config /etc/ncp-agent/agent.yaml
```

---

## CI 发版流程

1. 改版本号：`main.go` 的 `version` 常量 + `web/package.json` 的 `version`
2. 提交：
   ```bash
   git add -A && git commit -m "release: v2.x.x — <说明>"
   git push origin main
   git tag v2.x.x && git push origin v2.x.x
   ```
3. CI 自动构建并发布到 [Releases](https://github.com/DoBestone/NexCoreProxy-Master/releases)：
   - `nexcore-master-linux-{amd64,arm64}` — Master 后端
   - `ncp-agent-linux-{amd64,arm64}` — 节点 agent
   - `frontend-dist.tar.gz` — 前端
   - `install-agent.sh` / `update.sh`

---

## 数据流（自研 agent 架构）

### 节点拉配置
```
agent (60s 周期)
  → GET /api/v1/server/config?etag=<cur>
  Authorization: Bearer <node.agent_key>
  
master:
  ├─ 校 token → 反查 Node
  ├─ 比对 etag → 304 / 200
  └─ 200 时构造 AgentConfig:
      - inbounds (NodeID = node.id, enable=true) + 注入 cert PEM
      - users (joins orders→packages→package_inbounds, 排除过期/超额/禁用)
      - relays (RelayNodeID = node.id) + 解析多级中转目标

agent 收到后:
  ├─ xrayrender.Render → xray.json
  ├─ xray test → 校验
  ├─ atomic rename → /usr/local/etc/xray/config.json
  ├─ systemctl restart xray
  ├─ firewall.Reconcile → ufw/firewalld 开端口
  └─ 写缓存 + 更新 etag
```

### 节点推流量
```
agent (60s 周期)
  → xray api stats --reset → 解析 user>>>1@nx>>>traffic>>>uplink
  → POST /api/v1/server/push
     {stats: {1@nx: {up:..., down:...}}, online: {...}, system: {...}}

master:
  ├─ 入账：UPSERT user_traffic (按 user×node×小时桶)
  ├─ users.traffic_used += up+down
  ├─ 计算 kicks（超额/过期/禁用）→ 标 disabled + bump 受影响节点 etag
  └─ 返回 {success: true, kicks: [...], currentEtag: ...}

agent: kicks 非空 → 立即重拉 config（可选，puller 60s 也会赶上）
```

### etag 触发场景
| 操作 | bump 哪些节点 |
|------|---------------|
| Inbound 增删改 | 该 NodeID + 所有指向它的 Relay 的 RelayNodeID |
| Relay/Binding 改 | RelayNodeID（wrap 模式额外 bump BackendNodeID） |
| User UUID/启用变更 | 该用户授权范围内所有 NodeID + Relay 节点 |
| Cert 签发/续期 | 引用该域名的 Inbound 所在节点 |
| Order paid (激活套餐) | 套餐覆盖到的所有节点 |
| RelayHealth 状态翻转 | 该 RelayNodeID |

---

## 数据库迁移

新部署：`AutoMigrate` 自动创建所有新表。

老部署（v1.x → v2.0）：
```bash
# 启动新 master 后 AutoMigrate 会加大部分新列；以下是补漏 SQL（如需）：
mysql nexcore_proxy <<SQL
ALTER TABLE nodes ADD COLUMN role VARCHAR(20) DEFAULT 'backend';
ALTER TABLE nodes ADD COLUMN region VARCHAR(50);
ALTER TABLE nodes ADD COLUMN installed TINYINT(1) DEFAULT 0;
ALTER TABLE nodes ADD COLUMN agent_version VARCHAR(30);
ALTER TABLE users ADD COLUMN uuid VARCHAR(36);
ALTER TABLE users ADD COLUMN trojan_password VARCHAR(64);
ALTER TABLE users ADD COLUMN ss2022_password VARCHAR(64);
ALTER TABLE users ADD COLUMN speed_limit INT DEFAULT 0;
ALTER TABLE users ADD COLUMN device_limit INT DEFAULT 0;
ALTER TABLE users ADD COLUMN reset_day INT DEFAULT 1;
ALTER TABLE users ADD UNIQUE INDEX idx_user_uuid (uuid);
ALTER TABLE packages ADD COLUMN transfer_gb INT DEFAULT 0;
ALTER TABLE packages ADD COLUMN device_limit INT DEFAULT 0;
ALTER TABLE packages ADD COLUMN speed_limit INT DEFAULT 0;
SQL

# 新表由 AutoMigrate 自动建：inbounds / package_inbounds / relays / relay_bindings /
# node_config_versions / user_traffic / node_online_ips / node_events /
# acme_accounts / certificates

# 一次性清理老表（可选）
nexcore-master --db-pass <pwd> --drop-legacy
```

---

## UI 规范

遵循 NexCore 统一 UI 规范（紧凑控制台 + 反 AI 战术），核心：
- 主色 `#3b82f6` dominant，状态用 6px 圆点不用色块
- 表格行高 40-44px，表头 12px / 单元格 13px / padding 10×14
- mono 字体（端口/UUID/域名/地址）+ tabular-nums
- 表单 form-item 间距 14px，label 12px / 26px 高
- 操作列文字按钮，无图标按钮
- 协议 pill 用克制配色（蓝/灰/暖色），不彩虹
- 无 `transition: all`，无 hover scale

---

## 联调流程

1. 启动 master + 前端 dev server
2. UI 添加节点（IP + SSH 凭据） → 点「部署 ncp-agent」
3. 在节点上 `journalctl -u ncp-agent -f` 看 puller 日志
4. UI 创建 Inbound → 节点上 `cat /usr/local/etc/xray/config.json` 验证渲染
5. UI 创建套餐并关联 Inbound
6. 注册用户 → 购买套餐 → 拿订阅链接
7. 客户端导入 → 验证连通

---

## 常见调试

```bash
# 节点端
systemctl status ncp-agent xray
journalctl -u ncp-agent -f
journalctl -u xray -f
ss -tlnp | grep -E '443|8443|8388'

# Master 端
tail -f /opt/nexcoreproxy-master/logs/server.log
mysql nexcore_proxy -e "SELECT * FROM node_config_versions"
mysql nexcore_proxy -e "SELECT * FROM user_traffic ORDER BY bucket_hour DESC LIMIT 10"
```
