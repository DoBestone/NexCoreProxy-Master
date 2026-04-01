#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  NexCoreProxy Master · 交互式安装脚本
#  代理节点集中管理平台
#  从 GitHub Releases 下载预编译产物，无需 Go/Node.js 环境
#  URL 和认证由 NexCore 代理自动处理
#  支持: Linux (amd64 / arm64)
#  数据库: MySQL 8.0
# ─────────────────────────────────────────────────────────────
set -euo pipefail

# ── 全局常量 ──────────────────────────────────────────────────
GITHUB_REPO="DoBestone/NexCoreProxy-Master"
SERVICE_NAME="nexcoreproxy-master"
BINARY_NAME="nexcore-master"
DEFAULT_INSTALL_DIR="/opt/nexcoreproxy-master"

# ── 全局变量 ──────────────────────────────────────────────────
OS="" ARCH="" PKG_MGR=""
INSTALL_DIR=""
WEB_PORT="" ADMIN_USER="" ADMIN_PASS=""
DB_HOST="" DB_PORT="" DB_NAME="" DB_USER="" DB_PASS=""
JWT_SECRET=""
MYSQL_ROOT_MODE="" MYSQL_ROOT_PASSWORD=""
NEED_INSTALL_MYSQL=false

# ── 颜色 ─────────────────────────────────────────────────────
R='\033[0;31m' G='\033[0;32m' Y='\033[1;33m' B='\033[0;34m'
C='\033[0;36m' W='\033[1;37m' DIM='\033[2m' N='\033[0m'

info()    { echo -e "  ${C}ℹ${N}  $*"; }
ok()      { echo -e "  ${G}✔${N}  $*"; }
warn()    { echo -e "  ${Y}⚠${N}  $*"; }
err()     { echo -e "  ${R}✖${N}  $*" >&2; exit 1; }
step()    { echo ""; echo -e "  ${B}──${N} ${W}$*${N} ${B}──${N}"; }
divider() { echo -e "  ${DIM}──────────────────────────────────────────${N}"; }

# ── 用户输入辅助 ──────────────────────────────────────────────
prompt_input() {
  local label="$1" default="${2:-}" var
  if [ -n "$default" ]; then
    read -rp "  $label [${default}]: " var
    echo "${var:-$default}"
  else
    while true; do
      read -rp "  $label: " var
      [ -n "$var" ] && break
      warn "不能为空"
    done
    echo "$var"
  fi
}

prompt_secret() {
  local label="$1" var
  while true; do
    read -rp "  $label: " var
    [ -z "$var" ] && { printf "  ${Y}⚠${N}  不能为空\n" >&2; continue; }
    break
  done
  echo "$var"
}

prompt_yn() {
  local label="$1" default="${2:-y}" ans
  read -rp "  $label [${default}]: " ans
  ans="${ans:-$default}"
  [[ "$ans" =~ ^[Yy] ]]
}

# ── 收集配置 ──────────────────────────────────────────────────
collect_config() {
  step "基本配置"
  INSTALL_DIR=$(prompt_input "安装目录" "$DEFAULT_INSTALL_DIR")

  local port_input
  while true; do
    port_input=$(prompt_input "Web 管理端口" "8080")
    if [[ "$port_input" =~ ^[0-9]+$ ]] && [ "$port_input" -ge 1 ] && [ "$port_input" -le 65535 ]; then
      WEB_PORT="$port_input"; break
    fi
    warn "端口范围 1-65535"
  done

  step "管理员账号"
  ADMIN_USER=$(prompt_input "管理员用户名" "admin")
  ADMIN_PASS=$(prompt_secret "管理员密码")

  step "MySQL 数据库"
  info "NexCoreProxy Master 需要 MySQL 数据库（推荐 8.0+）"
  if prompt_yn "本机自动安装 MySQL？" "y"; then
    DB_HOST="127.0.0.1"
    DB_PORT="3306"
    DB_NAME=$(prompt_input "数据库名" "nexcore_proxy")
    DB_USER=$(prompt_input "数据库用户" "nexcore_proxy")
    DB_PASS=$(prompt_secret "数据库密码")
    NEED_INSTALL_MYSQL=true
  else
    info "请输入已有 MySQL 连接信息"
    DB_HOST=$(prompt_input "MySQL 主机" "127.0.0.1")
    DB_PORT=$(prompt_input "MySQL 端口" "3306")
    DB_NAME=$(prompt_input "数据库名" "nexcore_proxy")
    DB_USER=$(prompt_input "数据库用户" "nexcore_proxy")
    DB_PASS=$(prompt_secret "数据库密码")
    NEED_INSTALL_MYSQL=false
  fi

  JWT_SECRET=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32 || true)

  divider
  step "配置预览"
  echo -e "  安装目录      ${W}${INSTALL_DIR}${N}"
  echo -e "  Web 端口      ${W}${WEB_PORT}${N}"
  echo -e "  管理员        ${W}${ADMIN_USER}${N} / ${DIM}(已设置)${N}"
  echo -e "  MySQL         ${W}${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}${N}"
  echo ""
  prompt_yn "确认以上配置并开始安装？" "y" || { err "已取消"; }
}

# ── 系统检测 ──────────────────────────────────────────────────
detect_system() {
  step "系统检测"
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$OS" in
    linux*)  OS="linux" ;;
    darwin*) OS="darwin" ;;
    *)       err "不支持的系统: $OS" ;;
  esac

  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             err "不支持的架构: $ARCH" ;;
  esac

  if [ "$OS" = "linux" ]; then
    if command -v apt-get &>/dev/null; then PKG_MGR="apt"
    elif command -v yum &>/dev/null; then PKG_MGR="yum"
    elif command -v dnf &>/dev/null; then PKG_MGR="dnf"
    fi
  fi
  ok "系统: ${OS}/${ARCH}  包管理器: ${PKG_MGR:-未知}"
}

# ── 依赖安装 ──────────────────────────────────────────────────
install_deps() {
  step "依赖检查"
  if ! command -v curl &>/dev/null; then
    info "安装 curl..."
    case "$PKG_MGR" in
      apt) sudo apt-get update -qq && sudo apt-get install -y curl ;;
      yum) sudo yum install -y curl ;;
      dnf) sudo dnf install -y curl ;;
    esac
  fi
  ok "curl ✓"
}

# ── MySQL ────────────────────────────────────────────────────
mysql_root_exec() {
  local sql="$1"
  case "$MYSQL_ROOT_MODE" in
    sudo)     sudo mysql -u root -e "$sql" ;;
    local)    mysql -u root -e "$sql" ;;
    password) MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -u root -e "$sql" ;;
    *)        return 1 ;;
  esac
}

prepare_mysql_root_access() {
  if [ "$OS" = "linux" ] && sudo mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
    MYSQL_ROOT_MODE="sudo"; return 0
  fi
  if mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
    MYSQL_ROOT_MODE="local"; return 0
  fi
  warn "MySQL root 需要密码"
  local attempts=0
  while [ $attempts -lt 3 ]; do
    local pass
    pass=$(prompt_secret "MySQL root 密码")
    if MYSQL_PWD="$pass" mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
      MYSQL_ROOT_MODE="password"
      MYSQL_ROOT_PASSWORD="$pass"
      ok "root 密码验证成功"
      return 0
    fi
    attempts=$((attempts + 1))
    warn "验证失败 (${attempts}/3)"
  done
  return 1
}

install_mysql() {
  ${NEED_INSTALL_MYSQL} || return 0
  step "MySQL"

  if command -v mysql &>/dev/null; then
    ok "MySQL 已安装"
    sudo systemctl start mysql 2>/dev/null || sudo systemctl start mysqld 2>/dev/null || true
  else
    info "安装 MySQL..."
    case "$PKG_MGR" in
      apt) sudo apt-get update -qq && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server mysql-client ;;
      yum|dnf) sudo ${PKG_MGR} install -y mysql-server ;;
    esac
    sudo systemctl start mysql 2>/dev/null || sudo systemctl start mysqld 2>/dev/null || true
    sudo systemctl enable mysql 2>/dev/null || sudo systemctl enable mysqld 2>/dev/null || true
    ok "MySQL 已安装并启动"
  fi

  if ! prepare_mysql_root_access; then
    warn "无法获取 root 权限，请手动创建数据库和用户"
    return
  fi

  mysql_root_exec "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null || true
  mysql_root_exec "CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';" 2>/dev/null || true
  mysql_root_exec "CREATE USER IF NOT EXISTS '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASS}';" 2>/dev/null || true
  mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';" 2>/dev/null || true
  mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';" 2>/dev/null || true
  mysql_root_exec "FLUSH PRIVILEGES;" 2>/dev/null || true
  ok "数据库 ${DB_NAME} 和用户 ${DB_USER} 已就绪"
}

# ── 停止旧服务 ────────────────────────────────────────────────
stop_old_service() {
  local pids
  pids=$(pgrep -x "$BINARY_NAME" 2>/dev/null || true)
  if [ -n "$pids" ]; then
    info "停止旧进程..."
    echo "$pids" | xargs kill 2>/dev/null || true
    sleep 2
  fi
  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
    sudo systemctl stop "$SERVICE_NAME" --no-block 2>/dev/null || true
  fi
}

# ── 下载预编译产物 ────────────────────────────────────────────
download_release() {
  step "获取 NexCoreProxy Master"
  info "从 GitHub Releases 下载预编译产物..."
  mkdir -p "$INSTALL_DIR"

  # 使用标准 GitHub API URL，NexCore 代理会自动改写并注入认证
  local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
  local release_info
  release_info=$(curl -fsSL "$api_url" 2>/dev/null) || {
    err "无法访问 GitHub Releases，请检查网络连接"
  }

  local asset_name="${BINARY_NAME}-${OS}-${ARCH}"
  local download_url
  download_url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*${asset_name}[^\"]*\"" | head -1 | cut -d'"' -f4 || true)

  if [ -z "$download_url" ]; then
    err "Release 中未找到 ${asset_name}，请检查是否已发布对应平台的二进制"
  fi

  local tmp_bin; tmp_bin=$(mktemp)
  info "下载后端: ${download_url}"
  curl -fsSL "$download_url" -o "$tmp_bin" || err "后端二进制下载失败"
  chmod +x "$tmp_bin"

  stop_old_service
  mkdir -p "${INSTALL_DIR}/bin"
  mv -f "$tmp_bin" "${INSTALL_DIR}/bin/${BINARY_NAME}"
  ok "后端二进制下载完成"

  # 下载前端
  local frontend_url
  frontend_url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*frontend-dist\.tar\.gz[^\"]*\"" | head -1 | cut -d'"' -f4 || true)

  if [ -n "$frontend_url" ]; then
    info "下载前端..."
    mkdir -p "${INSTALL_DIR}/web/dist"
    curl -fsSL "$frontend_url" -o "/tmp/frontend-dist.tar.gz"
    tar -xzf "/tmp/frontend-dist.tar.gz" -C "${INSTALL_DIR}/web/dist"
    rm -f "/tmp/frontend-dist.tar.gz"
    ok "前端资源下载完成"
  else
    warn "Release 中未找到 frontend-dist.tar.gz，前端未安装"
  fi

  # 下载 update.sh
  local update_url
  update_url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*update\.sh[^\"]*\"" | head -1 | cut -d'"' -f4 || true)
  if [ -n "$update_url" ]; then
    curl -fsSL "$update_url" -o "${INSTALL_DIR}/update.sh"
    chmod +x "${INSTALL_DIR}/update.sh"
    ok "update.sh 已下载"
  fi
}

# ── 生成配置 ──────────────────────────────────────────────────
write_config() {
  step "生成配置"
  mkdir -p "${INSTALL_DIR}/data" "${INSTALL_DIR}/logs"

  cat > "${INSTALL_DIR}/.env" <<EOF
# NexCoreProxy Master 配置
PORT=${WEB_PORT}
DB_HOST=${DB_HOST}
DB_PORT=${DB_PORT}
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}
DB_PASS=${DB_PASS}
JWT_SECRET=${JWT_SECRET}
NCP_ADMIN_USER=${ADMIN_USER}
NCP_ADMIN_PASS=${ADMIN_PASS}
EOF
  chmod 600 "${INSTALL_DIR}/.env"
  ok ".env 配置已生成（权限 600）"
}

# ── 系统服务 ──────────────────────────────────────────────────
setup_service() {
  step "系统服务"

  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
    sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" > /dev/null <<EOF
[Unit]
Description=NexCoreProxy Master 代理节点管理平台
After=network.target mysql.service

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/bin/${BINARY_NAME} -port \${PORT} -db-host \${DB_HOST} -db-port \${DB_PORT} -db-user \${DB_USER} -db-pass \${DB_PASS} -db-name \${DB_NAME}
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME"
    sudo systemctl restart "$SERVICE_NAME"
    sleep 2
    if systemctl is-active --quiet "$SERVICE_NAME"; then
      ok "服务已启动 (systemctl status ${SERVICE_NAME})"
    else
      warn "服务启动可能需要等待，请检查: journalctl -u ${SERVICE_NAME} -f"
    fi
  else
    info "非 Linux 系统，手动启动:"
    info "  cd ${INSTALL_DIR} && source .env && ./bin/${BINARY_NAME} -port \$PORT -db-host \$DB_HOST -db-port \$DB_PORT -db-user \$DB_USER -db-pass \"\$DB_PASS\" -db-name \$DB_NAME"
  fi
}

# ── 完成提示 ──────────────────────────────────────────────────
print_done() {
  echo ""
  echo -e "${G}  ╔════════════════════════════════════════════════════╗${N}"
  echo -e "${G}  ║${N}       ${W}NexCoreProxy Master 安装完成！${N}              ${G}║${N}"
  echo -e "${G}  ╠════════════════════════════════════════════════════╣${N}"
  echo -e "${G}  ║${N}  访问地址    ${C}http://<服务器IP>:${WEB_PORT}${N}"
  echo -e "${G}  ║${N}  管理账号    ${W}${ADMIN_USER}${N} / ${DIM}(你设置的密码)${N}"
  echo -e "${G}  ║${N}  安装目录    ${INSTALL_DIR}"
  echo -e "${G}  ║${N}  数据库      ${W}MySQL${N} (${DB_HOST}:${DB_PORT}/${DB_NAME})"
  echo -e "${G}  ║${N}  配置文件    ${INSTALL_DIR}/.env"
  echo -e "${G}  ╠════════════════════════════════════════════════════╣${N}"
  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
  echo -e "${G}  ║${N}  查看日志    ${W}journalctl -u ${SERVICE_NAME} -f${N}"
  echo -e "${G}  ║${N}  重启服务    ${W}systemctl restart ${SERVICE_NAME}${N}"
  echo -e "${G}  ║${N}  停止服务    ${W}systemctl stop ${SERVICE_NAME}${N}"
  echo -e "${G}  ║${N}  更新版本    ${W}cd ${INSTALL_DIR} && bash update.sh${N}"
  fi
  echo -e "${G}  ╚════════════════════════════════════════════════════╝${N}"
  echo ""
}

# ── 主流程 ────────────────────────────────────────────────────
banner() {
  clear 2>/dev/null || true
  echo ""
  echo -e "${B}   ███╗   ██╗ ██████╗██████╗ ${N}"
  echo -e "${B}   ████╗  ██║██╔════╝██╔══██╗${N}"
  echo -e "${B}   ██╔██╗ ██║██║     ██████╔╝${N}"
  echo -e "${B}   ██║╚██╗██║██║     ██╔═══╝ ${N}"
  echo -e "${B}   ██║ ╚████║╚██████╗██║     ${N}"
  echo -e "${B}   ╚═╝  ╚═══╝ ╚═════╝╚═╝     ${N}"
  echo ""
  echo -e "   ${W}NexCoreProxy Master${N}  ${DIM}·  代理节点集中管理平台 · 安装程序${N}"
  echo ""
}

main() {
  banner
  collect_config
  detect_system
  install_deps
  install_mysql
  download_release
  write_config
  setup_service
  print_done
}

main "$@"
