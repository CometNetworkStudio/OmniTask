import http from '../http'

export const usersApi = {
  list: (page = 1, size = 0) => http.get('/users', { params: { page, size } }),
  setRole: (id, role) => http.post(`/users/${id}/role`, { role }),
  setEnabled: (id, enabled) => http.post(`/users/${id}/enable`, { enabled }),
}
