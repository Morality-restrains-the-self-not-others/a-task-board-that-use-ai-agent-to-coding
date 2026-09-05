import { normalizeProjectTags } from './projectTagsUtils.js'
import {
  projectHasConfiguredRunTemplate,
  resolveTemplateForImagePatch,
  runTemplateIsHardwareComplete,
} from './projectRunTemplateUtils.js'

/** 是否允许开启「是否允许自动运行」（与 ProjectEdit 门槛一致：镜像 + 运行模版） */
export function canEnableDefaultAutoRun(project) {
  const imageId = String(project?.container_image_id || '').trim()
  if (!imageId) return false
  return projectHasConfiguredRunTemplate(project)
}

export function buildTagsPatchBody(tags) {
  return { tags: normalizeProjectTags(Array.isArray(tags) ? tags : []) }
}

/**
 * @param {string} imageId
 * @param {Array<{id?: unknown, name?: string}>} installedImages
 * @param {object|null} [runTemplate]
 * @param {object|null} [savedTemplate] 项目已存模版；与 live 草稿一起解析，避免半成品覆盖
 */
export function buildImagePatchBody(imageId, installedImages = [], runTemplate = null, savedTemplate = null) {
  const id = String(imageId || '').trim()
  if (!id) {
    return { container_image_id: '', container_image: '' }
  }
  const matched = (Array.isArray(installedImages) ? installedImages : []).find(
    (image) => String(image?.id) === id,
  )
  const name = String(matched?.name || '').trim()
  const body = {
    container_image_id: id,
    container_image: name,
  }
  const resolved = resolveTemplateForImagePatch(runTemplate, savedTemplate)
  // 仅附带硬件完整模版；半成品省略，让后端沿用库内完整模版做架构校验
  if (resolved && runTemplateIsHardwareComplete(resolved)) {
    body.server_run_template = resolved
  }
  return body
}

/**
 * 合并现有 server_run_template 并写入 default_auto_run
 * @param {object|null|undefined} project
 * @param {boolean} enabled
 */
export function buildAutoRunPatchBody(project, enabled) {
  const existing =
    project?.server_run_template && typeof project.server_run_template === 'object'
      ? { ...project.server_run_template }
      : {}
  return {
    server_run_template: {
      ...existing,
      default_auto_run: Boolean(enabled),
    },
  }
}

/** 清除镜像时：在现有模版上关闭 default_auto_run */
export function buildClearImagePatchBody(project, installedImages = []) {
  const existing =
    project?.server_run_template && typeof project.server_run_template === 'object'
      ? { ...project.server_run_template }
      : {}
  const base = buildImagePatchBody('', installedImages)
  return {
    ...base,
    server_run_template: {
      ...existing,
      default_auto_run: false,
    },
  }
}
