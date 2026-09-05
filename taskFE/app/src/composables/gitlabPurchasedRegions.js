/**
 * Normalize billing gitlab-resources[] into selectable sync sources.
 * Drop rows without gitlab_web_url (e.g. pending_node not yet mounted).
 * @param {unknown} raw
 * @returns {{ region: string, region_name: string, gitlab_web_url: string, provisioning_status: string }[]}
 */
export function normalizePurchasedGitlabRegions(raw) {
  const list = Array.isArray(raw) ? raw : []
  return list
    .map((r) => ({
      region: String(r?.region || '').trim(),
      region_name: String(r?.region_name || r?.region || '').trim(),
      gitlab_web_url: String(r?.gitlab_web_url || '').trim(),
      provisioning_status: String(r?.provisioning_status || '').trim(),
    }))
    .filter((r) => r.region && r.gitlab_web_url)
}
