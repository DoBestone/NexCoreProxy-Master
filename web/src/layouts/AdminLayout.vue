<template>
  <a-layout class="admin-layout">
    <!-- 桌面端侧边栏 -->
    <aside
      class="sider hide-mobile"
      :class="{ 'is-collapsed': collapsed }"
    >
      <!-- Logo -->
      <div class="sider-logo" @click="$router.push('/admin/dashboard')">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <div v-if="!collapsed" class="sider-logo-text">
          <span class="logo-main">NexCore</span>
          <span class="logo-sub">Admin Console</span>
        </div>
      </div>

      <!-- 分组菜单 -->
      <nav class="sider-menu">
        <template v-for="(group, gi) in menuGroups" :key="group.title + gi">
          <div v-if="!collapsed" class="menu-group-title">{{ group.title }}</div>
          <div v-else class="menu-group-sep"></div>

          <router-link
            v-for="item in group.items"
            :key="item.key"
            :to="item.path"
            class="menu-item"
            :class="{ 'is-active': activeKey === item.key }"
            :title="collapsed ? item.label : ''"
          >
            <span class="menu-icon"><component :is="item.icon" /></span>
            <span v-if="!collapsed" class="menu-label">{{ item.label }}</span>
          </router-link>
        </template>
      </nav>

      <!-- 底部 -->
      <div class="sider-footer">
        <button class="collapse-btn" @click="collapsed = !collapsed">
          <MenuUnfoldOutlined v-if="collapsed" />
          <MenuFoldOutlined v-else />
          <span v-if="!collapsed">收起侧栏</span>
        </button>
      </div>
    </aside>

    <!-- 主区 -->
    <a-layout class="main-layout" :class="{ 'is-collapsed': collapsed }">
      <!-- 顶栏 -->
      <header class="topbar">
        <div class="topbar-left">
          <button class="mobile-menu-btn hide-desktop" @click="mobileMenuVisible = true">
            <MenuOutlined />
          </button>
          <div class="crumbs">
            <span class="crumb-title">{{ pageTitle }}</span>
            <span v-if="pageSubtitle" class="crumb-sub">{{ pageSubtitle }}</span>
          </div>
        </div>

        <div class="topbar-right">
          <span class="role-badge hide-mobile">
            <SafetyOutlined />
            <span>管理端</span>
          </span>

          <a-dropdown placement="bottomRight" trigger="click">
            <div class="user-chip">
              <div class="user-avatar">{{ avatarLetter }}</div>
              <div class="user-meta hide-mobile">
                <span class="user-name">{{ username }}</span>
                <span class="user-role">超级管理员</span>
              </div>
              <DownOutlined class="user-chip-caret hide-mobile" />
            </div>
            <template #overlay>
              <a-menu class="user-dropdown">
                <a-menu-item key="settings" @click="$router.push('/admin/settings')">
                  <SettingOutlined />
                  <span>系统设置</span>
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout" @click="handleLogout">
                  <LogoutOutlined />
                  <span>退出登录</span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </header>

      <!-- 内容 -->
      <a-layout-content class="content">
        <div class="content-wrapper">
          <router-view v-slot="{ Component }">
            <keep-alive :max="10">
              <component :is="Component" :key="$route.path" />
            </keep-alive>
          </router-view>
        </div>
      </a-layout-content>
    </a-layout>

    <!-- 移动抽屉 -->
    <a-drawer
      v-model:open="mobileMenuVisible"
      placement="left"
      :width="276"
      :closable="false"
      class="mobile-drawer"
    >
      <div class="drawer-head">
        <div class="logo-icon drawer-logo-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
          </svg>
        </div>
        <div class="drawer-title">
          <span class="drawer-title-main">NexCore</span>
          <span class="drawer-title-sub">Admin Console</span>
        </div>
      </div>

      <nav class="drawer-menu">
        <template v-for="(group, gi) in menuGroups" :key="'d' + group.title + gi">
          <div class="menu-group-title drawer-group-title">{{ group.title }}</div>
          <a
            v-for="item in group.items"
            :key="item.key"
            class="drawer-menu-item"
            :class="{ 'is-active': activeKey === item.key }"
            @click="navigateTo(item.path)"
          >
            <span class="menu-icon"><component :is="item.icon" /></span>
            <span class="menu-label">{{ item.label }}</span>
          </a>
        </template>
      </nav>

      <div class="drawer-user">
        <div class="user-avatar">{{ avatarLetter }}</div>
        <div class="user-meta">
          <span class="user-name">{{ username }}</span>
          <span class="user-role">超级管理员</span>
        </div>
        <button class="drawer-logout" @click="handleLogout">
          <LogoutOutlined />
        </button>
      </div>
    </a-drawer>
  </a-layout>
</template>

<script setup>
import { ref, computed, onMounted, markRaw } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  DashboardOutlined,
  CloudServerOutlined,
  SwapOutlined,
  TeamOutlined,
  AppstoreOutlined,
  ShoppingOutlined,
  NotificationOutlined,
  MailOutlined,
  SettingOutlined,
  CloudSyncOutlined,
  LogoutOutlined,
  MenuOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  DownOutlined,
  SafetyOutlined,
  ApiOutlined,
  SafetyCertificateOutlined
} from '@ant-design/icons-vue'
import { logout } from '@/api'

const router = useRouter()
const route = useRoute()

const collapsed = ref(false)
const mobileMenuVisible = ref(false)
const username = ref(localStorage.getItem('admin_username') || 'admin')

const menuGroups = [
  {
    title: '概览',
    items: [
      { key: 'dashboard', path: '/admin/dashboard', label: '仪表盘', icon: markRaw(DashboardOutlined) }
    ]
  },
  {
    title: '资源',
    items: [
      { key: 'nodes',           path: '/admin/nodes',           label: '服务器管理', icon: markRaw(CloudServerOutlined) },
      { key: 'inbounds',        path: '/admin/inbounds',        label: '入站管理',   icon: markRaw(ApiOutlined) },
      { key: 'relay-bindings',  path: '/admin/relay-bindings',  label: '中转绑定',   icon: markRaw(SwapOutlined) },
      { key: 'certs',           path: '/admin/certs',           label: '证书管理',   icon: markRaw(SafetyCertificateOutlined) }
    ]
  },
  {
    title: '业务',
    items: [
      { key: 'users',    path: '/admin/users',    label: '用户管理', icon: markRaw(TeamOutlined) },
      { key: 'packages', path: '/admin/packages', label: '套餐管理', icon: markRaw(AppstoreOutlined) },
      { key: 'orders',   path: '/admin/orders',   label: '订单管理', icon: markRaw(ShoppingOutlined) }
    ]
  },
  {
    title: '系统',
    items: [
      { key: 'announcements',  path: '/admin/announcements',  label: '公告管理', icon: markRaw(NotificationOutlined) },
      { key: 'email-settings', path: '/admin/email-settings', label: '邮件配置', icon: markRaw(MailOutlined) },
      { key: 'settings',       path: '/admin/settings',       label: '系统设置', icon: markRaw(SettingOutlined) },
      { key: 'system-update',  path: '/admin/system-update',  label: '系统更新', icon: markRaw(CloudSyncOutlined) }
    ]
  }
]

const pageTitleMap = {
  dashboard:        { title: '仪表盘',     sub: '节点、用户与流量总览' },
  nodes:            { title: '服务器管理', sub: '节点部署、状态与入站配置' },
  inbounds:         { title: '入站管理',   sub: '协议入站定义，agent 拉取自动同步到 xray' },
  'relay-bindings': { title: '中转绑定',   sub: 'Relay 节点绑定 Backend，自动展开为转发条目' },
  certs:            { title: '证书管理',   sub: 'ACME 自动签发与续期（Cloudflare DNS-01）' },
  'relay-rules':    { title: '中转规则 (旧)', sub: '已被中转绑定取代，仅做兼容查看' },
  users:            { title: '用户管理',   sub: '账号、流量与到期时间' },
  packages:         { title: '套餐管理',   sub: '价格、流量与协议组合' },
  orders:           { title: '订单管理',   sub: '所有用户订单' },
  announcements:    { title: '公告管理',   sub: '登录页与系统公告' },
  'email-settings': { title: '邮件配置',   sub: 'SMTP 与通知模板' },
  settings:         { title: '系统设置',   sub: '站点、安全与对接参数' },
  'system-update':  { title: '系统更新',   sub: '版本检查与升级' }
}

const activeKey = computed(() => route.path.split('/')[2] || 'dashboard')
const pageTitle = computed(() => pageTitleMap[activeKey.value]?.title || '管理后台')
const pageSubtitle = computed(() => pageTitleMap[activeKey.value]?.sub || '')
const avatarLetter = computed(() => (username.value || 'A').charAt(0).toUpperCase())

onMounted(() => {
  if (window.innerWidth < 1280) collapsed.value = true
})

const navigateTo = (path) => {
  router.push(path)
  mobileMenuVisible.value = false
}

const handleLogout = async () => {
  try { await logout() } catch (e) {}
  localStorage.removeItem('admin_token')
  localStorage.removeItem('admin_username')
  message.success('已退出登录')
  router.push('/admin/login')
}
</script>

<style scoped>
/* ============================================================
   容器
   ============================================================ */
.admin-layout {
  min-height: 100vh;
  background: #f6f8fb;
  display: flex;
}

/* ============================================================
   侧边栏
   ============================================================ */
.sider {
  width: 232px;
  background: #ffffff;
  border-right: 1px solid #eef1f6;
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 100;
  display: flex;
  flex-direction: column;
  transition: width .24s cubic-bezier(.4,0,.2,1);
}

.sider.is-collapsed { width: 68px; }

.sider-logo {
  height: 64px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  cursor: pointer;
  flex-shrink: 0;
  border-bottom: 1px solid #f1f5f9;
}

.sider.is-collapsed .sider-logo {
  padding: 0;
  justify-content: center;
}

.logo-icon {
  width: 32px;
  height: 32px;
  border-radius: 9px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 10px rgba(59,130,246,.24);
  flex-shrink: 0;
}

.logo-icon svg { width: 18px; height: 18px; }

.sider-logo-text { display: flex; flex-direction: column; gap: 1px; line-height: 1.1; min-width: 0; }

.logo-main {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
}

.logo-sub {
  font-family: var(--font-mono);
  font-size: 9.5px;
  color: #94a3b8;
  letter-spacing: .14em;
  text-transform: uppercase;
}

/* 菜单 */
.sider-menu {
  flex: 1;
  padding: 14px 10px 18px;
  overflow-y: auto;
  overflow-x: hidden;
}

.menu-group-title {
  font-size: 10.5px;
  font-weight: 600;
  letter-spacing: .14em;
  color: #94a3b8;
  text-transform: uppercase;
  padding: 12px 14px 6px;
}

.menu-group-title:first-child { padding-top: 4px; }

.menu-group-sep {
  height: 1px;
  background: #f1f5f9;
  margin: 8px 6px;
}

.menu-group-sep:first-child { display: none; }

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 40px;
  padding: 0 12px;
  margin-bottom: 2px;
  border-radius: 9px;
  color: #475569;
  font-size: 13.5px;
  font-weight: 500;
  text-decoration: none;
  transition: background-color .15s, color .15s;
  position: relative;
}

.menu-item:hover {
  background: #f1f5f9;
  color: #1e293b;
}

.menu-item.is-active {
  background: #eff6ff;
  color: #2563eb;
  font-weight: 600;
}

.menu-item.is-active::before {
  content: '';
  position: absolute;
  left: -10px;
  top: 9px;
  bottom: 9px;
  width: 3px;
  background: #3b82f6;
  border-radius: 0 3px 3px 0;
}

.menu-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}

.menu-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sider.is-collapsed .menu-item {
  justify-content: center;
  padding: 0;
}

.sider.is-collapsed .menu-item.is-active::before { left: 0; }

/* 底部 */
.sider-footer {
  padding: 10px 12px 14px;
  border-top: 1px solid #f1f5f9;
}

.collapse-btn {
  width: 100%;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  background: #f8fafc;
  border-radius: 8px;
  cursor: pointer;
  color: #64748b;
  font-size: 12.5px;
  font-weight: 500;
  transition: background-color .15s, color .15s, border-color .15s;
}

.collapse-btn:hover {
  background: #eff6ff;
  color: #2563eb;
  border-color: #dbeafe;
}

/* ============================================================
   主区
   ============================================================ */
.main-layout {
  background: transparent;
  margin-left: 232px;
  min-height: 100vh;
  transition: margin-left .24s cubic-bezier(.4,0,.2,1);
}

.main-layout.is-collapsed { margin-left: 68px; }

.topbar {
  height: 64px;
  background: rgba(255,255,255,.88);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid #eef1f6;
  padding: 0 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  position: sticky;
  top: 0;
  z-index: 50;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.mobile-menu-btn {
  width: 38px;
  height: 38px;
  display: none;
  align-items: center;
  justify-content: center;
  border: none;
  background: #f1f5f9;
  border-radius: 9px;
  color: #334155;
  font-size: 18px;
  cursor: pointer;
  transition: background-color .15s, color .15s;
}

.mobile-menu-btn:hover { background: #eff6ff; color: #2563eb; }

.crumbs { display: flex; flex-direction: column; justify-content: center; gap: 2px; min-width: 0; }

.crumb-title {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -0.01em;
  line-height: 1.2;
}

.crumb-sub {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.2;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.role-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  background: #eff6ff;
  border: 1px solid #dbeafe;
  color: #2563eb;
  font-size: 12px;
  font-weight: 600;
  border-radius: 99px;
  letter-spacing: .02em;
}

.role-badge .anticon { font-size: 12px; }

.user-chip {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 10px 4px 4px;
  border-radius: 999px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: border-color .15s, background-color .15s;
}

.user-chip:hover {
  background: #f8fafc;
  border-color: #e2e8f0;
}

.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 13px;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(59,130,246,.22);
}

.user-meta {
  display: flex;
  flex-direction: column;
  gap: 1px;
  line-height: 1.2;
  min-width: 0;
}

.user-name {
  font-size: 13px;
  font-weight: 600;
  color: #0f172a;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role { font-size: 11px; color: #94a3b8; }

.user-chip-caret { font-size: 10px; color: #94a3b8; }

/* 内容 */
.content { padding: 0; }

.content-wrapper {
  max-width: 1280px;
  margin: 0 auto;
  padding: 18px 24px 48px;
}

/* ============================================================
   抽屉
   ============================================================ */
.mobile-drawer :deep(.ant-drawer-body) {
  padding: 0;
  display: flex;
  flex-direction: column;
}

.drawer-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 20px 18px;
  border-bottom: 1px solid #f1f5f9;
}

.drawer-logo-icon { width: 38px; height: 38px; border-radius: 10px; }
.drawer-logo-icon svg { width: 20px; height: 20px; }

.drawer-title { display: flex; flex-direction: column; gap: 2px; }
.drawer-title-main {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -0.01em;
}
.drawer-title-sub {
  font-family: var(--font-mono);
  font-size: 10px;
  color: #94a3b8;
  letter-spacing: .14em;
  text-transform: uppercase;
}

.drawer-menu {
  flex: 1;
  padding: 8px 12px 12px;
  overflow-y: auto;
}

.drawer-group-title {
  padding: 14px 14px 4px;
}

.drawer-menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 44px;
  padding: 0 14px;
  margin-bottom: 2px;
  border-radius: 9px;
  color: #475569;
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color .15s, color .15s;
}

.drawer-menu-item:hover { background: #f1f5f9; color: #1e293b; }
.drawer-menu-item.is-active {
  background: #eff6ff;
  color: #2563eb;
  font-weight: 600;
}

.drawer-user {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #f1f5f9;
  background: #f8fafc;
}

.drawer-logout {
  margin-left: auto;
  width: 36px;
  height: 36px;
  border: 1px solid #e2e8f0;
  background: #fff;
  border-radius: 9px;
  cursor: pointer;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color .15s, border-color .15s;
}

.drawer-logout:hover { color: #dc2626; border-color: #fecaca; }

/* ============================================================
   响应式
   ============================================================ */
.hide-mobile { display: flex; }
.hide-desktop { display: none; }

@media (max-width: 992px) {
  .sider { display: none !important; }
  .main-layout { margin-left: 0 !important; }
  .topbar { padding: 0 16px; }
  .content-wrapper { padding: 14px 14px 40px; }
  .mobile-menu-btn { display: flex; }
  .hide-mobile { display: none !important; }
  .hide-desktop { display: flex; }
}

@media (max-width: 576px) {
  .topbar { height: 58px; padding: 0 14px; }
  .crumb-sub { display: none; }
  .role-badge { display: none; }
  .content-wrapper { padding: 16px 12px 40px; }
  .user-meta { display: none; }
  .user-chip { padding: 4px; }
}
</style>
