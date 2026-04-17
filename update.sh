#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  NexCoreProxy Master · 自动更新脚本
#  从 GitHub Releases 下载预编译产物
#  URL 和认证由 NexCore 代理自动处理
#  用法:
#    bash update.sh           检查并更新到最新版本
#    bash update.sh --force   强制更新（即使版本相同）
# ─────────────────────────────────────────────────────────────
set -euo pipefail

# 支持 bash <(curl ...) 方式执行
if [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "${BASH_SOURCE[0]}" ] && [[ "${BASH_SOURCE[0]}" != /dev/* ]] && [[ "${BASH_SOURCE[0]}" != /proc/* ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
  SCRIPT_DIR="$PWD"
fi

SERVICE_NAME="nexcoreproxy-master"
GITHUB_REPO="DoBestone/NexCoreProxy-Master"
BINARY_NAME="nexcore-master"
AGENT_BINARY_NAME="ncp-agent"
BINARY="$SCRIPT_DIR/bin/${BINARY_NAME}"
ENV_FILE="$SCRIPT_DIR/.env"
BINARY_DIR_FOR_AGENTS="$SCRIPT_DIR/binaries"

# ── 颜色 ─────────────────────────────────────────────────────
R='\033[0;31m'; G='\033[0;32m'; Y='\033[1;33m'
B='\033[0;34m'; W='\033[1;37m'; DIM='\033[2m'; N='\033[0m'

info()    { echo -e "  ${B}▸${N} $*"; }
ok()      { echo -e "  ${G}✓${N} $*"; }
warn()    { echo -e "  ${Y}⚠${N}  $*"; }
err()     { echo -e "  ${R}✗${N} $*"; exit 1; }

FORCE=false
for arg in "$@"; do
  case "$arg" in
    --force) FORCE=true ;;
  esac
done

# ── 检测平台 ─────────────────────────────────────────────────
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             err "不支持的架构: $ARCH" ;;
esac
ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
info "平台: ${OS}/${ARCH}"

# ── 读取端口 ─────────────────────────────────────────────────
PORT=8080
if [ -f "$ENV_FILE" ]; then
  _port=$(grep -E '^PORT=' "$ENV_FILE" | cut -d'=' -f2 | tr -d '[:space:]' || true)
  [ -n "$_port" ] && PORT=$_port
fi

# ── 当前版本 ─────────────────────────────────────────────────
CURRENT=""
if [ -x "$BINARY" ]; then
  CURRENT=$("$BINARY" -v 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+[^ ]*' | head -1 || true)
fi
[ -z "$CURRENT" ] && CURRENT="unknown"
info "当前版本: ${W}${CURRENT}${N}"

# ── 最新版本 ─────────────────────────────────────────────────
# 使用标准 GitHub API URL，NexCore 代理会自动改写并注入认证
API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
RELEASE_JSON=$(curl -fsSL "$API_URL" 2>/dev/null || true)

LATEST=$(echo "$RELEASE_JSON" | grep '"tag_name"' | grep -o '"v[^"]*"' | tr -d '"' || true)

if [ -z "$LATEST" ]; then
  err "无法获取最新版本，请检查网络连接"
fi
info "最新版本: ${W}${LATEST}${N}"

if [ "$LATEST" = "$CURRENT" ] && [ "$FORCE" = false ]; then
  ok "已是最新版本"
  echo -e "  提示：使用 ${W}bash update.sh --force${N} 强制更新"
  exit 0
fi

# ── 备份 ─────────────────────────────────────────────────────
_backup() {
  local backup_dir="$SCRIPT_DIR/.backup/$(date +%Y%m%d_%H%M%S)"
  mkdir -p "$backup_dir"
  if [ -f "$ENV_FILE" ]; then
    cp "$ENV_FILE" "$backup_dir/.env"
    info "配置已备份 → $backup_dir"
  fi
  if [ -x "$BINARY" ]; then
    cp "$BINARY" "$backup_dir/${BINARY_NAME}.bak"
    info "旧二进制已备份"
  fi
}

# ── 停止服务 ─────────────────────────────────────────────────
_stop_service() {
  info "停止服务..."
  local PID
  PID=$(pgrep -x "$BINARY_NAME" 2>/dev/null | head -1 || true)
  if [ -n "$PID" ]; then
    kill "$PID" 2>/dev/null || true
    for _i in 1 2 3 4 5; do
      kill -0 "$PID" 2>/dev/null || break
      sleep 1
    done
    kill -9 "$PID" 2>/dev/null || true
    ok "服务已停止 (PID: ${PID})"
  else
    warn "服务未运行"
  fi
  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
    sudo systemctl stop "$SERVICE_NAME" --no-block 2>/dev/null || true
  elif [ "$OS" = "darwin" ]; then
    local plist="$HOME/Library/LaunchAgents/com.nexcoreproxy-master.plist"
    if [ -f "$plist" ]; then
      launchctl unload "$plist" 2>/dev/null || true
    fi
  fi
}

# ── 启动服务 ─────────────────────────────────────────────────
_start_service() {
  info "启动服务..."
  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null \
     && systemctl list-unit-files "${SERVICE_NAME}.service" &>/dev/null 2>&1; then
    sudo systemctl restart "$SERVICE_NAME"
    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
      ok "Systemd 服务已启动"
    else
      err "服务启动失败，请检查日志: journalctl -u ${SERVICE_NAME} -n 50"
    fi
  elif [ "$OS" = "darwin" ]; then
    local plist="$HOME/Library/LaunchAgents/com.nexcoreproxy-master.plist"
    if [ -f "$plist" ]; then
      launchctl load "$plist"
      ok "LaunchAgent 已重新加载"
    else
      cd "$SCRIPT_DIR"
      mkdir -p "$SCRIPT_DIR/logs"
      nohup "$BINARY" >> "$SCRIPT_DIR/logs/server.log" 2>&1 &
      local NEW_PID=$!
      sleep 2
      if kill -0 "$NEW_PID" 2>/dev/null; then
        ok "服务已启动 (PID: ${NEW_PID})"
      else
        err "服务启动失败，请检查日志: $SCRIPT_DIR/logs/server.log"
      fi
    fi
  else
    cd "$SCRIPT_DIR"
    mkdir -p "$SCRIPT_DIR/logs"
    nohup "$BINARY" >> "$SCRIPT_DIR/logs/server.log" 2>&1 &
    local NEW_PID=$!
    sleep 2
    if kill -0 "$NEW_PID" 2>/dev/null; then
      ok "服务已启动 (PID: ${NEW_PID})"
    else
      err "服务启动失败"
    fi
  fi
}

# ── 下载更新 ─────────────────────────────────────────────────
_update_from_release() {
  local download_url="" frontend_url=""

  download_url=$(echo "$RELEASE_JSON" \
    | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*${ASSET_NAME}[^\"]*\"" \
    | head -1 | cut -d'"' -f4 || true)
  frontend_url=$(echo "$RELEASE_JSON" \
    | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*frontend-dist\.tar\.gz[^\"]*\"" \
    | head -1 | cut -d'"' -f4 || true)

  if [ -z "$download_url" ]; then
    err "Release ${LATEST} 没有 ${ASSET_NAME} 预编译包"
  fi

  # 下载二进制
  info "下载 ${LATEST} (${ASSET_NAME})..."
  local tmp_binary
  tmp_binary=$(mktemp "${SCRIPT_DIR}/.ncp-update.XXXXXX")

  if ! curl -fL --progress-bar -o "$tmp_binary" "$download_url"; then
    rm -f "$tmp_binary"
    err "下载二进制失败"
  fi
  chmod +x "$tmp_binary"
  ok "二进制下载完成"

  # 替换二进制
  mkdir -p "$SCRIPT_DIR/bin"
  mv -f "$tmp_binary" "$BINARY"
  ok "二进制文件已更新 → ${LATEST}"

  # 下载前端
  if [ -n "$frontend_url" ]; then
    info "下载前端包..."
    local tmp_frontend
    tmp_frontend=$(mktemp "${SCRIPT_DIR}/.frontend-update.XXXXXX")
    if curl -fL --progress-bar -o "$tmp_frontend" "$frontend_url"; then
      local dist_dir="${SCRIPT_DIR}/web/dist"
      mkdir -p "$dist_dir"
      rm -rf "${dist_dir:?}"/*
      tar -xzf "$tmp_frontend" -C "$dist_dir"
      rm -f "$tmp_frontend"
      ok "前端已更新"
    else
      warn "前端下载失败（已跳过，不影响后端）"
      rm -f "$tmp_frontend"
    fi
  else
    warn "Release 中无前端包，跳过前端更新"
  fi

  # 更新 update.sh 自身
  #
  # ⚠️ 关键：必须先写 .new 再 mv 重命名（rename 给新 inode）；
  # 不能 curl 直接 -o update.sh —— bash 还在读这个 fd，
  # 直接覆写 inode 内容会让 bash 后半段 bytes 错位语法崩
  local update_url
  update_url=$(echo "$RELEASE_JSON" \
    | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*update\.sh[^\"]*\"" \
    | head -1 | cut -d'"' -f4 || true)
  if [ -n "$update_url" ]; then
    if curl -fsSL "$update_url" -o "${SCRIPT_DIR}/update.sh.new" 2>/dev/null; then
      chmod +x "${SCRIPT_DIR}/update.sh.new"
      mv -f "${SCRIPT_DIR}/update.sh.new" "${SCRIPT_DIR}/update.sh"
    fi
  fi

  # 下载 ncp-agent 二进制到 binaries/（master 通过 /api/binaries 服务节点）
  mkdir -p "$BINARY_DIR_FOR_AGENTS"
  local arch agent_url
  for arch in amd64 arm64; do
    local asset="${AGENT_BINARY_NAME}-linux-${arch}"
    agent_url=$(echo "$RELEASE_JSON" \
      | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*${asset}[^\"]*\"" \
      | head -1 | cut -d'"' -f4 || true)
    if [ -n "$agent_url" ]; then
      info "下载 ${asset}..."
      if curl -fsSL "$agent_url" -o "${BINARY_DIR_FOR_AGENTS}/${asset}.new"; then
        chmod +x "${BINARY_DIR_FOR_AGENTS}/${asset}.new"
        mv -f "${BINARY_DIR_FOR_AGENTS}/${asset}.new" "${BINARY_DIR_FOR_AGENTS}/${asset}"
        ok "${asset} 已更新"
      else
        warn "${asset} 下载失败"
        rm -f "${BINARY_DIR_FOR_AGENTS}/${asset}.new"
      fi
    fi
  done

  # 同步下发节点安装脚本（管理员手动 SSH 装节点时用）
  local install_agent_url
  install_agent_url=$(echo "$RELEASE_JSON" \
    | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*install-agent\.sh[^\"]*\"" \
    | head -1 | cut -d'"' -f4 || true)
  if [ -n "$install_agent_url" ]; then
    if curl -fsSL "$install_agent_url" -o "${SCRIPT_DIR}/install-agent.sh.new" 2>/dev/null; then
      chmod +x "${SCRIPT_DIR}/install-agent.sh.new"
      mv -f "${SCRIPT_DIR}/install-agent.sh.new" "${SCRIPT_DIR}/install-agent.sh"
    fi
  fi
}

# ── 检查 .env 是否缺 v2.0 字段，缺了就给提示 ──
_check_env_v2_fields() {
  [ -f "$ENV_FILE" ] || return 0
  local missing=()
  for key in MASTER_URL AGENT_BINARY_URL NCP_BINARY_DIR; do
    grep -qE "^${key}=" "$ENV_FILE" || missing+=("$key")
  done
  [ ${#missing[@]} -eq 0 ] && return 0
  echo ""
  warn "检测到 .env 缺少 v2.0 新字段：${missing[*]}"
  warn "为让节点端 ncp-agent 能正常部署/回连，请在 ${ENV_FILE} 末尾追加："
  cat <<EOF
  ${DIM}─────────────────────────────────────────────${N}
  MASTER_URL=https://master.example.com
  AGENT_BINARY_URL=\${MASTER_URL}/api/binaries/ncp-agent-linux-\${ARCH}
  NCP_BINARY_DIR=${SCRIPT_DIR}/binaries
  ALERT_EMAIL=
  XRAY_VERSION=1.8.24
  AGENT_VERSION=0.1.0
  BACKUP_DIR=${SCRIPT_DIR}/backups
  ${DIM}─────────────────────────────────────────────${N}
EOF
  warn "并把 systemd ExecStart 中加上："
  echo -e "  ${DIM}--master-url \${MASTER_URL} --agent-binary-url \${AGENT_BINARY_URL}${N}"
}

# ── 主流程 ────────────────────────────────────────────────────
echo ""
echo -e "  ${B}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo -e "  ${W}NexCoreProxy Master${N}  ${DIM}自动更新${N}"
echo -e "  ${B}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo ""

_backup
_stop_service
_update_from_release
_check_env_v2_fields
_start_service

echo ""
echo -e "  ${G}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo -e "  ${G}✓${N} 更新完成: ${DIM}${CURRENT}${N} → ${W}${LATEST}${N}"
echo -e "  ${DIM}配置备份: ${SCRIPT_DIR}/.backup/${N}"
echo -e "  ${DIM}节点二进制: ${BINARY_DIR_FOR_AGENTS}${N}"
echo -e "  ${G}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
echo ""
