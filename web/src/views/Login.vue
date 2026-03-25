<template>
  <div class="login-container">
    <a-card class="login-card">
      <template #title>
        <div class="login-title">
          <span>NexCore代理主机</span>
        </div>
      </template>
      <a-form :model="form" @finish="handleLogin">
        <a-form-item name="username" :rules="[{ required: true, message: '请输入用户名' }]">
          <a-input v-model:value="form.username" placeholder="用户名" size="large">
            <template #prefix><UserOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item name="password" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model:value="form.password" placeholder="密码" size="large">
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" :loading="loading" block>
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined } from '@ant-design/icons-vue'
import { login } from '@/api'

const router = useRouter()
const loading = ref(false)
const form = ref({
  username: '',
  password: ''
})

const handleLogin = async () => {
  loading.value = true
  try {
    const res = await login(form.value)
    if (res.success) {
      localStorage.setItem('token', 'session_' + form.value.username)
      localStorage.setItem('username', form.value.username)
      message.success('登录成功')
      
      // 根据用户角色跳转
      const role = res.obj?.role || 'user'
      if (role === 'admin') {
        router.push('/admin/dashboard')
      } else {
        router.push('/user/nodes')
      }
    }
  } catch (e) {
    message.error('登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.login-title {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  font-size: 20px;
  font-weight: bold;
  color: #1890ff;
}
</style>