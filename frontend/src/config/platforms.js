export const PLATFORMS = [
  {
    value: 'cidaren',
    title: '词达人',
    icon: 'mdi-book-open-variant',
    credentialFields: [{ key: 'token', label: 'Token', type: 'password' }],
    actions: [
      { value: 'study', title: '班级学习任务' },
      { value: 'class', title: '班级任务' },
    ],
    params: [
      { key: 'task_name', label: '任务名称（子串，可空）', type: 'text', default: '' },
      { key: 'task_type', label: '任务类型（1 学习 / 2 测试）', type: 'number', default: 1 },
      { key: 'progress_lt', label: '仅处理进度低于', type: 'number', default: 100 },
      { key: 'limit', label: '最多任务数（0=不限）', type: 'number', default: 0 },
      { key: 'max_steps', label: '单任务最多题数', type: 'number', default: 1000 },
      { key: 'think_min', label: '思考最短（秒）', type: 'number', default: 2 },
      { key: 'think_max', label: '思考最长（秒）', type: 'number', default: 4 },
      { key: 'spend_min', label: 'time_spent 最短（秒）', type: 'number', default: 5 },
      { key: 'spend_max', label: 'time_spent 最长（秒）', type: 'number', default: 15 },
    ],
  },
  {
    value: 'welearn',
    title: 'WeLearn',
    icon: 'mdi-headphones',
    credentialFields: [
      { key: 'username', label: '账号', type: 'text' },
      { key: 'password', label: '密码', type: 'password' },
    ],
    actions: [
      { value: 'time', title: '挂时长' },
      { value: 'progress', title: '提交进度' },
    ],
    params: [
      { key: 'cid', label: '课程 id', type: 'text', default: '' },
      { key: 'unit_idx', label: '单元下标（0 起，空=全部）', type: 'text', default: '' },
      { key: 'duration', label: '挂时长（秒）', type: 'number', default: 60 },
      { key: 'accuracy', label: '正确率', type: 'number', default: 100 },
      { key: 'include_completed', label: '含已完成 SCO', type: 'switch', default: false },
      { key: 'limit', label: '最多 SCO 数（0=不限）', type: 'number', default: 0 },
    ],
  },
]

export function platformByValue(value) {
  return PLATFORMS.find((platform) => platform.value === value)
}

export function defaultParams(platformValue) {
  const platform = platformByValue(platformValue)
  const params = {}
  platform?.params.forEach((item) => {
    params[item.key] = item.default
  })
  return params
}
