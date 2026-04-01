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
  
  // 默认尝试两个 token
  return localStorage.getItem('admin_token') || localStorage.getItem('user_token')
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
      const logoutPath = getLogoutPath()
      window.location.href = logoutPath
    }
    message.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

export default request