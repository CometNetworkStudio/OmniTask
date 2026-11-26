import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || `${import.meta.env.BASE_URL}api/v1`,
  timeout: 15000,
  withCredentials: true,
})

const CSRF_COOKIE = 'omnitask_csrf'
const CSRF_HEADER = 'X-CSRF-Token'
const SAFE_METHODS = ['GET', 'HEAD', 'OPTIONS']

function readCookie(name) {
  const prefix = `${name}=`
  const row = document.cookie.split('; ').find((item) => item.startsWith(prefix))
  return row ? decodeURIComponent(row.slice(prefix.length)) : ''
}

http.interceptors.request.use((config) => {
  const method = String(config.method || 'get').toUpperCase()
  if (!SAFE_METHODS.includes(method)) {
    const token = readCookie(CSRF_COOKIE)
    if (token) config.headers[CSRF_HEADER] = token
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
    if (err.response?.status === 401 && !err.config?.skipAuthRedirect) {
      const path = window.location.pathname
      if (!['/login', '/register'].includes(path)) {
        window.location.assign('/login')
      }
    }
    const message =
      err.response?.data?.data?.error || err.response?.data?.message || err.message || '网络错误'
    return Promise.reject(new Error(message))
  },
)

export default http
