export function formatTime(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

export const STATUS_META = {
  pending: { text: '待执行', color: 'grey' },
  queued: { text: '排队中', color: 'info' },
  running: { text: '执行中', color: 'warning' },
  succeeded: { text: '成功', color: 'success' },
  failed: { text: '失败', color: 'error' },
  canceled: { text: '已取消', color: 'grey' },
  interrupted: { text: '已中断', color: 'warning' },
}

export function statusMeta(status) {
  return STATUS_META[status] || { text: status || '-', color: 'grey' }
}

export function formatAgo(value) {
  if (!value) return ''
  const seconds = Math.max(0, (Date.now() - new Date(value).getTime()) / 1000)
  if (seconds < 60) return '刚刚'
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} 小时前`
  return `${Math.floor(seconds / 86400)} 天前`
}
