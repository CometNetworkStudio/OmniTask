import http from '../http'

export const questionsApi = {
  list: (platform, page = 1, size = 20) =>
    http.get('/questions', { params: { platform, page, size } }),
  lookup: (platform, questionKey) =>
    http.get('/questions/lookup', { params: { platform, question_key: questionKey } }),
}
