export function connectTaskSocket(taskId, { onEvent, onClose } = {}) {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const base = import.meta.env.BASE_URL || '/'
  const url = `${protocol}://${window.location.host}${base}v1/ws?task_id=${encodeURIComponent(taskId)}`
  const socket = new WebSocket(url)
  socket.onmessage = (event) => {
    try {
      onEvent?.(JSON.parse(event.data))
    } catch {
      // 忽略非 JSON 帧
    }
  }
  socket.onclose = () => onClose?.()
  return socket
}
