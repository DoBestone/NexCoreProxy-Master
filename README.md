# NexCoreProxy

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/DoBestone/NexCoreProxy-Master)](https://github.com/DoBestone/NexCoreProxy-Master/releases)

自研 agent 架构的多节点代理管理平台。Master 是唯一权威，节点端只跑 xray + 自研 ncp-agent，告别 3x-ui 的双 source-of-truth 困境。

---

## 架构总览

```
┌─────────────────────────────────────────────────┐
│  Master (Go + Vue + MySQL)                      │
│  · users / packages / orders                    │
│  · inbounds / relays / certs                    │
│  · /api/v1/server/{config,push}  (agent 协议)   │
│  · /api/binaries/:name           (agent 下发)   │
└─────────────────────────────────────────────────┘
              ▲                    ▲
              │ pull config (etag) │ push stats (60s)
              │                    │
┌─────────────┴────────────────────┴──────────────┐
│  Node                                            │
│  ┌──────────────┐    ┌────────────────────┐    │
│  │ xray-core    │◄───│ ncp-agent          │    │
│  │              │    │ - 渲染 xray.json   │    │
│  │              │───►│ - 拉 stats gRPC    │    │
│  └──────────────┘    │ - 防火墙端口收敛   │    │
│                      │ - 自动升级（窗口 03-05）│
│                      └────────────────────┘    │
└─────────────────────────────────────────────────┘
```

---

## 一键安装（Master 端）

```bash
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/install.sh)
```

交互式向导会问：
- 安装目录 / Web 端口 / 管理员账号
- MySQL 配置（自动安装或对接已有）
- 域名 + Let's Encrypt SSL（可选）
- **Master 公网 URL**（节点 ncp-agent 回连用，必填）
- 告警邮箱（可选）

安装脚本会下载：
- `nexcore-master-linux-{arch}` — Master 后端
- `frontend-dist.tar.gz` — Vue 前端
- `ncp-agent-linux-{amd64,arm64}` — 节点端二进制（master 通过 `/api/binaries` 服务）
- `update.sh` — 自动更新脚本

---

## 添加节点（任一方式）

### 方式 1：管理后台一键部署
1. UI → 服务器管理 → 添加节点（IP / SSH 凭据）
2. 点「部署 ncp-agent」
3. 自动卸载现有 3x-ui（数据备份到 `/root/x-ui-backup-*`）
4. 装 xray + ncp-agent + systemd 单元
5. agent 回连 master，UI 节点变绿，**SSH 密码自动清除**

### 方式 2：手动 SSH 跑脚本
```bash
ssh root@<node>
NCP_MASTER_URL=https://master.example.com \
NCP_NODE_ID=1 \
NCP_NODE_TOKEN=$(从 nodes.agent_key 拿) \
NCP_AGENT_URL='https://master.example.com/api/binaries/ncp-agent-linux-${ARCH}' \
bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/Agent/install-agent.sh)
```

---

## 核心功能

### 协议支持
| 协议 | 客户端 | 状态 |
|------|--------|------|
| VLESS + Reality (XTLS-Vision) | ✅ | 推荐主力 |
| VLESS + TLS (ws/grpc/h2) | ✅ | CDN 友好 |
| VMess + TLS | ✅ | 老客户端兼容 |
| Trojan + TLS | ✅ | |
| Shadowsocks-2022 | ✅ | 高速 |
| Hysteria2 + 端口跳跃 | ✅ | UDP 高速 |
| TUIC v5 | ✅ | UDP 备选 |

### 节点管理
- **零人工节点上线**：SSH 一键部署，自动卸载旧 3x-ui
- **配置自动同步**：DB 任何变更 → bump etag → 节点 60s 内拉到新配置 → xray reload
- **自动升级**：master 标版本 → 节点凌晨 03-05 自动升级 ncp-agent / xray，失败回滚

### 中转
- **节点级整体绑定**（RelayBinding）：选 Relay + Backend → 自动展开为每条 Inbound 的 Relay
- **transparent 模式**：dokodemo 透传，UUID/协议跟落地完全一致
- **wrap 模式**：协议套壳，relay 暴露成 vless+reality 等伪装协议，outbound 用 trunk 凭据回落地
- **多级中转**：Relay A → Relay B → Backend，递归解析目标
- **自动健康探测**：60s TCP probe，bad 自动从订阅剔除

### 订阅
- 一个 URL 支持多格式：v2rayN base64 / Clash.Meta / sing-box（按 UA 嗅探或 `?type=` 强制）
- 流量信息塞 `Subscription-Userinfo` header，客户端自动显示已用/剩余/到期
- 直连 + 中转条目自动展开（`节点A | inbound` / `中转→节点A | inbound`）

### 套餐
- 套餐 ↔ 入站 多对多关联（不同套餐覆盖不同节点子集）
- 限额：流量 / 时长 / 设备数 / 速度 / 月重置日
- 余额支付即时激活，外部支付管理员确认后自动 bump etag

### ACME 证书
- Cloudflare DNS-01 自动签发
- 入站填 `CertDomain` → master 签发 → 通过 config 推给 agent → xray 自动用上
- 到期 < 30 天自动续，续完 bump 涉及节点

### 自动化运维
- 默认入站集（minimal / standard / full）一键预设
- 防火墙端口自动收敛（ufw / firewalld / iptables）
- 业务定时：过期扫描 / 月流量重置 / 在线 IP 清理
- DB 备份（mysqldump → gzip，保留 7 日 + 4 周 + 12 月）
- 节点离线邮件告警（去重 1h，恢复发"已恢复"）

---

## 自动更新

```bash
cd /opt/nexcoreproxy-master
bash update.sh             # 检查并升级
bash update.sh --force     # 强制重装
```

更新会：
- 备份当前二进制 + .env 到 `.backup/<时间戳>/`
- 拉最新 master 后端、ncp-agent 二进制（amd64+arm64）、前端、scripts
- 检测 .env 是否缺 v2.0 字段，缺了给修复提示
- 重启 systemd 服务

---

## 端口与目录约定

| 项 | 默认 |
|----|------|
| Web 管理端口 | 8080（前端 + API） |
| MySQL | 127.0.0.1:3306 / `nexcore_proxy` |
| 安装目录 | `/opt/nexcoreproxy-master/` |
| 数据目录 | `<install>/data/` |
| 日志目录 | `<install>/logs/` |
| 备份目录 | `<install>/backups/` |
| 节点二进制目录 | `<install>/binaries/` |

节点端：

| 项 | 路径 |
|----|------|
| ncp-agent 二进制 | `/usr/local/bin/ncp-agent` |
| xray 二进制 | `/usr/local/bin/xray` |
| agent 配置 | `/etc/ncp-agent/agent.yaml` |
| xray 配置 | `/usr/local/etc/xray/config.json` |
| 缓存（断网容灾） | `/var/lib/ncp-agent/` |
| 日志 | `/var/log/ncp-agent/` |
| 自动签发证书 | `/usr/local/etc/xray/certs/` |

---

## 技术栈

| 层 | 技术 |
|----|------|
| Master 后端 | Go 1.22+ / Gin / GORM / MySQL |
| Master 前端 | Vue 3 / Vite / Ant Design Vue 4 |
| 节点 agent | Go 1.22+（独立 module，瘦二进制 ~6MB） |
| 节点协议 | xray-core 1.8.24 |
| ACME | go-acme/lego (Cloudflare DNS-01) |
| 证书 / 反代 | Caddy / Nginx + Let's Encrypt |

---

## 手动编译

```bash
# Master 后端
git clone https://github.com/DoBestone/NexCoreProxy-Master.git
cd NexCoreProxy-Master
go mod tidy && go build -o nexcore-master .

# 前端
cd web && npm install && npm run build

# 节点 agent（多架构，输出到 ./binaries）
bash build-agent-binaries.sh
```

---

## API 速查

公开：
- `GET /api/packages` 套餐列表
- `GET /api/sub/:token` 订阅（UA 嗅探 / `?type=clash|singbox|v2rayn`）
- `GET /api/binaries/:name` 二进制下载（agent 安装用）

agent 协议（Bearer = node.agent_key）：
- `GET /api/v1/server/config?etag=` 拉配置（304 可省）
- `POST /api/v1/server/push` 上报流量 + 在线 + 系统快照

管理员：
- `GET /api/inbounds` `POST /api/inbounds`（CRUD + `?nodeId=`）
- `GET /api/relay-bindings` `POST /api/relay-bindings`（CRUD）
- `GET /api/certs` `POST /api/certs/issue` `PUT /api/acme/settings`
- `POST /api/nodes/:id/install-agent` 一键部署
- `POST /api/nodes/:id/provision` 写入预设入站集
- `PUT /api/packages/:id/inbounds` 关联入站

---

## 文档

- [DEPLOY.md](../DEPLOY.md) — 完整部署 + 故障排查
- [DEVELOPMENT.md](DEVELOPMENT.md) — 开发指南

---

## License

MIT
