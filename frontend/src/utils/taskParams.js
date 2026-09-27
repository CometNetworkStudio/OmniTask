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

// 解析 sparkline 列的值："1,2,3" → [1,2,3]
export function parseSparkline(value) {
  return String(value ?? '')
    .split(/[, ]+/)
    .map((n) => Number.parseFloat(n))
    .filter((n) => Number.isFinite(n))
}
