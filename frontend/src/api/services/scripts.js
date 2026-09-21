import http from '../http'

export const scriptsApi = {
  list: () => http.get('/scripts'),
  install: (url, ref = '') => http.post('/scripts/install', { url, ref }, { timeout: 600000 }),
  toggle: (id, enabled) => http.post(`/scripts/${id}/enable`, { enabled }),
  uninstall: (id) => http.delete(`/scripts/${id}`),
}
