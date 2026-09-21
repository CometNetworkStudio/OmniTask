import http from '../http'

export const scriptsApi = {
  list: () => http.get('/scripts'),
  install: (url, ref = '') => http.post('/scripts/install', { url, ref }, { timeout: 30000 }),
  installStatus: (jobId) => http.get(`/scripts/install/${jobId}`),
  toggle: (id, enabled) => http.post(`/scripts/${id}/enable`, { enabled }),
  uninstall: (id) => http.delete(`/scripts/${id}`),
}
