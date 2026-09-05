export function memberNameRequiredForLink(linkKind) {
  return linkKind !== 'open'
}

export function parseOpenMaxUses(raw) {
  const s = String(raw ?? '').trim()
  if (!s) return 0
  const n = Number(s)
  if (!Number.isFinite(n) || n <= 0) return 0
  return Math.floor(n)
}

export function formatInviteUseLabel(invite) {
  if (!invite || invite.link_kind !== 'open') {
    return '单次'
  }
  const used = Number(invite.use_count || 0)
  const max = Number(invite.max_uses || 0)
  if (!max) {
    return `已加入 ${used} / 不限`
  }
  return `已加入 ${used} / ${max}`
}

export function buildLinkInviteBody({
  formData,
  pendingGrants,
  selectedRoleNames,
  linkKind,
  maxUses,
}) {
  const body = {
    invite_method: 'link',
    company_member_name: formData.company_member_name,
    role: formData.role,
    expiration_days: formData.expiration_days,
    workspace_id: formData.workspace_id,
  }
  if (formData.role !== 'admin' && pendingGrants.length) {
    body.grants = pendingGrants
  }
  if (formData.role !== 'admin' && selectedRoleNames.length) {
    body.role_names = selectedRoleNames
  }
  if (linkKind === 'open') {
    body.link_kind = 'open'
    body.max_uses = parseOpenMaxUses(maxUses)
  }
  return body
}
