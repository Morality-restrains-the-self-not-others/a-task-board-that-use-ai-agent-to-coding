export function inboxBodyWithoutDuplicatedReason(body, reason) {
  const raw = String(body ?? '')
  const why = String(reason ?? '').trim()
  if (!why) return raw
  const marker = `理由：${why}`
  const idx = raw.lastIndexOf(marker)
  if (idx < 0) return raw
  if (raw.slice(idx + marker.length).trim() !== '') return raw
  return raw.slice(0, idx).replace(/[\s\n]+$/g, '')
}
