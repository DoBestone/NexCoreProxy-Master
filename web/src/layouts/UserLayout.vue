<template>
  <a-layout class="layout">
    <a-layout-header class="header">
      <div class="logo" @click="$router.push('/user/nodes')">
        <span>NexCore代理主机</span>
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" mode="horizontal" theme="light">
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
      </a-menu>
      <div class="header-right">
        <a-tag color="green">用户端</a-tag>
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
    <a-layout-footer class="footer">
      NexCore代理主机 © 2026
    </a-layout-footer>
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
  UserOutlined,
  LogoutOutlined
} from '@ant-design/icons-vue'
import { logout } from '@/api'

const router = useRouter()
const route = useRoute()
const selectedKeys = ref(['nodes'])
const username = ref(localStorage.getItem('username') || 'user')

onMounted(() => {
  const path = route.path.split('/')[2] || 'nodes'
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
  background: #f5f7fa;
}

.header {
  background: #fff;
  padding: 0 24px;
  display: flex;
  align-items: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  position: fixed;
  width: 100%;
  z-index: 100;
}

.logo {
  font-size: 18px;
  font-weight: bold;
  color: #1890ff;
  cursor: pointer;
  margin-right: 24px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-left: auto;
}

.content {
  margin-top: 64px;
  padding: 24px;
  min-height: calc(100vh - 64px - 70px);
}

.footer {
  text-align: center;
  background: transparent;
  color: #999;
}
</style>