/**
 * WorkPanel：按机器节点运行态过滤看板任务（摘要「已启动 / 闲置」点击）。
 */

/**
 * @typedef {'started' | 'idle' | null | undefined} MachineRuntimeFilter
 * @typedef {{ machineRunning?: boolean, containerRunning?: boolean }} TaskRuntimeIndicator
 */

/**
 * @param {MachineRuntimeFilter} filter
 * @returns {'started' | 'idle' | null}
 */
export function normalizeMachineRuntimeFilter(filter) {
  if (filter === 'started' || filter === 'idle') {
    return filter
  }
  return null
}

/**
 * 切换过滤：点同一项清除，点另一项切换。
 * @param {MachineRuntimeFilter} current
 * @param {'started' | 'idle'} next
 * @returns {'started' | 'idle' | null}
 */
export function toggleMachineRuntimeFilter(current, next) {
  const cur = normalizeMachineRuntimeFilter(current)
  if (next !== 'started' && next !== 'idle') {
    return cur
  }
  return cur === next ? null : next
}

/**
 * @param {MachineRuntimeFilter} filter
 * @returns {string}
 */
export function machineRuntimeFilterChipLabel(filter) {
  const f = normalizeMachineRuntimeFilter(filter)
  if (f === 'started') {
    return '过滤：已启动'
  }
  if (f === 'idle') {
    return '过滤：闲置'
  }
  return ''
}

/**
 * @param {Array<{ id?: string|number }>} todos
 * @param {Record<string, TaskRuntimeIndicator>} indicators
 * @param {MachineRuntimeFilter} filter
 * @param {{ indicatorsReady?: boolean }} [opts]
 * @returns {Array<{ id?: string|number }>}
 */
export function filterTodosByMachineRuntime(todos, indicators, filter, opts = {}) {
  const list = Array.isArray(todos) ? todos : []
  const f = normalizeMachineRuntimeFilter(filter)
  if (!f) {
    return list
  }
  // 指示器尚未拉取完成时不要按空 map 过滤成空列表（否则摘要「已启动 N」与看板 0 打架）
  if (opts.indicatorsReady === false) {
    return list
  }
  const map = indicators && typeof indicators === 'object' ? indicators : Object.create(null)
  return list.filter((todo) => {
    if (todo == null || typeof todo !== 'object') {
      return false
    }
    const ind = map[String(todo.id)]
    const machineRunning = Boolean(ind?.machineRunning)
    const containerRunning = Boolean(ind?.containerRunning)
    if (f === 'started') {
      return machineRunning
    }
    // idle：机器在、容器未跑
    return machineRunning && !containerRunning
  })
}

/**
 * 摘要有计数但过滤结果为空：指示器未就绪、任务列表未加载、或任务不在当前 todos。
 * @param {{
 *   filter: MachineRuntimeFilter,
 *   filteredCount: number,
 *   todosCount: number,
 *   indicatorsReady: boolean,
 *   startedCount?: number,
 *   idleCount?: number,
 * }} args
 * @returns {boolean}
 */
export function machineRuntimeFilterMismatch(args) {
  const f = normalizeMachineRuntimeFilter(args?.filter)
  if (!f || !args?.indicatorsReady) {
    return false
  }
  if (Number(args.filteredCount) > 0) {
    return false
  }
  if (f === 'started') {
    return Number(args.startedCount) > 0
  }
  return Number(args.idleCount) > 0
}
