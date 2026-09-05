export function deliveryAttemptsTooltip(inv) {
  if (!inv?.deliveryAttempts?.length) return `发送次数: ${inv?.emailSendAttempts ?? 0}`
  const lines = inv.deliveryAttempts.map(a => {
    const statusLabel = { delivered: '已送达', queued: '已排队', failed: '失败', skipped_unsubscribed: '已退订跳过' }[a.deliveryStatus] || a.deliveryStatus
    const methodLabel = { kafka: 'Kafka', smtp: 'SMTP', fallback: '回退' }[a.deliveryMethod] || a.deliveryMethod
    const time = a.attemptedAt ? new Date(a.attemptedAt).toLocaleString('zh-CN') : '?'
    const err = a.deliveryError ? ` (${a.deliveryError})` : ''
    return `第${a.attemptNumber}次 [${methodLabel}] ${statusLabel}${err} — ${time}`
  })
  return `投递历史 (共${inv.emailSendAttempts ?? 0}次):\n${lines.join('\n')}`
}

export function deliveryTooltip(inv) {
  if (!inv) return ''
  if (inv.deliveryStatus === 'failed' && inv.deliveryError) {
    return `失败原因: ${inv.deliveryError}`
  }
  if (inv.deliveryStatus === 'queued') {
    return '邮件已提交到消息队列，等待后台投递（Kafka 异步发送）'
  }
  if (inv.deliveryStatus === 'delivered') {
    return '邮件已通过 SMTP 直接发送并确认送达'
  }
  if (inv.deliveryStatus === 'skipped_unsubscribed') {
    return '该邮箱已退订邮件邀请，未发送邮件'
  }
  return ''
}

export function formatInviteTime(isoStr) {
  if (!isoStr) return '—'
  try {
    const d = new Date(isoStr)
    if (isNaN(d.getTime())) return isoStr
    return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  } catch {
    return isoStr
  }
}
