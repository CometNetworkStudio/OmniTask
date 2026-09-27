import http from '../http'

export const credentialsApi = {
  list: () => http.get('/credentials'),
  create: (payload) => http.post('/credentials', payload),
  update: (id, payload) => http.put(`/credentials/${id}`, payload),
  archive: (id) => http.delete(`/credentials/${id}`),
  accessLogs: (limit = 50) => http.get('/credentials/access-logs', { params: { limit } }),
}
