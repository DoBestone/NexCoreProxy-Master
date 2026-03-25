<template>
  <a-layout class="layout">
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible breakpoint="lg" theme="light">
      <div class="logo">
        <span v-if="!collapsed">NexCore</span>
        <span v-else>NC</span>
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" mode="inline">
        <a-menu-item key="dashboard" @click="$router.push('/admin/dashboard')">
          <template #icon><DashboardOutlined /></template>
          <span>仪表盘</span>
        </a-menu-item>
        <a-menu-item key="nodes" @click="$router.push('/admin/nodes')">
          <template #icon><CloudServerOutlined /></template>
          <span>服务器管理</span>
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
        <a-menu-item key="settings" @click="$router.push('/admin/settings')">
          <template #icon><SettingOutlined /></template>
          <span>系统设置</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="header">
        <menu-unfold-outlined v-if="collapsed" class="trigger" @click="collapsed = !collapsed" />
        <menu-fold-outlined v-else class="trigger" @click="collapsed = !collapsed" />
        <div class="header-right">
          <a-tag color="blue">管理端</a-tag>
          <a-dropdown>
            <a class="ant-dropdown-link" @click.prevent>
              <UserOutlined />
              <span style="margin-left: 8px">{{ username }}</span>
            </a>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="handleLogout">
                  <LogoutOutlined />
                  <span style="margin-left: 8px">退出登录</span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
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
  UserOutlined,
  LogoutOutlined
} from '@ant-design/icons-vue'
import { logout } from '@/api'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const selectedKeys = ref(['dashboard'])
const username = ref(localStorage.getItem('username') || 'admin')

onMounted(() => {
  const path = route.path.split('/')[2] || 'dashboard'
  selectedKeys.value = [path]
})

const handleLogout = async () => {
  try {
    await logout()
  } catch (e) {}
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  message.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
}

.logo {
  height: 48px;
  margin: 16px;
  background: linear-gradient(135deg, #1890ff 0%, #36cfc9 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: bold;
  font-size: 16px;
}

.header {
  background: #fff;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}

.trigger {
  font-size: 18px;
  cursor: pointer;
  color: #666;
}

.trigger:hover {
  color: #1890ff;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.content {
  margin: 24px;
  padding: 24px;
  background: #fff;
  border-radius: 8px;
  min-height: calc(100vh - 112px);
}
</style>