/**
 * TaskDetail composable facade — wires split modules (OPT-20260823-007).
 * Keep `export function useTaskDetail` as the TaskDetail.vue entry.
 */
import { initUseTaskDetailCore } from './taskDetail/useTaskDetailCore.js'
import { initUseTaskDetailFetches } from './taskDetail/useTaskDetailFetches.js'
import { initUseTaskDetailContainer } from './taskDetail/useTaskDetailContainer.js'
import { initUseTaskDetailActions } from './taskDetail/useTaskDetailActions.js'
import { buildUseTaskDetailApi } from './taskDetail/useTaskDetailApi.js'

export function useTaskDetail(props, { emit, route, router }) {
  const d = {}
  initUseTaskDetailCore(d, props, { emit, route, router })
  initUseTaskDetailFetches(d)
  initUseTaskDetailContainer(d)
  initUseTaskDetailActions(d)
  return buildUseTaskDetailApi(d)
}
