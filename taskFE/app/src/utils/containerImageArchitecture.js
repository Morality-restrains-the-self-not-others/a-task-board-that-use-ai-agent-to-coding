/** Canonical CPU architecture labels used in task detail / cloud instance filtering. */
export const CONTAINER_IMAGE_ARCHITECTURE_OPTIONS = Object.freeze([
  { value: 'x86_64', label: 'x86_64 (amd64)' },
  { value: 'arm64', label: 'arm64 (aarch64)' },
])

const KNOWN_CPU_ARCHITECTURES = new Set(['x86_64', 'arm64'])

export function normalizeArchitecture(value) {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return ''
  if (normalized === 'x86' || normalized === 'x86_64' || normalized === 'amd64') return 'x86_64'
  if (normalized === 'arm' || normalized === 'arm64' || normalized === 'aarch64') return 'arm64'
  return normalized
}

export function normalizeArchitectureList(architectures) {
  if (!Array.isArray(architectures)) return []
  const seen = new Set()
  const out = []
  for (const item of architectures) {
    const canonical = normalizeArchitecture(item)
    if (!canonical || seen.has(canonical)) continue
    seen.add(canonical)
    out.push(canonical)
  }
  return out
}

export function knownCPUArchitectures(values) {
  return normalizeArchitectureList(values).filter((value) => KNOWN_CPU_ARCHITECTURES.has(value))
}

/** Keep in sync with taskProjectService extractCPUArchitecturesFromText. */
export function extractCPUArchitecturesFromText(...texts) {
  const found = []
  for (const text of texts) {
    const source = String(text || '')
    for (const match of source.matchAll(/(?:^|[^a-z0-9])(x86_64|amd64|aarch64|arm64)(?:[^a-z0-9]|$)/gi)) {
      found.push(match[1])
    }
  }
  return knownCPUArchitectures(found)
}

export function resolveImageArchitectures(imageOrArchitectures) {
  if (Array.isArray(imageOrArchitectures)) {
    return knownCPUArchitectures(imageOrArchitectures)
  }
  const declared = knownCPUArchitectures(imageOrArchitectures?.target_architectures)
  if (declared.length) return declared
  return extractCPUArchitecturesFromText(
    imageOrArchitectures?.version,
    imageOrArchitectures?.name,
    imageOrArchitectures?.image_url,
  )
}

export function resolvePrimaryImageArchitecture({ task = null, selectedImageId = '', installedImages = [] } = {}) {
  const imageId = String(selectedImageId || task?.container_image_id || task?.container_image?.id || '').trim()
  if (imageId) {
    const images = Array.isArray(installedImages) ? installedImages : []
    const image = images.find((item) => String(item?.id) === imageId)
    const fromImage = resolveImageArchitectures(image)
    if (fromImage.length > 0) return fromImage[0]
  }
  const nested = task?.container_image
  const fromTask = resolveImageArchitectures(nested)
  return fromTask[0] || ''
}

/** Keep in sync with taskCloudService inferInstanceArchitecture (ecs.r6.* is x86). */
export function inferInstanceArchitecture(instanceType) {
  const t = String(instanceType || '').trim().toLowerCase()
  if (!t) return ''
  if (
    t.includes('arm')
    || t.includes('g8y')
    || t.includes('c8y')
    || t.includes('r8y')
    || t.includes('c6r')
    || t.includes('g6r')
    || t.includes('r6r')
  ) {
    return 'arm64'
  }
  return 'x86_64'
}

export function instanceCompatibleWithImageArchitectures(instanceType, architectures) {
  const arches = resolveImageArchitectures(architectures)
  if (!arches.length) return false
  const got = inferInstanceArchitecture(instanceType)
  return Boolean(got) && arches.includes(got)
}

export function formatContainerImageArchitectures(imageOrArchitectures) {
  const normalized = resolveImageArchitectures(imageOrArchitectures)
  if (!normalized.length) return ''
  return normalized.join(', ')
}

export function formatContainerImageArchitectureSuffix(imageOrArchitectures) {
  const label = formatContainerImageArchitectures(imageOrArchitectures)
  return label ? ` [${label}]` : ''
}

/** Keep in sync with taskProjectService formatImageRequiredArch. */
export function formatImageRequiredArch(imageOrArchitectures) {
  const canonical = resolveImageArchitectures(imageOrArchitectures)
  if (canonical.length) return canonical.join(', ')
  const raw = Array.isArray(imageOrArchitectures)
    ? imageOrArchitectures
    : (Array.isArray(imageOrArchitectures?.target_architectures)
      ? imageOrArchitectures.target_architectures
      : [])
  const shown = []
  for (const item of raw) {
    const value = String(item ?? '').trim()
    if (!value || value === '<nil>') continue
    shown.push(value)
  }
  if (shown.length) return `${shown.join(', ')}（无法识别为 x86_64/arm64）`
  return '未声明'
}

/** Keep in sync with taskProjectService formatInstanceSupportedArch. */
export function formatInstanceSupportedArch(instanceType) {
  const inst = String(instanceType || '').trim()
  const got = inferInstanceArchitecture(inst)
  if (!got) {
    return inst ? `无法从实例规格 ${inst} 推断` : '未声明'
  }
  return inst ? `${got}（实例规格 ${inst}）` : got
}
