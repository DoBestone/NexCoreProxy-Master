<template>
  <a-layout class="admin-layout">
    <!-- 侧边栏 -->
    <a-layout-sider
      v-model:collapsed="collapsed"
      :trigger="null"
      collapsible
      breakpoint="lg"
      theme="light"
      class="sider"
      :width="240"
      :collapsedWidth="72"
    >
      <!-- Logo -->
      <div class="logo" :class="{ collapsed: collapsed }">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </div>
        <span v-if="!collapsed" class="logo-text">NexCore</span>
      </div>
      
      <!-- 导航菜单 -->
      <div class="menu-wrapper">
        <a-menu
          v-model:selectedKeys="selectedKeys"
          mode="inline"
          class="nav-menu"
        >
          <a-menu-item key="dashboard" @click="$router.push('/admin/dashboard')">
            <template #icon><DashboardOutlined /></template>
            <span>仪表盘</span>
          </a-menu-item>
          <a-menu-item key="nodes" @click="$router.push('/admin/nodes')">
            <template #icon><CloudServerOutlined /></template>
            <span>服务器管理</span>
          </a-menu-item>
          <a-menu-item key="relay-rules" @click="$router.push('/admin/relay-rules')">
            <template #icon><SwapOutlined /></template>
            <span>中转规则</span>
          </a-menu-item>
          <a-menu-item key="users" @click="$router.push('/admin/users')">
            <template #icon><TeamOutlined /></template>
            <span>用户管理</span>
          </a-menu-item>
          <a-menu-item key="packages" @click="$router.push('/admin/packages')">
            <template #icon><AppstoreOutlined /></template>
            <span>套餐管理</span>
          </a-menu-item>
          <a-menu-item key="orders" @click="$router.push('/admin/orders')">
            <template #icon><ShoppingOutlined /></template>
            <span>订单管理</span>
          </a-menu-item>
          <a-menu-item key="tickets" @click="$router.push('/admin/tickets')">
            <template #icon><MessageOutlined /></template>
            <span>工单管理</span>
          </a-menu-item>
          <a-menu-item key="templates" @click="$router.push('/admin/templates')">
            <template #icon><FileTextOutlined /></template>
            <span>服务模板</span>
          </a-menu-item>
          <a-menu-item key="announcements" @click="$router.push('/admin/announcements')">
            <template #icon><NotificationOutlined /></template>
            <span>公告管理</span>
          </a-menu-item>
          <a-menu-item key="email-settings" @click="$router.push('/admin/email-settings')">
            <template #icon><MailOutlined /></template>
            <span>邮件配置</span>
          </a-menu-item>
          <a-menu-item key="settings" @click="$router.push('/admin/settings')">
            <template #icon><SettingOutlined /></template>
            <span>系统设置</span>
          </a-menu-item>
          <a-menu-item key="system-update" @click="$router.push('/admin/system-update')">
            <template #icon><CloudSyncOutlined /></template>
            <span>系统更新</span>
          </a-menu-item>
        </a-menu>
      </div>
      
      <!-- 底部折叠按钮 -->
      <div class="sider-footer">
        <button class="collapse-btn" @click="collapsed = !collapsed">
          <MenuUnfoldOutlined v-if="collapsed" />
          <MenuFoldOutlined v-else />
        </button>
      </div>
    </a-layout-sider>
    
    <!-- 主内容区 -->
    <a-layout class="main-layout">
      <!-- 顶部导航 -->
      <a-layout-header class="header">
        <div class="header-left">
          <!-- 移动端菜单按钮 -->
          <button class="mobile-menu-btn" @click="mobileMenuVisible = true">
            <MenuOutlined />
          </button>
          <div class="breadcrumb">
            <span class="page-title">{{ pageTitle }}</span>
          </div>
        </div>
        
        <div class="header-right">
          <a-tag color="blue" class="role-tag">
            <template #icon><SafetyOutlined /></template>
            管理端
          </a-tag>
          
          <a-dropdown placement="bottomRight">
            <div class="user-info">
              <a-avatar :size="32" class="user-avatar">
                {{ username.charAt(0).toUpperCase() }}
              </a-avatar>
              <span class="user-name">{{ username }}</span>
            </div>
            <template #overlay>
              <a-menu class="user-menu">
                <a-menu-item key="logout" @click="handleLogout">
                  <LogoutOutlined />
                  <span>退出登录</span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>
      
      <!-- 内容区域 -->
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
    
    <!-- 移动端抽屉菜单 -->
    <a-drawer
      v-model:open="mobileMenuVisible"
      placement="left"
      :width="260"
      class="mobile-drawer"
      @close="mobileMenuVisible = false"
    >
      <template #title>
        <div class="drawer-logo">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <span>NexCore</span>
        </div>
      </template>
      <a-menu
        v-model:selectedKeys="selectedKeys"
        mode="inline"
        class="mobile-nav"
      >
        <a-menu-item key="dashboard" @click="navigateTo('/admin/dashboard')">
          <template #icon><DashboardOutlined /></template>
          <span>仪表盘</span>
        </a-menu-item>
        <a-menu-item key="nodes" @click="navigateTo('/admin/nodes')">
          <template #icon><CloudServerOutlined /></template>
          <span>服务器管理</span>
        </a-menu-item>
        <a-menu-item key="relay-rules" @click="navigateTo('/admin/relay-rules')">
          <template #icon><SwapOutlined /></template>
          <span>中转规则</span>
        </a-menu-item>
        <a-menu-item key="users" @click="navigateTo('/admin/users')">
          <template #icon><TeamOutlined /></template>
          <span>用户管理</span>
        </a-menu-item>
        <a-menu-item key="packages" @click="navigateTo('/admin/packages')">
          <template #icon><AppstoreOutlined /></template>
          <span>套餐管理</span>
        </a-menu-item>
        <a-menu-item key="orders" @click="navigateTo('/admin/orders')">
          <template #icon><ShoppingOutlined /></template>
          <span>订单管理</span>
        </a-menu-item>
        <a-menu-item key="tickets" @click="navigateTo('/admin/tickets')">
          <template #icon><MessageOutlined /></template>
          <span>工单管理</span>
        </a-menu-item>
        <a-menu-item key="templates" @click="navigateTo('/admin/templates')">
          <template #icon><FileTextOutlined /></template>
          <span>服务模板</span>
        </a-menu-item>
        <a-menu-item key="announcements" @click="navigateTo('/admin/announcements')">
          <template #icon><NotificationOutlined /></template>
          <span>公告管理</span>
        </a-menu-item>
        <a-menu-item key="email-settings" @click="navigateTo('/admin/email-settings')">
          <template #icon><MailOutlined /></template>
          <span>邮件配置</span>
        </a-menu-item>
        <a-menu-item key="settings" @click="navigateTo('/admin/settings')">
          <template #icon><SettingOutlined /></template>
          <span>系统设置</span>
        </a-menu-item>
        <a-menu-item key="system-update" @click="navigateTo('/admin/system-update')">
          <template #icon><CloudSyncOutlined /></template>
          <span>系统更新</span>
        </a-menu-item>
      </a-menu>
    </a-drawer>
  </a-layout>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  DashboardOutlined,
  CloudServerOutlined,
  TeamOutlined,
  AppstoreOutlined,
  ShoppingOutlined,
  MessageOutlined,
  FileTextOutlined,
  SettingOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  LogoutOutlined,
  SafetyOutlined,
  NotificationOutlined,
  MailOutlined,
  CloudSyncOutlined,
  SwapOutlined
} from '@ant-design/icons-vue'
import { logout } from '@/api'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const mobileMenuVisible = ref(false)
const selectedKeys = ref(['dashboard'])
const username = ref(localStorage.getItem('admin_username') || 'admin')

const pageTitles = {
  dashboard: '仪表盘',
  nodes: '服务器管理',
  users: '用户管理',
  packages: '套餐管理',
  orders: '订单管理',
  tickets: '工单管理',
  templates: '服务模板',
  announcements: '公告管理',
  'email-settings': '邮件配置',
  settings: '系统设置'
}

const pageTitle = computed(() => {
  const key = route.path.split('/')[2] || 'dashboard'
  return pageTitles[key] || '管理后台'
})

onMounted(() => {
  const path = route.path.split('/')[2] || 'dashboard'
  selectedKeys.value = [path]
  
  if (window.innerWidth < 992) {
    collapsed.value = true
  }
})

const navigateTo = (path) => {
  router.push(path)
  mobileMenuVisible.value = false
}

const handleLogout = async () => {
  try {
    await logout()
  } catch (e) {}
  localStorage.removeItem('admin_token')
  localStorage.removeItem('admin_username')
  message.success('已退出登录')
  router.push('/admin/login')
}
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
  background: #f8fafc;
}

/* 侧边栏 */
.sider {
  background: #ffffff !important;
  border-right: 1px solid #eef0f5;
  position: fixed !important;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
  display: flex;
  flex-direction: column;
  height: 100vh;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.04);
}

/* Logo */
.logo {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  margin: 0;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
}

.logo.collapsed {
  padding: 0;
  justify-content: center;
}

.logo-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.logo-icon svg {
  width: 20px;
  height: 20px;
}

.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: white;
  margin-left: 10px;
  letter-spacing: -0.5px;
}

/* 菜单容器 */
.menu-wrapper {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 12px;
}

.menu-wrapper::-webkit-scrollbar {
  width: 4px;
}

.menu-wrapper::-webkit-scrollbar-thumb {
  background: #e0e0e0;
  border-radius: 2px;
}

.menu-wrapper::-webkit-scrollbar-thumb:hover {
  background: #bdbdbd;
}

/* 导航菜单 */
.nav-menu {
  border: none !important;
  background: transparent !important;
}

.nav-menu :deep(.ant-menu-item) {
  height: 48px;
  line-height: 48px;
  margin: 0;
  border-radius: 0 !important;
  color: #5f6368;
  font-weight: 500;
  font-size: 14px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.nav-menu :deep(.ant-menu-item:hover) {
  color: #3b82f6;
  background: #f0f7ff;
}

.nav-menu :deep(.ant-menu-item-selected) {
  color: #3b82f6;
  background: #eff6ff !important;
  font-weight: 600;
  border-left: 3px solid #3b82f6;
}

.nav-menu :deep(.ant-menu-item .anticon) {
  font-size: 16px;
}

/* 侧边栏底部 */
.sider-footer {
  padding: 12px 16px;
  border-top: 1px solid #e2e8f0;
  flex-shrink: 0;
}

.sider-footer .collapse-btn {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #5f6368;
  font-size: 16px;
  transition: all 0.2s ease;
}

.sider-footer .collapse-btn:hover {
  background: #eff6ff;
  color: #3b82f6;
}

/* 主布局 */
.main-layout {
  background: #f8fafc;
  margin-left: 240px;
  transition: margin-left 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.main-layout.collapsed {
  margin-left: 72px;
}

/* 顶部导航 */
.header {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 64px;
  border-bottom: 1px solid #eef0f5;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
  position: sticky;
  top: 0;
  z-index: 99;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mobile-menu-btn {
  display: none;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #5f6368;
  font-size: 18px;
  transition: all 0.2s ease;
}

.mobile-menu-btn:hover {
  background: #eff6ff;
  color: #3b82f6;
}

.page-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f1f1f;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.role-tag {
  padding: 4px 12px;
  border-radius: 20px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 12px 4px 4px;
  border-radius: 24px;
  transition: all 0.2s ease;
}

.user-info:hover {
  background: #f8fafc;
}

.user-avatar {
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
}

.user-name {
  font-weight: 500;
  color: #1f1f1f;
}

/* 内容区域 */
.content {
  padding: 0;
  min-height: calc(100vh - 64px);
}

.content-wrapper {
  padding: 24px;
  padding-bottom: 80px;
  max-width: 1200px;
  margin: 0 auto;
}

/* 移动端抽屉 */
.mobile-drawer :deep(.ant-drawer-header) {
  border-bottom: 1px solid #e2e8f0;
  padding: 16px 24px;
}

.drawer-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 18px;
  font-weight: 700;
  color: #3b82f6;
}

.drawer-logo .logo-icon {
  width: 32px;
  height: 32px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 8px;
  color: white;
}

.mobile-nav {
  border: none !important;
}

.mobile-nav :deep(.ant-menu-item) {
  height: 48px;
  line-height: 48px;
  margin: 4px 0;
  border-radius: 10px;
}

/* 响应式 */
@media (max-width: 992px) {
  .sider {
    display: none;
  }
  
  .main-layout {
    margin-left: 0;
  }
  
  .content-wrapper {
    padding: 16px;
  }
  
  .header {
    padding: 0 16px;
  }
  
  .mobile-menu-btn {
    display: flex;
  }
}

@media (max-width: 576px) {
  .header-right {
    gap: 8px;
  }
  
  .role-tag {
    display: none;
  }
  
  .user-name {
    display: none;
  }
  
  .content-wrapper {
    padding: 12px;
  }
}
</style>