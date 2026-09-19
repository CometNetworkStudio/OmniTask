import axios from 'axios'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
  timeout: 15000,
})

http.interceptors.response.use(
  (res) => res.data,
  (err) => Promise.reject(err),
)

export default http
