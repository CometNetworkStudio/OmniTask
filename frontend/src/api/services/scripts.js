import http from '../http'

export const scriptsApi = {
  list: () => http.get('/scripts'),
  get: (id) => http.get(`/scripts/${id}`),
  install: (url, ref = '') => http.post('/scripts/install', { url, ref }, { timeout: 30000 }),
  installStatus: (jobId) => http.get(`/scripts/install/${jobId}`),
  reinstall: (id) => http.post(`/scripts/${id}/reinstall`, {}, { timeout: 30000 }),
  rollback: (id) => http.post(`/scripts/${id}/rollback`, {}, { timeout: 30000 }),
  debug: (id, payload) => http.post(`/scripts/${id}/debug`, payload, { timeout: 90000 }),
  toggle: (id, enabled) => http.post(`/scripts/${id}/enable`, { enabled }),
  uninstall: (id) => http.delete(`/scripts/${id}`),
}
