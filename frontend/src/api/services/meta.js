import http from '../http'

export const metaApi = {
  get: () => http.get('/meta'),
}
