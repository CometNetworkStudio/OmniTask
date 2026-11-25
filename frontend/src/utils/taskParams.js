// 根据平台参数 schema + 表单值构建执行参数；可选按 discovery.param_map
// 把选中条目的字段并入（条目字段优先）。
export function buildParams(fields, values, paramMap, itemFields) {
  const out = {}
  for (const field of fields || []) {
    const value = values?.[field.key]
    if (field.type === 'switch') {
      if (value) out[field.key] = 'true'
      continue
    }
    if (value === '' || value === null || value === undefined) continue
    out[field.key] = String(value)
  }
  if (paramMap && itemFields) {
    for (const [paramKey, fieldKey] of Object.entries(paramMap)) {
      const value = itemFields[fieldKey]
      if (value !== undefined && value !== null) out[paramKey] = String(value)
    }
  }
  return out
}

// 解析 progress 列的值：支持 0.6 / 60 / "60%"。
export function parseProgress(value) {
  if (value === null || value === undefined || value === '') return 0
  const text = String(value)
  if (text.includes('%')) {
    const n = Number.parseFloat(text)
    return Number.isFinite(n) ? Math.max(0, Math.min(100, n)) : 0
  }
  const n = Number.parseFloat(text)
  if (!Number.isFinite(n)) return 0
  if (n <= 1) return Math.max(0, Math.min(100, n * 100))
  return Math.max(0, Math.min(100, n))
}

// 解析 chip 列的值："已完成:success" / "进行中" → { text, color }
export function parseChip(value) {
  const text = String(value ?? '')
  const idx = text.indexOf(':')
  if (idx < 0) return { text, color: undefined }
  return { text: text.slice(0, idx), color: text.slice(idx + 1) || undefined }
}

// 解析时长文本：支持 "50小时24分钟"、"31分钟"、"01:02:03"、"1h 20m"。
export function parseDuration(value) {
  if (value === null || value === undefined || value === '') return 0
  const text = String(value).trim()
  const clock = text.match(/^(\d+):(\d{1,2})(?::(\d{1,2}))?$/)
  if (clock) {
    return Number(clock[1]) * 3600 + Number(clock[2]) * 60 + Number(clock[3] || 0)
  }
  let total = 0
  let matched = false
  const units = /(\d+(?:\.\d+)?)\s*(天|小时|时|分钟|分|秒|h|m|s)/gi
  let match
  while ((match = units.exec(text))) {
    matched = true
    const amount = Number(match[1])
    const unit = match[2].toLowerCase()
    if (unit === '天') total += amount * 86400
    else if (unit === '小时' || unit === '时' || unit === 'h') total += amount * 3600
    else if (unit === '分钟' || unit === '分' || unit === 'm') total += amount * 60
    else total += amount
  }
  if (matched) return total
  const number = Number.parseFloat(text.replace(/[,\s]/g, ''))
  return Number.isFinite(number) ? number : 0
}

// 按 manifest 的 sort 类型比较字段；sort 为空时按 type/文本兜底。
// 不同脚本的字段格式不同，脚本可在 columns[].sort 声明 duration/percent/number/date。
export function compareFieldValues(a, b, column = {}) {
  const mode =
    column.sort ||
    (column.type === 'progress' ? 'progress' : column.type === 'date' ? 'date' : 'text')
  const left = a ?? ''
  const right = b ?? ''
  if (mode === 'progress') return parseProgress(left) - parseProgress(right)
  if (mode === 'duration') return parseDuration(left) - parseDuration(right)
  const numeric = (value) => {
    const n = Number.parseFloat(String(value).replace(/[,%\s]/g, ''))
    return Number.isFinite(n) ? n : 0
  }
  if (mode === 'number' || mode === 'percent') return numeric(left) - numeric(right)
  if (mode === 'date') {
    const at = Date.parse(left)
    const bt = Date.parse(right)
    return (Number.isFinite(at) ? at : 0) - (Number.isFinite(bt) ? bt : 0)
  }
  return String(left).localeCompare(String(right), 'zh-CN', { numeric: true, sensitivity: 'base' })
}

// 解析 sparkline 列的值："1,2,3" → [1,2,3]
export function parseSparkline(value) {
  return String(value ?? '')
    .split(/[, ]+/)
    .map((n) => Number.parseFloat(n))
    .filter((n) => Number.isFinite(n))
}
