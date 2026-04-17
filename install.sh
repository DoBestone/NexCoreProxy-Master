#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
#  NexCoreProxy Master · 交互式安装脚本 v2.0
#  代理节点集中管理平台
#  从 GitHub Releases 下载预编译产物，无需 Go/Node.js 环境
#  URL 和认证由 NexCore 代理自动处理
#  支持: macOS / Linux (amd64 / arm64)
#  数据库: MySQL 8.0
#  可选: Nginx 反向代理 + Let's Encrypt SSL
# ─────────────────────────────────────────────────────────────
set -euo pipefail

# ── 全局常量 ──────────────────────────────────────────────────
GITHUB_REPO="DoBestone/NexCoreProxy-Master"
SERVICE_NAME="nexcoreproxy-master"
BINARY_NAME="nexcore-master"
AGENT_BINARY_NAME="ncp-agent"
DEFAULT_INSTALL_DIR="/opt/nexcoreproxy-master"
DEFAULT_XRAY_VERSION="1.8.24"
DEFAULT_AGENT_VERSION="0.1.0"

# ── 全局变量 ──────────────────────────────────────────────────
OS="" ARCH="" PKG_MGR=""
INSTALL_DIR="" INSTALL_MODE=""
WEB_PORT="" ADMIN_USER="" ADMIN_PASS=""
DB_HOST="" DB_PORT="" DB_NAME="" DB_USER="" DB_PASS=""
JWT_SECRET=""
USE_DOMAIN=false DOMAIN="" USE_NGINX=false USE_SSL=false CERTBOT_EMAIL=""
MYSQL_ROOT_MODE="" MYSQL_ROOT_PASSWORD=""
NEED_INSTALL_MYSQL=false
MASTER_URL="" ALERT_EMAIL=""

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

  step "域名与 SSL（可选）"
  if prompt_yn "是否使用自定义域名？" "n"; then
    USE_DOMAIN=true
    DOMAIN=$(prompt_input "域名（如 proxy.example.com）")
    USE_NGINX=true
    if prompt_yn "是否启用 HTTPS（Let's Encrypt）？" "y"; then
      USE_SSL=true
      CERTBOT_EMAIL=$(prompt_input "用于证书通知的邮箱（可留空）" "")
    fi
  else
    if prompt_yn "是否配置 Nginx 反向代理？" "n"; then
      USE_NGINX=true
    fi
  fi

  # ── Master 公网 URL（节点 ncp-agent 回连用，必须公网可达） ──
  step "节点回连地址（必填）"
  info "节点上的 ncp-agent 周期性向 Master 拉配置 + 推流量，需要一个公网可达的地址"
  local default_master_url=""
  if $USE_SSL; then
    default_master_url="https://${DOMAIN}"
  elif $USE_DOMAIN; then
    default_master_url="http://${DOMAIN}"
  else
    local ip
    ip=$(curl -fsS --max-time 5 https://api.ipify.org 2>/dev/null || echo "")
    [ -n "$ip" ] && default_master_url="http://${ip}:${WEB_PORT}"
  fi
  MASTER_URL=$(prompt_input "Master 公网 URL（节点 agent 回连）" "${default_master_url:-https://master.example.com}")
  ALERT_EMAIL=$(prompt_input "节点离线告警邮箱（留空则用首个 admin 邮箱）" "")

  # 生成随机密钥
  JWT_SECRET=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32 || true)

  divider
  step "配置预览"
  echo -e "  安装目录      ${W}${INSTALL_DIR}${N}"
  echo -e "  Web 端口      ${W}${WEB_PORT}${N}"
  echo -e "  管理员        ${W}${ADMIN_USER}${N} / ${DIM}(已设置)${N}"
  echo -e "  MySQL         ${W}${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}${N}"
  echo -e "  Master URL    ${W}${MASTER_URL}${N}"
  [ -n "$ALERT_EMAIL" ] && echo -e "  告警邮箱      ${W}${ALERT_EMAIL}${N}"
  if $USE_DOMAIN; then
    echo -e "  域名          ${W}${DOMAIN}${N}"
    echo -e "  SSL           ${W}$($USE_SSL && echo '是' || echo '否')${N}"
  fi
  echo ""
  prompt_yn "确认以上配置并开始安装？" "y" || { err "已取消"; exit 1; }
}

# ── 系统检测 ──────────────────────────────────────────────────
detect_system() {
  step "系统检测"
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$OS" in
    linux*)  OS="linux" ;;
    darwin*) OS="darwin" ;;
    *)       err "不支持的系统: $OS"; exit 1 ;;
  esac

  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)             err "不支持的架构: $ARCH"; exit 1 ;;
  esac

  if [ "$OS" = "linux" ]; then
    if command -v apt-get &>/dev/null; then PKG_MGR="apt"
    elif command -v yum &>/dev/null; then PKG_MGR="yum"
    elif command -v dnf &>/dev/null; then PKG_MGR="dnf"
    elif command -v pacman &>/dev/null; then PKG_MGR="pacman"
    fi
  elif [ "$OS" = "darwin" ]; then
    PKG_MGR="brew"
    if ! command -v brew &>/dev/null; then
      warn "未检测到 Homebrew，某些依赖可能需手动安装"
    fi
  fi
  ok "系统: ${OS}/${ARCH}  包管理器: ${PKG_MGR:-未知}"
}

# ── 依赖检测与安装 ────────────────────────────────────────────
check_curl() {
  command -v curl &>/dev/null && return
  info "安装 curl..."
  case "$PKG_MGR" in
    apt)    sudo apt-get update -qq && sudo apt-get install -y curl ;;
    yum)    sudo yum install -y curl ;;
    dnf)    sudo dnf install -y curl ;;
    pacman) sudo pacman -S --noconfirm curl ;;
    brew)   brew install curl ;;
    *)      err "请手动安装 curl"; exit 1 ;;
  esac
  ok "curl 已安装"
}

check_nginx() {
  $USE_NGINX || return 0
  command -v nginx &>/dev/null && { ok "Nginx ✓"; return; }
  info "安装 Nginx..."
  case "$PKG_MGR" in
    apt)    sudo apt-get update -qq && sudo apt-get install -y nginx ;;
    yum)    sudo yum install -y nginx ;;
    dnf)    sudo dnf install -y nginx ;;
    pacman) sudo pacman -S --noconfirm nginx ;;
    brew)   brew install nginx ;;
    *)      err "请手动安装 Nginx"; exit 1 ;;
  esac
  ok "Nginx 已安装"
}

check_certbot() {
  $USE_SSL || return 0
  command -v certbot &>/dev/null && { ok "Certbot ✓"; return; }
  info "安装 Certbot..."
  case "$PKG_MGR" in
    apt)    sudo apt-get install -y certbot python3-certbot-nginx ;;
    yum|dnf) sudo ${PKG_MGR} install -y certbot python3-certbot-nginx ;;
    pacman) sudo pacman -S --noconfirm certbot certbot-nginx ;;
    brew)   brew install certbot ;;
    *)      err "请手动安装 certbot"; exit 1 ;;
  esac
  ok "Certbot 已安装"
}

# ── 重置 MySQL root 密码（忘记密码时使用）─────────────────────
reset_mysql_root_password() {
  step "重置 MySQL root 密码"
  warn "此操作将短暂停止 MySQL 服务（约 10 秒），需要 sudo 权限"
  local new_pass
  new_pass=$(prompt_secret "新的 MySQL root 密码（明文可见，请确认无误）")

  local mysql_svc="mysql"
  sudo systemctl is-active mysqld >/dev/null 2>&1 && mysql_svc="mysqld"

  info "停止 MySQL（${mysql_svc}）..."
  sudo systemctl stop "$mysql_svc" || { err "无法停止 MySQL 服务，请手动操作"; return 1; }

  sudo mkdir -p /var/run/mysqld && sudo chown mysql:mysql /var/run/mysqld 2>/dev/null || true

  info "以无授权模式临时启动 MySQL..."
  sudo mysqld --user=mysql --skip-grant-tables --skip-networking &>/dev/null &
  local BGPID=$!
  sleep 5

  info "写入新密码..."
  local tmp_sql; tmp_sql=$(mktemp)
  cat > "$tmp_sql" <<ENDSQL
FLUSH PRIVILEGES;
ALTER USER 'root'@'localhost' IDENTIFIED WITH caching_sha2_password BY '${new_pass}';
FLUSH PRIVILEGES;
ENDSQL
  local reset_ok=false
  mysql -u root < "$tmp_sql" >/dev/null 2>&1 && reset_ok=true
  rm -f "$tmp_sql"

  sudo kill "$BGPID" 2>/dev/null || true
  sleep 1
  sudo pkill -x mysqld 2>/dev/null || true
  sleep 2

  info "重启 MySQL（${mysql_svc}）..."
  sudo systemctl start "$mysql_svc"
  sleep 3

  if $reset_ok && MYSQL_PWD="$new_pass" mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
    MYSQL_ROOT_MODE="password"
    MYSQL_ROOT_PASSWORD="$new_pass"
    ok "MySQL root 密码已成功重置"
    return 0
  fi

  err "密码重置失败，请参考文档手动处理:"
  info "https://dev.mysql.com/doc/refman/8.0/en/resetting-permissions.html"
  sudo systemctl start "$mysql_svc" 2>/dev/null || true
  return 1
}

prepare_mysql_root_access() {
  if [ "$OS" = "linux" ] && sudo mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
    MYSQL_ROOT_MODE="sudo"
    return 0
  fi

  if mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
    MYSQL_ROOT_MODE="local"
    return 0
  fi

  warn "检测到 MySQL root 需要密码，接下来用于建库和授权"
  local attempts=0
  while [ $attempts -lt 3 ]; do
    local pass
    pass=$(prompt_secret "MySQL root 密码")
    if MYSQL_PWD="$pass" mysql -u root -e "SELECT 1;" >/dev/null 2>&1; then
      MYSQL_ROOT_MODE="password"
      MYSQL_ROOT_PASSWORD="$pass"
      ok "MySQL root 密码验证成功"
      return 0
    fi
    attempts=$((attempts + 1))
    warn "root 密码验证失败 (${attempts}/3)"
  done

  warn "3 次密码均验证失败"
  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
    if prompt_yn "是否尝试自动重置 MySQL root 密码（推荐忘记密码时使用）？" "n"; then
      reset_mysql_root_password && return 0
    fi
  fi
  return 1
}

mysql_root_exec() {
  local sql="$1"
  case "$MYSQL_ROOT_MODE" in
    sudo)     sudo mysql -u root -e "$sql" ;;
    local)    mysql -u root -e "$sql" ;;
    password) MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -u root -e "$sql" ;;
    *)        return 1 ;;
  esac
}

# ── MySQL 安装辅助函数 ────────────────────────────────────────

_do_install_mysql_pkg() {
  echo ""
  info "即将全新安装 MySQL，请先设置 root 密码（明文显示，请确认无误）"
  info "此密码用于后续数据库管理，请牢记或妥善保存"
  local new_root_pass
  new_root_pass=$(prompt_secret "MySQL root 密码")
  echo ""

  info "安装 MySQL..."
  case "$PKG_MGR" in
    apt)
      sudo apt-get update -qq
      sudo DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server mysql-client
      sudo systemctl start mysql
      sudo systemctl enable mysql
      ;;
    yum|dnf)
      sudo ${PKG_MGR} install -y mysql-server
      sudo systemctl start mysqld
      sudo systemctl enable mysqld
      ;;
    pacman)
      sudo pacman -S --noconfirm mariadb
      sudo mariadb-install-db --user=mysql --basedir=/usr --datadir=/var/lib/mysql 2>/dev/null || true
      sudo systemctl start mariadb
      sudo systemctl enable mariadb
      ;;
    brew)
      brew install mysql
      brew services start mysql
      ;;
    *) err "请手动安装 MySQL"; exit 1 ;;
  esac
  ok "MySQL 已安装并启动"

  info "设置 MySQL root 密码..."
  local set_pass_sql
  if sudo mysql -u root -e "SELECT @@version;" 2>/dev/null | grep -qi "mariadb"; then
    set_pass_sql="ALTER USER 'root'@'localhost' IDENTIFIED BY '${new_root_pass}'; FLUSH PRIVILEGES;"
  else
    set_pass_sql="ALTER USER 'root'@'localhost' IDENTIFIED WITH caching_sha2_password BY '${new_root_pass}'; FLUSH PRIVILEGES;"
  fi

  if sudo mysql -u root -e "$set_pass_sql" 2>/dev/null; then
    MYSQL_ROOT_MODE="password"
    MYSQL_ROOT_PASSWORD="$new_root_pass"
    ok "MySQL root 密码设置成功"
    echo -e "  ${DIM}请保存：root 密码 = ${W}${new_root_pass}${N}"
  else
    warn "root 密码设置失败，继续使用 sudo 免密模式（仅当前会话有效）"
    MYSQL_ROOT_MODE="sudo"
  fi
}

_show_manual_sql_hint() {
  warn "请手动在 MySQL 执行以下 SQL 后重启服务："
  echo "  CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4;"
  echo "  CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '<你的密码>';"
  echo "  CREATE USER IF NOT EXISTS '${DB_USER}'@'%'       IDENTIFIED BY '<你的密码>';"
  echo "  GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';"
  echo "  GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';"
  echo "  FLUSH PRIVILEGES;"
}

_ensure_mysql_db() {
  mysql_root_exec "CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null \
    || { err "创建数据库失败"; _show_manual_sql_hint; return 1; }
  ok "数据库 ${DB_NAME} 已就绪"
}

_sync_mysql_password() {
  mysql_root_exec "ALTER USER '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';" 2>/dev/null || true
  mysql_root_exec "ALTER USER '${DB_USER}'@'%'       IDENTIFIED BY '${DB_PASS}';" 2>/dev/null || true
  mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';" 2>/dev/null || true
  mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';" 2>/dev/null || true
  mysql_root_exec "FLUSH PRIVILEGES;" 2>/dev/null || true
  ok "用户 ${DB_USER} 密码已同步，现有数据保留"
}

_ensure_mysql_user() {
  local user_exists
  user_exists=$(mysql_root_exec \
    "SELECT COUNT(*) FROM mysql.user WHERE User='${DB_USER}' AND Host='localhost';" \
    2>/dev/null | grep -E '^[0-9]+$' | tail -1)

  if [ "${user_exists:-0}" -eq 0 ]; then
    mysql_root_exec "CREATE USER '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';" 2>/dev/null || true
    mysql_root_exec "CREATE USER '${DB_USER}'@'%'       IDENTIFIED BY '${DB_PASS}';" 2>/dev/null || true
    mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';" 2>/dev/null || true
    mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';" 2>/dev/null || true
    mysql_root_exec "FLUSH PRIVILEGES;" 2>/dev/null || true
    ok "用户 ${DB_USER} 创建完成"
    return 0
  fi

  local table_count
  table_count=$(mysql_root_exec \
    "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='${DB_NAME}';" \
    2>/dev/null | grep -E '^[0-9]+$' | tail -1)

  echo ""
  warn "用户 ${W}${DB_USER}${N} 已存在于 MySQL 中"
  [ "${table_count:-0}" -gt 0 ] && \
    warn "数据库 ${W}${DB_NAME}${N} 已有 ${table_count} 张数据表"
  echo ""
  echo -e "  ${W}请选择操作：${N}"
  echo -e "  ${G}1${N})  仅同步密码       （保留现有数据，推荐）"
  echo -e "  ${Y}2${N})  更换数据库信息   （重新填写库名 / 用户 / 密码）"
  echo -e "  ${R}3${N})  完全重置         ⚠  将删除 ${W}${DB_NAME}${N} 全部数据，不可恢复！"
  echo ""
  local choice
  read -rp "  选项 [1]: " choice
  choice="${choice:-1}"

  case "$choice" in
    2)
      step "重新设置数据库信息"
      DB_NAME=$(prompt_input "数据库名" "$DB_NAME")
      DB_USER=$(prompt_input "数据库用户" "$DB_USER")
      DB_PASS=$(prompt_secret "数据库密码")
      _ensure_mysql_db
      _ensure_mysql_user
      ;;
    3)
      warn "⚠  即将删除数据库 ${W}${DB_NAME}${N} 全部数据，此操作不可撤销！"
      if prompt_yn "确认继续清空并重建？" "n"; then
        mysql_root_exec "DROP DATABASE IF EXISTS \`${DB_NAME}\`;" 2>/dev/null || true
        mysql_root_exec "DROP USER IF EXISTS '${DB_USER}'@'localhost';" 2>/dev/null || true
        mysql_root_exec "DROP USER IF EXISTS '${DB_USER}'@'%';" 2>/dev/null || true
        mysql_root_exec "CREATE DATABASE \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null
        mysql_root_exec "CREATE USER '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';" 2>/dev/null
        mysql_root_exec "CREATE USER '${DB_USER}'@'%'       IDENTIFIED BY '${DB_PASS}';" 2>/dev/null
        mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';" 2>/dev/null
        mysql_root_exec "GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'%';" 2>/dev/null
        mysql_root_exec "FLUSH PRIVILEGES;" 2>/dev/null
        ok "数据库 ${DB_NAME} 已完全重置"
      else
        info "已取消重置，改为同步密码"
        _sync_mysql_password
      fi
      ;;
    *)
      _sync_mysql_password
      ;;
  esac
}

# ── MySQL 安装入口 ────────────────────────────────────────────
install_mysql() {
  ${NEED_INSTALL_MYSQL} || return 0
  step "MySQL"

  if command -v mysql &>/dev/null; then
    ok "MySQL 已安装"
    if [ "$OS" = "linux" ]; then
      sudo systemctl start mysql 2>/dev/null \
        || sudo systemctl start mysqld 2>/dev/null \
        || sudo systemctl start mariadb 2>/dev/null || true
    fi
  else
    _do_install_mysql_pkg
  fi

  info "配置数据库和用户..."
  if ! prepare_mysql_root_access; then
    _show_manual_sql_hint
    return
  fi

  _ensure_mysql_db || return
  _ensure_mysql_user
}

verify_mysql_connection() {
  step "验证 MySQL 连接"
  if MYSQL_PWD="$DB_PASS" mysql -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" -e "USE \`${DB_NAME}\`;" 2>/dev/null; then
    ok "MySQL 连接成功"
  else
    warn "MySQL 连接失败，请检查配置。安装将继续，但启动时可能报错。"
  fi
}

# ── 停止旧服务 ────────────────────────────────────────────────
_stop_service_if_running() {
  local pid_list
  pid_list=$(pgrep -x "$BINARY_NAME" 2>/dev/null || true)
  if [ -n "$pid_list" ]; then
    info "停止旧的后端进程..."
    echo "$pid_list" | xargs kill 2>/dev/null || true
    sleep 2
    echo "$pid_list" | xargs kill -9 2>/dev/null || true
    ok "旧进程已停止"
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

  local download_url=""
  local asset_name="${BINARY_NAME}-${OS}-${ARCH}"
  download_url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*${asset_name}[^\"]*\"" | head -1 | cut -d'"' -f4 || true)

  if [ -z "$download_url" ]; then
    err "Release 中未找到 ${asset_name}，请检查是否已发布对应平台的二进制"
  fi

  local tmp_bin; tmp_bin=$(mktemp)
  info "下载后端: ${download_url}"
  curl -fsSL "$download_url" -o "$tmp_bin" || err "后端二进制下载失败"
  chmod +x "$tmp_bin"

  _stop_service_if_running
  mkdir -p "${INSTALL_DIR}/bin"
  mv -f "$tmp_bin" "${INSTALL_DIR}/bin/${BINARY_NAME}"
  ok "后端二进制下载完成"

  download_frontend_dist "$release_info"
  download_agent_binaries "$release_info"

  # 下载 update.sh 到安装目录
  local update_url
  update_url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*update\.sh[^\"]*\"" | head -1 | cut -d'"' -f4 || true)
  if [ -n "$update_url" ]; then
    curl -fsSL "$update_url" -o "${INSTALL_DIR}/update.sh"
    chmod +x "${INSTALL_DIR}/update.sh"
    ok "update.sh 已下载到 ${INSTALL_DIR}/update.sh"
  fi

  INSTALL_MODE="预编译二进制"
}

# ── 下载 ncp-agent 二进制（amd64 + arm64，给节点端拉取用） ──
download_agent_binaries() {
  local release_info="$1"
  info "下载 ncp-agent 二进制（amd64 + arm64）..."
  mkdir -p "${INSTALL_DIR}/binaries"

  local arch found=0
  for arch in amd64 arm64; do
    local asset="${AGENT_BINARY_NAME}-linux-${arch}"
    local url
    url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*${asset}[^\"]*\"" | head -1 | cut -d'"' -f4 || true)
    if [ -z "$url" ]; then
      warn "Release 中未找到 ${asset}（节点 install-agent.sh 将无法从本机下载）"
      continue
    fi
    curl -fsSL "$url" -o "${INSTALL_DIR}/binaries/${asset}"
    chmod +x "${INSTALL_DIR}/binaries/${asset}"
    ok "${asset} 下载完成"
    found=$((found+1))
  done

  if [ "$found" = "0" ]; then
    warn "未下载到任何 ncp-agent 二进制；节点端部署将失败"
  fi
}

# ── 下载预编译前端 ────────────────────────────────────────────
download_frontend_dist() {
  local release_info="$1"
  info "下载预编译前端资源..."

  local frontend_url
  frontend_url=$(echo "$release_info" | grep -o "\"browser_download_url\":[[:space:]]*\"[^\"]*frontend-dist\.tar\.gz[^\"]*\"" | head -1 | cut -d'"' -f4 || true)

  if [ -n "$frontend_url" ]; then
    mkdir -p "${INSTALL_DIR}/web/dist"
    info "下载前端: ${frontend_url}"
    curl -fsSL "$frontend_url" -o "/tmp/frontend-dist.tar.gz"
    tar -xzf "/tmp/frontend-dist.tar.gz" -C "${INSTALL_DIR}/web/dist"
    rm -f "/tmp/frontend-dist.tar.gz"
    ok "前端资源下载完成"
  else
    warn "Release 中未找到 frontend-dist.tar.gz，前端未安装"
    warn "请手动下载并解压到 ${INSTALL_DIR}/web/dist"
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

# === v2.0 自研 agent 架构 ===
# 节点 ncp-agent 回连地址（必须公网可达）
MASTER_URL=${MASTER_URL}
# 节点端拉 ncp-agent 二进制的完整 URL；\${ARCH} 占位符由节点侧 install-agent.sh 替换
AGENT_BINARY_URL=${MASTER_URL}/api/binaries/ncp-agent-linux-\${ARCH}
# ncp-agent 二进制下发目录（master 通过 /api/binaries/:name 服务）
NCP_BINARY_DIR=${INSTALL_DIR}/binaries
# 节点离线告警邮箱
ALERT_EMAIL=${ALERT_EMAIL}
# 目标版本（agent / xray 与本值不一致时触发自升级）
XRAY_VERSION=${DEFAULT_XRAY_VERSION}
AGENT_VERSION=${DEFAULT_AGENT_VERSION}
# 数据库备份目录
BACKUP_DIR=${INSTALL_DIR}/backups
EOF
  chmod 600 "${INSTALL_DIR}/.env"
  ok ".env 配置已生成（权限 600）"
}

# ── Nginx 配置 ────────────────────────────────────────────────
setup_nginx() {
  $USE_NGINX || return 0
  step "配置 Nginx"

  local frontend_dist="${INSTALL_DIR}/web/dist"
  if [ ! -d "$frontend_dist" ]; then
    warn "前端 dist 目录不存在: ${frontend_dist}，Nginx 配置可能不完整"
  fi

  if [ "$OS" = "darwin" ]; then
    NGINX_CONF_DIR="/usr/local/etc/nginx/servers"
    mkdir -p "$NGINX_CONF_DIR"
    NGINX_CONF="$NGINX_CONF_DIR/nexcoreproxy-master.conf"
  else
    NGINX_CONF_DIR="/etc/nginx/sites-available"
    sudo mkdir -p "$NGINX_CONF_DIR"
    NGINX_CONF="$NGINX_CONF_DIR/nexcoreproxy-master"
  fi

  if $USE_SSL; then
    _write_nginx_ssl
  else
    _write_nginx_http
  fi

  if [ "$OS" = "linux" ]; then
    sudo ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/nexcoreproxy-master"
    sudo nginx -t && sudo systemctl reload nginx || true
  else
    nginx -t && { brew services restart nginx 2>/dev/null || nginx -s reload; } || true
  fi

  ok "Nginx 配置完成"
}

_nginx_server_name() {
  if $USE_DOMAIN; then
    echo "$DOMAIN"
  else
    echo "_"
  fi
}

_write_nginx_http() {
  info "生成 HTTP 反向代理配置..."
  local server_name; server_name=$(_nginx_server_name)
  local tee_cmd; [ "$OS" = "darwin" ] && tee_cmd="tee" || tee_cmd="sudo tee"
  $tee_cmd "$NGINX_CONF" > /dev/null <<EOF
server {
    listen 80;
    server_name ${server_name};

    add_header X-Frame-Options SAMEORIGIN;
    add_header X-Content-Type-Options nosniff;

    root ${INSTALL_DIR}/web/dist;
    index index.html;

    # 前端路由 (Vue Router history mode)
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API 反向代理
    location /api/ {
        proxy_pass         http://127.0.0.1:${WEB_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host              \$host;
        proxy_set_header   X-Real-IP         \$remote_addr;
        proxy_set_header   X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
    }
}
EOF
  ok "HTTP 配置 → ${NGINX_CONF}"
  if $USE_DOMAIN; then
    warn "SSL 未配置，建议后续运行: sudo certbot --nginx -d ${DOMAIN}"
  fi
}

_write_nginx_ssl() {
  step "申请 SSL 证书（文件验证）"
  info "使用 webroot 文件验证方式"
  prompt_yn "DNS 已解析，80 端口已开放？继续申请？" "y" || {
    warn "跳过 SSL 申请，已配置 HTTP 反向代理"
    USE_SSL=false; _write_nginx_http; return
  }

  local webroot="/var/www/certbot"
  sudo mkdir -p "${webroot}/.well-known/acme-challenge"
  sudo chown -R www-data:www-data "$webroot" 2>/dev/null \
    || sudo chown -R "$(whoami)" "$webroot"

  info "配置临时 Nginx 用于文件验证..."
  local tee_cmd; [ "$OS" = "darwin" ] && tee_cmd="tee" || tee_cmd="sudo tee"
  $tee_cmd "$NGINX_CONF" > /dev/null <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    location /.well-known/acme-challenge/ {
        root ${webroot};
        allow all;
    }

    location / {
      return 301 https://\$server_name\$request_uri;
    }
}
EOF

  if [ "$OS" = "linux" ]; then
    sudo ln -sf "$NGINX_CONF" "/etc/nginx/sites-enabled/nexcoreproxy-master"
    sudo nginx -t && sudo systemctl reload nginx
  else
    nginx -t && (brew services restart nginx 2>/dev/null || nginx -s reload)
  fi
  ok "临时 HTTP 配置已应用"

  local certbot_args=("certonly" "--webroot" "-w" "$webroot" "-d" "$DOMAIN" "--non-interactive" "--agree-tos")
  [ -n "$CERTBOT_EMAIL" ] && certbot_args+=("--email" "$CERTBOT_EMAIL") \
                          || certbot_args+=("--register-unsafely-without-email")

  if sudo certbot "${certbot_args[@]}"; then
    ok "SSL 证书申请成功"
    _write_nginx_ssl_conf
    if [ "$OS" = "linux" ]; then
      ( crontab -l 2>/dev/null || true; echo "0 3 * * * certbot renew --quiet --webroot -w ${webroot} --deploy-hook 'systemctl reload nginx'" ) \
        | sort -u | crontab -
      ok "已添加证书自动续期 Cron"
    fi
  else
    warn "SSL 申请失败，已降级为 HTTP 配置"
    USE_SSL=false; _write_nginx_http
  fi
}

_write_nginx_ssl_conf() {
  local cert_dir="/etc/letsencrypt/live/${DOMAIN}"
  local webroot="/var/www/certbot"
  local tee_cmd; [ "$OS" = "darwin" ] && tee_cmd="tee" || tee_cmd="sudo tee"
  $tee_cmd "$NGINX_CONF" > /dev/null <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    root ${INSTALL_DIR}/web/dist;
    index index.html;

    location /.well-known/acme-challenge/ {
        root ${webroot};
        allow all;
    }

    location /api/ {
        if (\$http_x_forwarded_proto != "https") {
            return 301 https://\$server_name\$request_uri;
        }
        proxy_pass         http://127.0.0.1:${WEB_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host              \$host;
        proxy_set_header   X-Real-IP         \$remote_addr;
        proxy_set_header   X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
    }

    location / {
        if (\$http_x_forwarded_proto != "https") {
            return 301 https://\$server_name\$request_uri;
        }
        try_files \$uri \$uri/ /index.html;
    }
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN};

    ssl_certificate     ${cert_dir}/fullchain.pem;
    ssl_certificate_key ${cert_dir}/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 1d;

    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Frame-Options SAMEORIGIN;
    add_header X-Content-Type-Options nosniff;

    root ${INSTALL_DIR}/web/dist;
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    location /api/ {
        proxy_pass         http://127.0.0.1:${WEB_PORT};
        proxy_http_version 1.1;
        proxy_set_header   Host              \$host;
        proxy_set_header   X-Real-IP         \$remote_addr;
        proxy_set_header   X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
    }
}
EOF
  ok "HTTPS 配置 → ${NGINX_CONF}"

  if [ "$OS" = "linux" ]; then
    sudo nginx -t && sudo systemctl reload nginx
  else
    nginx -t && (brew services restart nginx 2>/dev/null || nginx -s reload)
  fi
}

# ── 系统服务 ──────────────────────────────────────────────────
setup_service() {
  step "系统服务"
  if ! prompt_yn "是否配置为开机自启服务？" "y"; then
    info "跳过，手动启动: cd ${INSTALL_DIR} && source .env && ./bin/${BINARY_NAME}"
    return
  fi

  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
    _setup_systemd
  elif [ "$OS" = "darwin" ]; then
    _setup_launchd
  else
    warn "未找到服务管理器，手动启动: cd ${INSTALL_DIR} && source .env && ./bin/${BINARY_NAME}"
  fi
}

_setup_systemd() {
  local svc="/etc/systemd/system/${SERVICE_NAME}.service"
  sudo tee "$svc" > /dev/null <<EOF
[Unit]
Description=NexCoreProxy Master 代理节点管理平台
After=network.target mysql.service

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/bin/${BINARY_NAME} \\
  --port \${PORT} \\
  --db-host \${DB_HOST} --db-port \${DB_PORT} \\
  --db-user \${DB_USER} --db-pass \${DB_PASS} --db-name \${DB_NAME} \\
  --master-url \${MASTER_URL} \\
  --agent-binary-url \${AGENT_BINARY_URL} \\
  --xray-version \${XRAY_VERSION} \\
  --agent-version \${AGENT_VERSION}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
  sudo systemctl daemon-reload
  sudo systemctl enable  "$SERVICE_NAME"
  sudo systemctl restart "$SERVICE_NAME"
  sleep 2
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    ok "Systemd 服务已启动 (systemctl status ${SERVICE_NAME})"
  else
    warn "服务启动可能需要等待，请检查: journalctl -u ${SERVICE_NAME} -f"
  fi
}

_setup_launchd() {
  local plist="$HOME/Library/LaunchAgents/com.nexcoreproxy-master.plist"
  mkdir -p "$HOME/Library/LaunchAgents"

  cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>             <string>com.nexcoreproxy-master</string>
  <key>ProgramArguments</key>  <array><string>${INSTALL_DIR}/bin/${BINARY_NAME}</string></array>
  <key>WorkingDirectory</key>  <string>${INSTALL_DIR}</string>
  <key>RunAtLoad</key>         <true/>
  <key>KeepAlive</key>         <true/>
  <key>StandardOutPath</key>   <string>${INSTALL_DIR}/logs/server.log</string>
  <key>StandardErrorPath</key> <string>${INSTALL_DIR}/logs/server.log</string>
</dict>
</plist>
EOF
  launchctl unload "$plist" 2>/dev/null || true
  launchctl load   "$plist"
  ok "LaunchAgent 已加载（开机自启）"
}

# ── 完成提示 ──────────────────────────────────────────────────
print_done() {
  local access_url
  if $USE_SSL; then
    access_url="https://${DOMAIN}"
  elif $USE_DOMAIN; then
    access_url="http://${DOMAIN}"
  elif $USE_NGINX; then
    access_url="http://localhost"
  else
    access_url="http://localhost:${WEB_PORT}"
  fi

  echo ""
  echo -e "${G}  ╔════════════════════════════════════════════════════╗${N}"
  echo -e "${G}  ║${N}       ${W}NexCoreProxy Master 安装完成！${N}              ${G}║${N}"
  echo -e "${G}  ╠════════════════════════════════════════════════════╣${N}"
  echo -e "${G}  ║${N}  访问地址    ${C}${access_url}${N}"
  echo -e "${G}  ║${N}  管理账号    ${W}${ADMIN_USER}${N} / ${DIM}(你设置的密码)${N}"
  echo -e "${G}  ║${N}  安装目录    ${INSTALL_DIR}"
  echo -e "${G}  ║${N}  安装方式    ${W}${INSTALL_MODE}${N}"
  echo -e "${G}  ║${N}  数据库      ${W}MySQL${N} (${DB_HOST}:${DB_PORT}/${DB_NAME})"
  echo -e "${G}  ║${N}  配置文件    ${INSTALL_DIR}/.env"
  if $USE_SSL; then
    echo -e "${G}  ║${N}  SSL 证书    ${G}Let's Encrypt（90天自动续期）${N}"
  fi
  echo -e "${G}  ╠════════════════════════════════════════════════════╣${N}"
  if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
    echo -e "${G}  ║${N}  查看日志    ${W}journalctl -u ${SERVICE_NAME} -f${N}"
    echo -e "${G}  ║${N}  重启服务    ${W}systemctl restart ${SERVICE_NAME}${N}"
    echo -e "${G}  ║${N}  停止服务    ${W}systemctl stop ${SERVICE_NAME}${N}"
  elif [ "$OS" = "darwin" ]; then
    echo -e "${G}  ║${N}  查看日志    ${W}tail -f ${INSTALL_DIR}/logs/server.log${N}"
  fi
  echo -e "${G}  ║${N}  更新版本    ${W}cd ${INSTALL_DIR} && bash update.sh${N}"
  if ! $USE_NGINX; then
    echo -e "${G}  ║${N}"
    echo -e "${G}  ║${N}  ${Y}提示: 未配置 Nginx，前端需额外部署${N}"
    echo -e "${G}  ║${N}  ${DIM}前端 dist 目录: ${INSTALL_DIR}/web/dist${N}"
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
  echo -e "   ${W}NexCoreProxy Master${N}  ${DIM}·  代理节点集中管理平台 · 安装程序 v2.0${N}"
  echo ""
}

main() {
  banner
  collect_config
  detect_system
  step "依赖检查"
  check_curl
  check_nginx
  check_certbot
  install_mysql
  verify_mysql_connection
  download_release
  write_config
  setup_nginx
  setup_service
  print_done
}

main "$@"
