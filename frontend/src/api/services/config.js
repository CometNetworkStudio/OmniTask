import http from '../http'

export const configApi = {
  get: () => http.get('/admin/config'),
  update: (values) => http.put('/admin/config', { values }),
  reload: () => http.post('/admin/config/reload'),
}
