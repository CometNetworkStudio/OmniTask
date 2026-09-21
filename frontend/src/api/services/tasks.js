import http from '../http'

export const tasksApi = {
  list: (page = 1, size = 20) => http.get('/tasks', { params: { page, size } }),
  get: (id) => http.get(`/tasks/${id}`),
  create: (payload) => http.post('/tasks', payload),
  cancel: (id) => http.post(`/tasks/${id}/cancel`),
  retry: (id) => http.post(`/tasks/${id}/retry`),
  remove: (id) => http.delete(`/tasks/${id}`),
}
