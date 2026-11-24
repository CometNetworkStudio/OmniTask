import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || `${import.meta.env.BASE_URL}api/v1`,
  timeout: 15000,
})

http.interceptors.request.use((config) => {
  const token = localStorage.getItem('omnitask_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) {
        return body.data
      }
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return body
  },
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('omnitask_token')
      localStorage.removeItem('omnitask_user')
      if (window.location.pathname !== '/login') {
        window.location.assign('/login')
      }
    }
    const message =
      err.response?.data?.data?.error || err.response?.data?.message || err.message || '网络错误'
    return Promise.reject(new Error(message))
  },
)

export default http
