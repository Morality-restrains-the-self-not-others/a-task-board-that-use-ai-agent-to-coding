/**
 * Format auto_run_steps_* fields for UI preview panels.
 */
export function resolveAutoRunStepsPreview({
  markdown = '',
  extractStatus = '',
  liveMarkdown = null,
  liveError = '',
} = {}) {
  const live = liveMarkdown != null ? String(liveMarkdown) : null
  if (live != null && String(live).trim() !== '') {
    return {
      markdown: String(live),
      source: 'live',
      emptyHint: '',
      failed: false,
    }
  }
  const status = String(extractStatus || '').trim()
  const md = String(markdown || '')
  if (md.trim()) {
    return {
      markdown: md,
      source: 'cache',
      emptyHint: '',
      failed: false,
    }
  }
  if (status === 'failed' || status === 'auth_failed') {
    return {
      markdown: '',
      source: 'cache',
      emptyHint: liveError || '自动运行说明抽取失败，暂无法预览',
      failed: true,
    }
  }
  if (status === 'not_found' || status === 'pending') {
    return {
      markdown: '',
      source: 'cache',
      emptyHint:
        status === 'pending'
          ? '自动运行说明抽取中…'
          : '暂无自动运行说明（镜像内未找到 autoRunStep.md）',
      failed: false,
    }
  }
  return {
    markdown: '',
    source: 'cache',
    emptyHint: liveError || '暂无自动运行说明',
    failed: false,
  }
}

/** 镜像市场/创建任务：无正文且非抽取中/失败时不占位。 */
export function shouldShowImageAutoRunSteps(image) {
  const preview = resolveAutoRunStepsPreview({
    markdown: image?.auto_run_steps_md || '',
    extractStatus: image?.auto_run_steps_extract_status || '',
  })
  if (String(preview.markdown || '').trim()) return true
  if (preview.failed) return true
  return String(image?.auto_run_steps_extract_status || '').trim() === 'pending'
}

export function resolveSelectedInstalledImage(editingTask, installedImages) {
  const imageId = editingTask?.container_image?.id
  if (imageId == null || String(imageId).trim() === '') return null
  const want = String(imageId)
  return (installedImages || []).find((img) => String(img.id) === want) || null
}
