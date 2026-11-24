import http from '../http'

export const platformsApi = {
  list: () => http.get('/platforms'),
  items: (id, credentialId) =>
    http.get(`/platforms/${id}/items`, {
      params: credentialId ? { credential_id: credentialId } : {},
    }),
  discover: (id, credentialId, params = {}) =>
    http.post(
      `/platforms/${id}/discover`,
      { credential_id: credentialId || '', params },
      { timeout: 120000 },
    ),
}
