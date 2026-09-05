/**
 * Normalize API userdata_template (object, bare id, or null) for display / form binding.
 */
export function normalizeUserdataTemplate(t) {
  if (t == null || t === '') return null;
  if (typeof t === 'string' || typeof t === 'number') {
    const id = String(t).trim();
    if (!id) return null;
    return { id };
  }
  if (typeof t === 'object' && t.id != null && String(t.id).trim() !== '') {
    return {
      id: String(t.id),
      name: t.name != null ? String(t.name) : '',
      version: t.version != null ? String(t.version) : '',
    };
  }
  return null;
}

/**
 * Resolve a display label, enriching bare ids from options when name is missing.
 * @param {unknown} t
 * @param {Array<{id: string|number, name?: string, version?: string}>} [options]
 */
export function userdataTemplateDisplayLabel(t, options = []) {
  const normalized = normalizeUserdataTemplate(t);
  if (!normalized) return '—';
  let name = normalized.name || '';
  let version = normalized.version || '';
  if (!name && Array.isArray(options)) {
    const opt = options.find((x) => String(x.id) === normalized.id);
    if (opt) {
      name = opt.name != null ? String(opt.name) : '';
      version = opt.version != null ? String(opt.version) : version;
    }
  }
  if (!name) return '—';
  return `${name} v${version}`.trim();
}
