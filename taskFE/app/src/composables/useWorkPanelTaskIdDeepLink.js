/**
 * WorkPanel：根据路由 query.task_id 在 todos 就绪后打开任务详情。
 */
import { watch } from 'vue'
import { findTodoByTaskIdQuery } from '../utils/navbarTaskSearch.js'

/**
 * @param {{
 *   route: import('vue-router').RouteLocationNormalizedLoaded,
 *   router: import('vue-router').Router,
 *   todos: import('vue').Ref<unknown[]>,
 *   openTask: (task: unknown) => void,
 * }} deps
 */
export function useWorkPanelTaskIdDeepLink(deps) {
  const { route, router, todos, openTask } = deps
  let lastOpened = ''

  const tryOpen = () => {
    const taskId = String(route.query?.task_id || '').trim()
    if (!taskId) {
      lastOpened = ''
      return
    }
    if (taskId === lastOpened) return
    const hit = findTodoByTaskIdQuery(todos.value || [], taskId)
    if (!hit) return
    lastOpened = taskId
    console.info('[WorkPanel] deep-link open task_id', taskId)
    openTask(hit)
  }

  watch(
    () => [route.query?.task_id, todos.value],
    () => tryOpen(),
    { deep: true, immediate: true },
  )

  /** 关闭详情时可清掉 task_id，避免再次打开同一任务时不触发 */
  const clearTaskIdQuery = () => {
    if (!route.query?.task_id) return
    const next = { ...route.query }
    delete next.task_id
    router.replace({ query: next }).catch(() => {})
  }

  return { tryOpen, clearTaskIdQuery }
}
