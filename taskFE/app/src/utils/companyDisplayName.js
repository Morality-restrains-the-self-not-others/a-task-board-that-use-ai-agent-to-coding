/**
 * 租户公司展示名解析。
 * 公司名称由 taskTenantService / accounts 拥有；禁止把 company_id 当名称展示。
 */

export function pickCompanyDisplayName(payload) {
  if (!payload || typeof payload !== 'object') return ''
  return String(payload.name ?? '').trim()
}

/**
 * 读取当前租户公司名称。失败时返回空串，由调用方展示「未设置」。
 * 边界 catch：名称是装饰字段，不得拖垮项目详情主流程。
 */
export async function fetchTenantCompanyDisplayName(apiFetchFn, tenantId) {
  const tid = String(tenantId || '').trim()
  if (!tid) return ''
  if (typeof apiFetchFn !== 'function') return ''
  try {
    const res = await apiFetchFn(
      `/api/tenant/${encodeURIComponent(tid)}/accounts/companies/current/`,
      {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      },
    )
    if (!res || !res.ok) return ''
    const data = await res.json()
    return pickCompanyDisplayName(data)
  } catch {
    return ''
  }
}
