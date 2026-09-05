import { computed, ref, watch } from 'vue'

// OPT-20260724-016: 全量 Map-based 存储，按 taskId 隔离 $image 提及状态。
// 移除全局 pendingImageMention ref，改为基于 _mentions Map 的 computed 视图，
// 保留向后兼容的无 taskId 调用（使用 DEFAULT_SLOT）。

const DEFAULT_SLOT = '__default__'

/** @type {Map<string, import('vue').Ref<{id: string, name: string}|null>>} */
const _mentions = new Map()

function _slot(taskId) {
  const key = taskId != null ? String(taskId) : DEFAULT_SLOT
  if (!_mentions.has(key)) {
    _mentions.set(key, ref(null))
  }
  return _mentions.get(key)
}

/**
 * 设置当前待提交的 $镜像 提及。
 * @param {{ id: string, name: string }|null} mention
 * @param {string|number} [taskId] — 传入时按任务隔离；省略时写入默认槽
 */
export function setPendingImageMention(mention, taskId) {
  const value = (mention && mention.id)
    ? {
        id: String(mention.id),
        name: String(mention.name || ''),
        skill: String(mention.skill || ''),
      }
    : null
  _slot(taskId).value = value
}

/**
 * 读取当前待提交的 $镜像 提及。
 * @param {string|number} [taskId]
 * @returns {{ id: string, name: string }|null}
 */
export function getPendingImageMention(taskId) {
  return _slot(taskId).value
}

/**
 * 清除待提交的 $镜像 提及。
 * @param {string|number} [taskId] — 传入时只清该任务槽；省略时清默认槽
 */
export function clearPendingImageMention(taskId) {
  _slot(taskId).value = null
}

/**
 * 向后兼容的 reactive 视图：指向默认槽。
 * 调用方直接读写 .value 等价于无 taskId 调用 setPendingImageMention / getPendingImageMention。
 * 逐步迁移调用方为显式传 taskId 的 setPendingImageMention / clearPendingImageMention。
 */
export const pendingImageMention = computed({
  get: () => _slot().value,
  set: (val) => { _slot().value = val },
})
