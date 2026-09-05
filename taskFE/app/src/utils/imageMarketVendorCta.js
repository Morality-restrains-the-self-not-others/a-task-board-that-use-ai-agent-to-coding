/** 镜像市场厂商入口四态（纯函数，供 ImageMarket 与单测共用）。 */

export function showVendorSsoButton({ status, hasEmail, reviewEnabled } = {}) {
  if (status === 'pending') return false
  return status === 'qualified' || (reviewEnabled === false && !!hasEmail)
}

export function showBindEmailCta({ hasEmail } = {}) {
  return !hasEmail
}

export function vendorApplyButtonText(status) {
  if (status === 'pending') return '审核中'
  if (status === 'rejected') return '申请被驳回，重新申请'
  return '申请成为厂商门户'
}

export function vendorApplyButtonDisabled(status) {
  return status === 'pending'
}

export function canOpenVendorApplyForm({ status, hasEmail, reviewEnabled } = {}) {
  if (status === 'pending' || status === 'qualified') return false
  // 关审核且已有可投递邮箱 → 走 SSO，不再填表。
  if (reviewEnabled === false && !!hasEmail) return false
  return status === 'none' || status === 'rejected'
}

/** 有邮箱且可申请、表单未展开时显示「申请」CTA（点后再挂载表单）。 */
export function showVendorApplyOpenCta({ status, hasEmail, reviewEnabled, formOpen } = {}) {
  return canOpenVendorApplyForm({ status, hasEmail, reviewEnabled }) && !!hasEmail && !formOpen
}

export function showPendingCta({ status, hasEmail } = {}) {
  return !!hasEmail && status === 'pending'
}
