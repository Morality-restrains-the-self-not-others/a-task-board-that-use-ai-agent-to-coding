// FAQ markdown 占位符：联系邮箱 SSOT 为 conf/frontend/vue/config.yaml 的 contactEmail，
// 经 Vite define 注入 import.meta.env.VITE_CONTACT_EMAIL 后再替换。
export const CONTACT_EMAIL_PLACEHOLDER = '{{contactEmail}}'

export function applyFaqPlaceholders(content, contactEmail) {
  if (typeof content !== 'string') {
    throw new Error('FAQ content must be a string')
  }
  if (!content.includes(CONTACT_EMAIL_PLACEHOLDER)) {
    return content
  }
  const email = typeof contactEmail === 'string' ? contactEmail.trim() : ''
  if (!email) {
    throw new Error('vue.contactEmail is required to render FAQ {{contactEmail}} placeholders')
  }
  return content.split(CONTACT_EMAIL_PLACEHOLDER).join(email)
}
