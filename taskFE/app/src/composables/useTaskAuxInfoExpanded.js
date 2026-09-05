import { ref, watch } from 'vue'

/** localStorage：任务详情「任务辅助信息」是否展开 */
export const TASK_AUX_INFO_EXPANDED_KEY = 'task-detail-aux-info-expanded'

/**
 * @returns {boolean}
 */
export function readAuxInfoExpanded() {
  try {
    const raw = localStorage.getItem(TASK_AUX_INFO_EXPANDED_KEY)
    if (raw === null) return true
    return raw === '1' || raw === 'true'
  } catch {
    return true
  }
}

/**
 * @param {boolean} expanded
 */
export function writeAuxInfoExpanded(expanded) {
  try {
    localStorage.setItem(TASK_AUX_INFO_EXPANDED_KEY, expanded ? '1' : '0')
  } catch {
    // private mode / quota
  }
}

/**
 * 任务辅助信息折叠态，持久化到 localStorage（默认展开）。
 */
export function useTaskAuxInfoExpanded() {
  const auxInfoExpanded = ref(readAuxInfoExpanded())

  watch(auxInfoExpanded, (value) => {
    writeAuxInfoExpanded(Boolean(value))
  })

  return { auxInfoExpanded }
}
