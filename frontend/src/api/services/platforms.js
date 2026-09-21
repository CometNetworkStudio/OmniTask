import http from '../http'

export const platformsApi = {
  list: () => http.get('/platforms'),
  items: (id) => http.get(`/platforms/${id}/items`),
  discover: (id) => http.post(`/platforms/${id}/discover`, {}, { timeout: 120000 }),
}
