<template>
  <a-layout class="user-layout">
    <!-- 桌面端侧边栏 -->
    <aside
      class="sider hide-mobile"
      :class="{ 'is-collapsed': collapsed }"
    >
      <!-- 顶部 Logo -->
      <div class="sider-logo" @click="$router.push('/user/dashboard')">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <span v-if="!collapsed" class="logo-text">NexCore</span>
      </div>

      <!-- 菜单 -->
      <nav class="sider-menu">
        <router-link
          v-for="item in menuItems"
          :key="item.key"
          :to="item.path"
          class="menu-item"
          :class="{ 'is-active': activeKey === item.key }"
          :title="collapsed ? item.label : ''"
        >
          <span class="menu-icon">
            <component :is="item.icon" />
          </span>
          <span v-if="!collapsed" class="menu-label">{{ item.label }}</span>
          <span
            v-if="!collapsed && item.badge"
            class="menu-badge"
          >{{ item.badge }}</span>
        </router-link>
      </nav>

      <!-- 订阅状态卡（展开时显示） -->
      <div v-if="!collapsed" class="sider-widget">
        <div class="widget-top">
          <span class="widget-label">本月流量</span>
          <span class="widget-percent">{{ trafficPercent }}%</span>
        </div>
        <div class="widget-bar">
          <div class="widget-bar-fill" :style="{ width: trafficPercent + '%' }"></div>
        </div>
        <div class="widget-meta">
          已用 <strong>{{ trafficUsed }}</strong> / {{ trafficTotal }}
        </div>
      </div>

      <!-- 折叠按钮 -->
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
          <a-dropdown placement="bottomRight" trigger="click">
            <div class="user-chip">
              <div class="user-avatar">{{ avatarLetter }}</div>
              <div class="user-meta hide-mobile">
                <span class="user-name">{{ username }}</span>
                <span class="user-role">用户账号</span>
              </div>
              <DownOutlined class="user-chip-caret hide-mobile" />
            </div>
            <template #overlay>
              <a-menu class="user-dropdown">
                <a-menu-item key="settings" @click="$router.push('/user/settings')">
                  <SettingOutlined />
                  <span>账户设置</span>
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

      <!-- 内容区 -->
      <a-layout-content class="content">
        <div class="content-wrapper">
          <router-view v-slot="{ Component }">
            <keep-alive :max="10">
              <component :is="Component" :key="$route.path" />
            </keep-alive>
          </router-view>
        </div>
        <footer class="content-footer">
          NexCore © 2026 · 安全稳定的网络代理解决方案
        </footer>
      </a-layout-content>
    </a-layout>

    <!-- 移动端抽屉 -->
    <a-drawer
      v-model:open="mobileMenuVisible"
      placement="left"
      :width="272"
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
          <span class="drawer-title-sub">用户中心</span>
        </div>
      </div>
      <nav class="drawer-menu">
        <a
          v-for="item in menuItems"
          :key="item.key"
          class="drawer-menu-item"
          :class="{ 'is-active': activeKey === item.key }"
          @click="navigateTo(item.path)"
        >
          <span class="menu-icon">
            <component :is="item.icon" />
          </span>
          <span class="menu-label">{{ item.label }}</span>
          <span v-if="item.badge" class="menu-badge">{{ item.badge }}</span>
        </a>
      </nav>
      <div class="drawer-user">
        <div class="user-avatar">{{ avatarLetter }}</div>
        <div class="user-meta">
          <span class="user-name">{{ username }}</span>
          <span class="user-role">用户账号</span>
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
  AppstoreOutlined,
  CloudServerOutlined,
  ShoppingCartOutlined,
  LineChartOutlined,
  ShoppingOutlined,
  SettingOutlined,
  LogoutOutlined,
  MenuOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  DownOutlined
} from '@ant-design/icons-vue'
import { logout } from '@/api'

const router = useRouter()
const route = useRoute()

const collapsed = ref(false)
const mobileMenuVisible = ref(false)
const username = ref(localStorage.getItem('user_username') || 'user')

const menuItems = [
  { key: 'dashboard', path: '/user/dashboard', label: '仪表盘', icon: markRaw(AppstoreOutlined) },
  { key: 'nodes',     path: '/user/nodes',     label: '我的节点', icon: markRaw(CloudServerOutlined) },
  { key: 'buy',       path: '/user/buy',       label: '购买套餐', icon: markRaw(ShoppingCartOutlined) },
  { key: 'traffic',   path: '/user/traffic',   label: '流量统计', icon: markRaw(LineChartOutlined) },
  { key: 'orders',    path: '/user/orders',    label: '我的订单', icon: markRaw(ShoppingOutlined) },
  { key: 'settings',  path: '/user/settings',  label: '账户设置', icon: markRaw(SettingOutlined) }
]

const pageTitleMap = {
  dashboard: { title: '仪表盘',   sub: '账户与订阅总览' },
  nodes:     { title: '我的节点', sub: '订阅链接与节点列表' },
  buy:       { title: '购买套餐', sub: '选择适合的方案开通' },
  traffic:   { title: '流量统计', sub: '近 30 天用量趋势' },
  orders:    { title: '我的订单', sub: '历史购买与续费记录' },
  settings:  { title: '账户设置', sub: '资料、密码与通知' }
}

const activeKey = computed(() => route.path.split('/')[2] || 'dashboard')
const pageTitle = computed(() => (pageTitleMap[activeKey.value]?.title) || '用户中心')
const pageSubtitle = computed(() => pageTitleMap[activeKey.value]?.sub || '')
const avatarLetter = computed(() => (username.value || 'U').charAt(0).toUpperCase())

// 侧边栏流量 widget —— 占位数据，真实接入时替换
const trafficUsed = ref('—')
const trafficTotal = ref('—')
const trafficPercent = ref(0)

onMounted(() => {
  if (window.innerWidth < 1280) collapsed.value = true
})

const navigateTo = (path) => {
  router.push(path)
  mobileMenuVisible.value = false
}

const handleLogout = async () => {
  try { await logout() } catch (e) {}
  localStorage.removeItem('user_token')
  localStorage.removeItem('user_username')
  message.success('已退出登录')
  router.push('/user/login')
}
</script>

<style scoped>
/* ============================================================
   容器
   ============================================================ */
.user-layout {
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

.sider.is-collapsed {
  width: 68px;
}

.sider-logo {
  height: 64px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 20px;
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

.logo-text {
  font-family: var(--font-display, ui-sans-serif, system-ui);
  font-size: 16px;
  font-weight: 700;
  color: #1e293b;
  letter-spacing: -.01em;
}

/* 菜单 */
.sider-menu {
  flex: 1;
  padding: 12px 10px;
  overflow-y: auto;
  overflow-x: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 42px;
  padding: 0 12px;
  margin-bottom: 2px;
  border-radius: 10px;
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
  top: 10px;
  bottom: 10px;
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
  font-size: 17px;
  flex-shrink: 0;
}

.menu-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.menu-badge {
  font-size: 11px;
  padding: 1px 6px;
  background: #fef2f2;
  color: #dc2626;
  border-radius: 20px;
  font-weight: 600;
  line-height: 1.5;
}

.sider.is-collapsed .menu-item {
  justify-content: center;
  padding: 0;
}

.sider.is-collapsed .menu-item.is-active::before {
  left: 0;
}

/* widget */
.sider-widget {
  margin: 12px 14px;
  padding: 14px;
  border-radius: 12px;
  background: linear-gradient(160deg, #eff6ff 0%, #ffffff 100%);
  border: 1px solid #e2e8f0;
}

.widget-top {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 8px;
}

.widget-label { font-size: 11px; color: #64748b; letter-spacing: .02em; }
.widget-percent {
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
  font-variant-numeric: tabular-nums;
}

.widget-bar {
  height: 6px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}

.widget-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  border-radius: 3px;
  transition: width .3s ease;
}

.widget-meta {
  font-size: 11px;
  color: #64748b;
  font-variant-numeric: tabular-nums;
}

.widget-meta strong {
  color: #1e293b;
  font-family: var(--font-mono, ui-monospace, monospace);
  font-weight: 600;
}

/* 底部 */
.sider-footer {
  padding: 10px 14px 14px;
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
   主区 / 顶栏
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

.mobile-menu-btn:hover {
  background: #eff6ff;
  color: #2563eb;
}

.crumbs {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
  min-width: 0;
}

.crumb-title {
  font-family: var(--font-display, ui-sans-serif, system-ui);
  font-size: 16px;
  font-weight: 600;
  color: #0f172a;
  letter-spacing: -.01em;
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
  gap: 10px;
}

.topbar-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 12px;
  border: 1px solid #e2e8f0;
  background: #fff;
  border-radius: 9px;
  color: #475569;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: border-color .15s, color .15s, background-color .15s;
}

.topbar-btn:hover {
  border-color: #dbeafe;
  color: #2563eb;
  background: #f8fbff;
}

.topbar-btn-label { line-height: 1; }

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
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-role {
  font-size: 11px;
  color: #94a3b8;
}

.user-chip-caret {
  font-size: 10px;
  color: #94a3b8;
}

/* ============================================================
   内容
   ============================================================ */
.content {
  padding: 0;
}

.content-wrapper {
  max-width: 1200px;
  margin: 0 auto;
  padding: 18px 24px;
}

.content-footer {
  max-width: 1200px;
  margin: 0 auto;
  padding: 16px 24px 24px;
  color: #94a3b8;
  font-size: 12px;
  text-align: center;
}

/* ============================================================
   移动端抽屉
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
  font-family: var(--font-display, ui-sans-serif, system-ui);
  font-size: 17px;
  font-weight: 700;
  color: #0f172a;
  letter-spacing: -.01em;
}
.drawer-title-sub { font-size: 11px; color: #94a3b8; }

.drawer-menu {
  flex: 1;
  padding: 12px 12px;
  overflow-y: auto;
}

.drawer-menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 48px;
  padding: 0 14px;
  margin-bottom: 2px;
  border-radius: 10px;
  color: #475569;
  font-size: 14px;
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
  .content-wrapper { padding: 14px 14px; }
  .content-footer { padding: 14px; }
  .mobile-menu-btn { display: flex; }
  .hide-mobile { display: none !important; }
  .hide-desktop { display: flex; }
}

@media (max-width: 576px) {
  .topbar { height: 58px; padding: 0 14px; }
  .crumb-sub { display: none; }
  .topbar-btn { display: none; }
  .content-wrapper { padding: 16px 12px; }
  .user-meta { display: none; }
  .user-chip { padding: 4px; }
}
</style>
