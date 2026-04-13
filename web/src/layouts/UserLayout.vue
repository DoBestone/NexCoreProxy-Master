<template>
  <a-layout class="user-layout">
    <!-- 顶部导航 -->
    <a-layout-header class="header">
      <div class="header-inner">
        <!-- Logo -->
        <div class="logo" @click="$router.push('/user/nodes')">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" stroke-width="2"/>
              <path d="M2 17L12 22L22 17" stroke="currentColor" stroke-width="2"/>
              <path d="M2 12L12 17L22 12" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <span class="logo-text">NexCore</span>
        </div>
        
        <!-- 桌面端导航 -->
        <a-menu
          v-model:selectedKeys="selectedKeys"
          mode="horizontal"
          class="nav-menu hide-mobile"
        >
          <a-menu-item key="nodes" @click="$router.push('/user/nodes')">
            <template #icon><CloudServerOutlined /></template>
            <span>我的节点</span>
          </a-menu-item>
          <a-menu-item key="buy" @click="$router.push('/user/buy')">
            <template #icon><ShoppingCartOutlined /></template>
            <span>购买套餐</span>
          </a-menu-item>
          <a-menu-item key="traffic" @click="$router.push('/user/traffic')">
            <template #icon><LineChartOutlined /></template>
            <span>流量统计</span>
          </a-menu-item>
          <a-menu-item key="orders" @click="$router.push('/user/orders')">
            <template #icon><ShoppingOutlined /></template>
            <span>我的订单</span>
          </a-menu-item>
          <a-menu-item key="tickets" @click="$router.push('/user/tickets')">
            <template #icon><MessageOutlined /></template>
            <span>我的工单</span>
          </a-menu-item>
          <a-menu-item key="settings" @click="$router.push('/user/settings')">
            <template #icon><SettingOutlined /></template>
            <span>账户设置</span>
          </a-menu-item>
        </a-menu>
        
        <!-- 右侧操作区 -->
        <div class="header-right">
          <a-tag color="green" class="role-tag hide-mobile">
            <template #icon><UserOutlined /></template>
            用户端
          </a-tag>
          
          <a-dropdown placement="bottomRight">
            <div class="user-info">
              <a-avatar :size="32" class="user-avatar">
                {{ username.charAt(0).toUpperCase() }}
              </a-avatar>
              <span class="user-name">{{ username }}</span>
            </div>
            <template #overlay>
              <a-menu>
                <a-menu-item key="logout" @click="handleLogout">
                  <LogoutOutlined />
                  <span style="margin-left: 8px">退出登录</span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
          
          <!-- 移动端菜单按钮 -->
          <button class="mobile-menu-btn hide-desktop" @click="mobileMenuVisible = true">
            <MenuOutlined />
          </button>
        </div>
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
    
    <!-- 页脚 -->
    <a-layout-footer class="footer">
      <div class="footer-content">
        <p>NexCore © 2026 · 安全稳定的网络代理解决方案</p>
      </div>
    </a-layout-footer>
    
    <!-- 移动端抽屉菜单 -->
    <a-drawer
      v-model:open="mobileMenuVisible"
      placement="left"
      :width="280"
      class="mobile-drawer"
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
        <a-menu-item key="nodes" @click="navigateTo('/user/nodes')">
          <template #icon><CloudServerOutlined /></template>
          <span>我的节点</span>
        </a-menu-item>
        <a-menu-item key="buy" @click="navigateTo('/user/buy')">
          <template #icon><ShoppingCartOutlined /></template>
          <span>购买套餐</span>
        </a-menu-item>
        <a-menu-item key="traffic" @click="navigateTo('/user/traffic')">
          <template #icon><LineChartOutlined /></template>
          <span>流量统计</span>
        </a-menu-item>
        <a-menu-item key="orders" @click="navigateTo('/user/orders')">
          <template #icon><ShoppingOutlined /></template>
          <span>我的订单</span>
        </a-menu-item>
        <a-menu-item key="tickets" @click="navigateTo('/user/tickets')">
          <template #icon><MessageOutlined /></template>
          <span>我的工单</span>
        </a-menu-item>
        <a-menu-item key="settings" @click="navigateTo('/user/settings')">
          <template #icon><SettingOutlined /></template>
          <span>账户设置</span>
        </a-menu-item>
      </a-menu>
    </a-drawer>
    
    <!-- 移动端悬浮按钮 -->
    <button class="mobile-fab hide-desktop" @click="mobileMenuVisible = true">
      <MenuOutlined />
    </button>
  </a-layout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  CloudServerOutlined,
  ShoppingCartOutlined,
  LineChartOutlined,
  ShoppingOutlined,
  MessageOutlined,
  UserOutlined,
  LogoutOutlined,
  MenuOutlined,
  SettingOutlined
} from '@ant-design/icons-vue'
import { logout } from '@/api'

const router = useRouter()
const route = useRoute()
const mobileMenuVisible = ref(false)
const selectedKeys = ref(['nodes'])
const username = ref(localStorage.getItem('user_username') || 'user')

onMounted(() => {
  const path = route.path.split('/')[2] || 'nodes'
  selectedKeys.value = [path]
})

const navigateTo = (path) => {
  router.push(path)
  mobileMenuVisible.value = false
}

const handleLogout = async () => {
  try {
    await logout()
  } catch (e) {}
  localStorage.removeItem('user_token')
  localStorage.removeItem('user_username')
  message.success('已退出登录')
  router.push('/user/login')
}
</script>

<style scoped>
.user-layout {
  min-height: 100vh;
  background: linear-gradient(180deg, #f8fafc 0%, #eff6ff 100%);
  display: flex;
  flex-direction: column;
}

/* 顶部导航 */
.header {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  position: sticky;
  top: 0;
  z-index: 100;
  padding: 0;
  height: auto;
  line-height: normal;
  flex-shrink: 0;
}

.header-inner {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
  height: 64px;
  display: flex;
  align-items: center;
  gap: 32px;
}

/* Logo */
.logo {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  flex-shrink: 0;
  transition: transform 0.2s ease;
}

.logo:hover {
  transform: scale(1.02);
}

.logo-icon {
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 12px;
  color: white;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.25);
}

.logo-icon svg {
  width: 20px;
  height: 20px;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #3b82f6;
  letter-spacing: -0.5px;
}

/* 导航菜单 */
.nav-menu {
  flex: 1;
  border: none !important;
  background: transparent !important;
}

.nav-menu :deep(.ant-menu-item) {
  border-radius: 8px;
  margin: 0 4px;
  font-weight: 500;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.nav-menu :deep(.ant-menu-item:hover) {
  color: #3b82f6;
  background: #f0f7ff;
}

.nav-menu :deep(.ant-menu-item-selected) {
  color: #3b82f6 !important;
  background: linear-gradient(135deg, #eff6ff 0%, #d6e8ff 100%) !important;
  border-bottom: none !important;
}

.nav-menu :deep(.ant-menu-item-selected::after) {
  display: none;
}

/* 右侧操作区 */
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-left: auto;
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
  background: linear-gradient(135deg, #0891b2 0%, #22d3ee 100%);
}

.user-name {
  font-weight: 500;
  color: #1e293b;
}

/* 移动端菜单按钮 */
.mobile-menu-btn {
  width: 40px;
  height: 40px;
  display: none;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  color: #5f6368;
  font-size: 18px;
  transition: all 0.2s ease;
}

.mobile-menu-btn:hover {
  background: #eff6ff;
  color: #3b82f6;
}

/* 内容区域 */
.content {
  padding: 0;
  flex: 1;
}

.content-wrapper {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  padding-bottom: 80px;
  width: 100%;
}

/* 页脚 */
.footer {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(8px);
  text-align: center;
  padding: 20px 24px;
  border-top: 1px solid #e2e8f0;
  flex-shrink: 0;
}

.footer p {
  color: #64748b;
  font-size: 13px;
  margin: 0;
}

/* 移动端悬浮按钮 */
.mobile-fab {
  position: fixed;
  bottom: 24px;
  left: 24px;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  border: none;
  color: white;
  font-size: 22px;
  cursor: pointer;
  box-shadow: 0 4px 16px rgba(59, 130, 246, 0.3);
  display: none;
  align-items: center;
  justify-content: center;
  z-index: 90;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.mobile-fab:hover {
  transform: scale(1.05);
  box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
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
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 10px;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
}

.drawer-logo .logo-icon svg {
  width: 18px;
  height: 18px;
}

.mobile-nav {
  border: none !important;
}

.mobile-nav :deep(.ant-menu-item) {
  height: 48px;
  line-height: 48px;
  margin: 4px 0;
  border-radius: 10px;
  font-weight: 500;
}

/* 响应式 */
@media (max-width: 992px) {
  .header-inner {
    padding: 0 16px;
    gap: 16px;
  }
  
  .nav-menu {
    display: none;
  }
  
  .mobile-menu-btn {
    display: flex;
  }
  
  .mobile-fab {
    display: flex;
  }
  
  .content-wrapper {
    padding: 16px;
  }
}

@media (max-width: 576px) {
  .logo-text {
    display: none;
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
  
  .footer {
    padding: 16px;
  }
  
  .mobile-fab {
    bottom: 16px;
    left: 16px;
    width: 50px;
    height: 50px;
    font-size: 20px;
  }
}
</style>