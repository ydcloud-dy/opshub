import axios from 'axios'
import { ElMessage } from 'element-plus'

const request = axios.create({
  baseURL: '/',
  timeout: 60000 // 60秒超时
})

// Token过期跳转标志，防止重复跳转
let isRedirecting = false

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
request.interceptors.response.use(
  (response) => {
    // blob类型响应直接返回
    if (response.config.responseType === 'blob') {
      return response.data
    }

    // text类型响应直接返回（用于.cast文件等）
    if (response.config.responseType === 'text') {
      return response.data
    }

    const res = response.data
    // 检查业务状态码
    if (res.code !== 0 && res.code !== 200) {
      const url = response.config.url || ''

      // 检查是否是Token过期错误
      if (res.message && (
        res.message.includes('Token无效') ||
        res.message.includes('Token已过期') ||
        res.message.includes('token无效') ||
        res.message.includes('token已过期') ||
        res.message.includes('未登录') ||
        res.message.includes('登录已过期')
      )) {
        // 避免重复跳转
        if (!isRedirecting) {
          isRedirecting = true
          ElMessage.error('登录已过期，请重新登录')
          localStorage.removeItem('token')
          // 延迟跳转，让用户看到提示
          setTimeout(() => {
            window.location.href = '/login'
          }, 1000)
        }
        return Promise.reject({
          code: res.code,
          message: res.message || '请求失败',
          response: response
        })
      }

      // 只在非登录接口的情况下自动显示错误消息
      // 登录接口和验证码接口的错误由调用方处理,避免重复提示
      if (!url.includes('/login') && !url.includes('/captcha')) {
        ElMessage.error(res.message || '请求失败')
      }
      // 返回完整的响应对象，让调用方可以访问code和message
      return Promise.reject({
        code: res.code,
        message: res.message || '请求失败',
        response: response
      })
    }
    // 返回实际数据 (res.data)
    return res.data
  },
  (error) => {
    const status = error.response?.status
    const url = error.config?.url || ''

    // 401 - 未登录，跳转到登录页
    if (status === 401) {
      // 只在非登录请求时自动跳转到登录页
      if (!url.includes('/login') && !isRedirecting) {
        isRedirecting = true
        ElMessage.error('登录已过期，请重新登录')
        localStorage.removeItem('token')
        setTimeout(() => {
          window.location.href = '/login'
        }, 1000)
      } else if (url.includes('/login')) {
        // 登录接口的401错误,返回错误信息给调用方处理
        const errorMsg = error.response?.data?.message || '用户名或密码错误'
        return Promise.reject(new Error(errorMsg))
      }
      return Promise.reject(error)
    }

    // 403 - 权限不足，只显示错误消息，不跳转
    if (status === 403) {
      const errorMsg = error.response?.data?.message || '权限不足'
      ElMessage.error({
        message: errorMsg,
        duration: 5000,
        showClose: true
      })
      return Promise.reject(error)
    }

    // 其他错误 - 只对非登录接口显示错误消息
    if (!url.includes('/login')) {
      const errorMsg = error.response?.data?.message || error.message || '网络错误'
      ElMessage.error(errorMsg)
    }
    return Promise.reject(error)
  }
)

export default request
