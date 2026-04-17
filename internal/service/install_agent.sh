#!/usr/bin/env bash
#
# install-agent.sh — NexCore ncp-agent 节点端一键部署
#
# 由 Master 通过 SSH 上传到节点 /tmp/install-agent.sh 后执行：
#   bash /tmp/install-agent.sh
#
# 必填环境变量（Master 在执行前 export）：
#   NCP_MASTER_URL      Master 公网地址，如 https://master.example.com
#   NCP_NODE_ID         数字 ID
#   NCP_NODE_TOKEN      节点鉴权 token (= DB 里 nodes.agent_key)
#
# 可选：
#   NCP_AGENT_URL       ncp-agent 二进制下载地址；默认从 Master 的 /api/binaries 拉
#   XRAY_VERSION        默认 1.8.24
#
# 退出码：0=成功；非 0=失败，stderr 含具体原因
set -euo pipefail

log() { echo "[install-agent] $*"; }
die() { echo "[install-agent][error] $*" >&2; exit 1; }

[[ -n "${NCP_MASTER_URL:-}" ]]  || die "NCP_MASTER_URL not set"
[[ -n "${NCP_NODE_ID:-}" ]]     || die "NCP_NODE_ID not set"
[[ -n "${NCP_NODE_TOKEN:-}" ]]  || die "NCP_NODE_TOKEN not set"

XRAY_VERSION="${XRAY_VERSION:-1.8.24}"
# 别用 ${var:-${other}/...\${ARCH}} 嵌套形式 —— bash 解析里的 } 转义有坑会多出 }
if [ -z "${NCP_AGENT_URL:-}" ]; then
  NCP_AGENT_URL="${NCP_MASTER_URL}/api/binaries/ncp-agent-linux-\${ARCH}"
fi

# --- 1. OS / 架构检测 ---
ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
  x86_64|amd64)  ARCH="amd64"; XRAY_ASSET="Xray-linux-64.zip" ;;
  aarch64|arm64) ARCH="arm64"; XRAY_ASSET="Xray-linux-arm64-v8a.zip" ;;
  *) die "unsupported arch: $ARCH_RAW" ;;
esac
log "detected arch=$ARCH"

if ! command -v systemctl >/dev/null 2>&1; then
  die "systemd required (systemctl not found)"
fi

# --- 2. 依赖工具 ---
ensure_pkg() {
  for pkg in "$@"; do
    if ! command -v "$pkg" >/dev/null 2>&1; then
      log "installing $pkg"
      if   command -v apt-get >/dev/null; then DEBIAN_FRONTEND=noninteractive apt-get install -y "$pkg"
      elif command -v yum     >/dev/null; then yum install -y "$pkg"
      elif command -v dnf     >/dev/null; then dnf install -y "$pkg"
      else die "no supported package manager (apt/yum/dnf)"
      fi
    fi
  done
}
ensure_pkg curl unzip ca-certificates

# --- 3. 系统调优 ---
log "applying sysctl tuning"
cat >/etc/sysctl.d/99-nexcore.conf <<'EOF'
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
net.core.somaxconn=65535
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.tcp_fin_timeout=15
net.ipv4.tcp_tw_reuse=1
net.ipv4.tcp_rmem=4096 87380 67108864
net.ipv4.tcp_wmem=4096 65536 67108864
net.core.rmem_max=67108864
net.core.wmem_max=67108864
fs.file-max=1048576
EOF
sysctl --system >/dev/null 2>&1 || true

# --- 4. 时间同步 ---
timedatectl set-ntp true 2>/dev/null || true

# --- 5. 防火墙：默认放行常见端口；后续 ncp-agent 会按需追加 ---
if command -v ufw >/dev/null 2>&1; then
  ufw allow 22/tcp >/dev/null 2>&1 || true
  ufw allow 443/tcp >/dev/null 2>&1 || true
  ufw allow 443/udp >/dev/null 2>&1 || true
  ufw allow 80/tcp  >/dev/null 2>&1 || true
elif command -v firewall-cmd >/dev/null 2>&1; then
  firewall-cmd --permanent --add-port=22/tcp  >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=443/tcp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=443/udp >/dev/null 2>&1 || true
  firewall-cmd --permanent --add-port=80/tcp  >/dev/null 2>&1 || true
  firewall-cmd --reload >/dev/null 2>&1 || true
fi

# --- 6a. 卸载 3x-ui / x-ui (如已存在) ---
# ncp-agent 是 xray 的唯一控制者，必须独占 xray 进程；老 panel 必须先停掉
uninstall_xui() {
  local removed=0
  # 停服务
  for svc in x-ui 3x-ui ncp-api; do
    if systemctl list-unit-files 2>/dev/null | grep -q "^${svc}.service"; then
      log "stopping ${svc}.service"
      systemctl stop "${svc}.service" 2>/dev/null || true
      systemctl disable "${svc}.service" 2>/dev/null || true
      rm -f "/etc/systemd/system/${svc}.service" "/lib/systemd/system/${svc}.service"
      removed=1
    fi
  done
  # 杀残留进程（防止 systemd 没管到的 PID）
  pkill -9 -f "/usr/local/x-ui/x-ui" 2>/dev/null || true
  pkill -9 -f "/usr/local/x-ui/bin/xray-linux" 2>/dev/null || true
  pkill -9 -f "/usr/local/x-ui/ncp-api" 2>/dev/null || true
  # 清二进制 + 数据（数据先备份到 /root/x-ui-backup-<ts>，避免误删）
  if [[ -d /usr/local/x-ui ]]; then
    local backup="/root/x-ui-backup-$(date +%Y%m%d-%H%M%S)"
    log "backing up /usr/local/x-ui → ${backup}"
    mv /usr/local/x-ui "${backup}" || rm -rf /usr/local/x-ui
    removed=1
  fi
  for d in /etc/x-ui /var/log/x-ui; do
    [[ -d "$d" ]] && rm -rf "$d" && removed=1
  done
  # 清菜单脚本
  rm -f /usr/bin/x-ui /usr/bin/3x-ui
  if [[ "$removed" == "1" ]]; then
    systemctl daemon-reload
    log "x-ui / 3x-ui 已卸载（旧数据备份在 /root/x-ui-backup-*）"
  fi
}
uninstall_xui

# --- 6. 安装 xray-core ---
install_xray() {
  if [[ -x /usr/local/bin/xray ]]; then
    local cur
    cur="$(/usr/local/bin/xray version 2>/dev/null | head -1 | awk '{print $2}')"
    if [[ "$cur" == "$XRAY_VERSION" ]]; then
      log "xray $cur already installed"
      return
    fi
    log "upgrading xray $cur → $XRAY_VERSION"
  else
    log "installing xray $XRAY_VERSION"
  fi

  local url tmp
  url="https://github.com/XTLS/Xray-core/releases/download/v${XRAY_VERSION}/${XRAY_ASSET}"
  tmp="$(mktemp -d)"
  curl -fsSL -o "$tmp/xray.zip" "$url" || die "download xray failed: $url"
  unzip -q "$tmp/xray.zip" -d "$tmp/x"
  install -m 0755 "$tmp/x/xray" /usr/local/bin/xray
  install -d /usr/local/share/xray
  [[ -f "$tmp/x/geoip.dat" ]]   && install -m 0644 "$tmp/x/geoip.dat"   /usr/local/share/xray/geoip.dat
  [[ -f "$tmp/x/geosite.dat" ]] && install -m 0644 "$tmp/x/geosite.dat" /usr/local/share/xray/geosite.dat
  rm -rf "$tmp"
}
install_xray

cat >/etc/systemd/system/xray.service <<'EOF'
[Unit]
Description=Xray Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/xray run -config /usr/local/etc/xray/config.json
Restart=on-failure
RestartSec=3s
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

# --- 7. 安装 ncp-agent ---
log "installing ncp-agent"
mkdir -p /etc/ncp-agent /var/lib/ncp-agent /var/log/ncp-agent /usr/local/etc/xray
# 替换 ${ARCH} 占位：bash ${var/pat/repl} 里转义 } 有 parser 坑，直接 sed
url=$(echo "$NCP_AGENT_URL" | sed "s|\${ARCH}|$ARCH|g")
tmp="$(mktemp)"
curl -fsSL -o "$tmp" "$url" || die "download ncp-agent failed: $url"
install -m 0755 "$tmp" /usr/local/bin/ncp-agent
rm -f "$tmp"

# --- 8. 写 agent.yaml ---
cat >/etc/ncp-agent/agent.yaml <<EOF
master_url: ${NCP_MASTER_URL}
node_id: ${NCP_NODE_ID}
node_token: ${NCP_NODE_TOKEN}
xray_bin: /usr/local/bin/xray
xray_config_path: /usr/local/etc/xray/config.json
xray_service: xray
xray_api_port: 10085
cache_dir: /var/lib/ncp-agent
pull_interval: 60
push_interval: 60
log_level: info
manage_firewall: true
EOF
chmod 0640 /etc/ncp-agent/agent.yaml

# --- 9. 写 ncp-agent systemd 单元 ---
cat >/etc/systemd/system/ncp-agent.service <<'EOF'
[Unit]
Description=NexCore ncp-agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/ncp-agent --config /etc/ncp-agent/agent.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576
StandardOutput=append:/var/log/ncp-agent/agent.log
StandardError=append:/var/log/ncp-agent/agent.log

[Install]
WantedBy=multi-user.target
EOF

# --- 10. 写一个最小 xray.json，让 xray 第一次能起来；ncp-agent 上线后会立刻覆盖 ---
if [[ ! -f /usr/local/etc/xray/config.json ]]; then
  cat >/usr/local/etc/xray/config.json <<'EOF'
{
  "log": {"loglevel": "warning"},
  "inbounds": [],
  "outbounds": [{"protocol": "freedom", "tag": "direct"}]
}
EOF
fi

systemctl daemon-reload
systemctl enable --now xray.service ncp-agent.service

log "install complete"
log "  - xray:      $(/usr/local/bin/xray version | head -1)"
log "  - ncp-agent: $(/usr/local/bin/ncp-agent --help 2>&1 | head -1 || true)"
log "  - service status:"
systemctl is-active xray ncp-agent || true
