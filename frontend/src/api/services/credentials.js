import http from '../http'

export const credentialsApi = {
  list: () => http.get('/credentials'),
  upsert: (platform, data) => http.post('/credentials', { platform, data }),
}
