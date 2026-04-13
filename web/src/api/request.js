import axios from 'axios'
import { message } from 'ant-design-vue'

const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

// 获取正确的 token（根据当前路由判断）
const getToken = () => {
  const path = window.location.pathname
  
  // 管理端路由使用 admin_token
  if (path.startsWith('/admin')) {
    return localStorage.getItem('admin_token')
  }
  
  // 用户端路由使用 user_token
  if (path.startsWith('/user')) {
    return localStorage.getItem('user_token')
  }
  
  // 未匹配到明确路由时不发送 token，避免权限越界
  return null
}

// 获取登出跳转路径
const getLogoutPath = () => {
  const path = window.location.pathname
  
  if (path.startsWith('/admin')) {
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_username')
    return '/admin/login'
  }
  
  if (path.startsWith('/user')) {
    localStorage.removeItem('user_token')
    localStorage.removeItem('user_username')
    return '/user/login'
  }
  
  return '/admin/login'
}

request.interceptors.request.use(
  config => {
    const token = getToken()
    if (token) {
      config.headers['Authorization'] = token
    }
    return config
  },
  error => Promise.reject(error)
)

request.interceptors.response.use(
  response => {
    const res = response.data
    if (res.success === false) {
      message.error(res.msg || '请求失败')
      return Promise.reject(new Error(res.msg || 'Error'))
    }
    return res
  },
  error => {
    if (error.response?.status === 401 || error.response?.status === 403) {
      // 先清除 token，再跳转，避免状态残留
      const logoutPath = getLogoutPath()
      message.error('登录已过期，请重新登录')
      setTimeout(() => { window.location.href = logoutPath }, 300)
      return Promise.reject(error)
    }
    if (error.response?.status === 429) {
      message.warning('请求过于频繁，请稍后再试')
      return Promise.reject(error)
    }
    if (!error.response) {
      message.error('网络连接错误')
    } else {
      message.error(error.response?.data?.msg || '请求失败')
    }
    return Promise.reject(error)
  }
)

export default request