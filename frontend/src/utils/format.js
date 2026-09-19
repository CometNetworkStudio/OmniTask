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
}

export function statusMeta(status) {
  return STATUS_META[status] || { text: status || '-', color: 'grey' }
}
