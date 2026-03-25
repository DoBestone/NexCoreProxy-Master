import axios from 'axios'
import { message } from 'ant-design-vue'

const request = axios.create({
  baseURL: '/api',
  timeout: 30000
})

request.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
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
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    message.error(error.message || '网络错误')
    return Promise.reject(error)
  }
)

export default request