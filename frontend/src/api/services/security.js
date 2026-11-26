import http from '../http'

export const securityApi = {
  mySessions: () => http.get('/me/sessions'),
  revokeMySession: (id) => http.delete(`/me/sessions/${id}`),
  revokeOtherSessions: () => http.delete('/me/sessions'),
  overview: () => http.get('/admin/security'),
  sessions: (limit = 100) => http.get('/admin/sessions', { params: { limit } }),
  attempts: (limit = 50) => http.get('/admin/login-attempts', { params: { limit } }),
  revokeSession: (id) => http.delete(`/admin/sessions/${id}`),
  revokeUserSessions: (userId) => http.post(`/admin/users/${userId}/revoke-sessions`),
}
