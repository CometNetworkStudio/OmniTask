import http from '../http'

export const preferencesApi = {
  get: () => http.get('/me/preferences'),
  update: (pinnedPlatforms) => http.put('/me/preferences', { pinned_platforms: pinnedPlatforms }),
}
