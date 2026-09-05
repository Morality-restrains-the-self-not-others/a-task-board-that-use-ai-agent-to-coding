function tenantContactParts(tenant) {
  const phone = String(tenant?.phone || '').trim()
  const email = String(tenant?.email || '').trim()
  const parts = []
  if (phone && phone !== '无') parts.push(phone)
  if (email && email !== '无') parts.push(email)
  return parts
}

function tenantContactLine(tenant) {
  return tenantContactParts(tenant).join(' · ')
}

export { tenantContactLine, tenantContactParts }
