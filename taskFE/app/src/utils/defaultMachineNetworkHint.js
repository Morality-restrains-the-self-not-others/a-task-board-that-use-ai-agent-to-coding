/**
 * 租户默认机器节点（server-config-default）→ 自建 GitLab 同 VPC 提示用摘要。
 */

/**
 * @param {unknown} configs
 * @returns {boolean}
 */
export function shouldShowSameVpcHint(configs) {
  return Array.isArray(configs) && configs.length > 0
}

/**
 * @param {unknown} configs
 * @returns {{
 *   authorization_id: string,
 *   platform_type: string,
 *   region: string,
 *   vpc_id: string,
 *   vswitch_id: string,
 *   zone_id: string,
 * } | null}
 */
export function pickPrimaryDefaultNetwork(configs) {
  if (!shouldShowSameVpcHint(configs)) return null
  const list = configs.filter((row) => row && typeof row === 'object')
  if (list.length === 0) return null
  const withVpc = list.find((row) => String(row.vpc_id || '').trim())
  const row = withVpc || list[0]
  return {
    authorization_id: String(row.authorization_id || '').trim(),
    platform_type: String(row.platform_type || row.platform || '').trim(),
    region: String(row.region || '').trim(),
    vpc_id: String(row.vpc_id || '').trim(),
    vswitch_id: String(row.vswitch_id || '').trim(),
    zone_id: String(row.zone_id || '').trim(),
  }
}
