import http from '../http'

export const tasksApi = {
  list: (params = {}) => http.get('/tasks', { params }),
  get: (id) => http.get(`/tasks/${id}`),
  events: (id, params = {}) => http.get(`/tasks/${id}/events`, { params }),
  create: (payload) => http.post('/tasks', payload),
  cancel: (id) => http.post(`/tasks/${id}/cancel`),
  retry: (id) => http.post(`/tasks/${id}/retry`),
  remove: (id) => http.delete(`/tasks/${id}`),
}
