/**
 * Map billing resource_type codes to zh-CN labels.
 *
 * Single source of truth shared by OrderDetail / BillingOrders /
 * SystemAdminOrderRecords so a newly added resource type gets one label
 * instead of drifting across three copies.
 *
 * Unknown types fall back to the raw code (still readable until a label
 * lands here). See OPT-20260815-002.
 *
 * @param {string} resourceType
 * @returns {string}
 */
export function billingResourceTypeLabel(resourceType) {
  const map = {
    task_post: '任务帖',
    gitlab_disk: 'GitLab 磁盘',
    gitlab_traffic: 'GitLab 流量',
  }
  return map[resourceType] || resourceType
}
