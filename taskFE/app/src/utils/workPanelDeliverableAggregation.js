/**
 * 工作面板：按交付物类别聚合 / 面包屑过滤（纯函数）。
 */

import {
  deliverableContentOptionLabel,
  todoToDeliverableContent,
} from './workPanelDeliverableContent.js'

export const UNCATEGORIZED_DELIVERABLE_ID = '__uncategorized__'
export const UNCATEGORIZED_DELIVERABLE_NAME = '未分类'
export const UNKNOWN_DELIVERABLE_NAME = '未知类别'

/** @param {unknown} todo */
export function resolveTodoDeliverableObjId(todo) {
  if (!todo || typeof todo !== 'object') return null
  const raw =
    todo.deliverable_obj_id ??
    todo.deliverableObjId ??
    todo.deliverable_obj?.id ??
    todo.deliverableObj?.id ??
    todo.task_type?.id ??
    todo.taskType?.id
  if (raw == null || raw === '') return null
  return String(raw)
}

/**
 * 任务详情 / 看板：将 deliverable_obj_id 解析为可读类别名。
 * @param {unknown} todo
 * @param {Array<{id?: unknown, name?: unknown}>} [categories]
 * @returns {string}
 */
export function resolveDeliverableCategoryDisplayName(todo, categories = []) {
  const id = resolveTodoDeliverableObjId(todo)
  if (!id) return UNCATEGORIZED_DELIVERABLE_NAME
  const nestedName =
    (todo && typeof todo === 'object' && (todo.deliverable_obj?.name || todo.task_type?.name)) || ''
  if (String(nestedName).trim()) return String(nestedName).trim()
  const list = Array.isArray(categories) ? categories : []
  const found = list.find((c) => c && String(c.id) === id)
  if (found?.name != null && String(found.name).trim()) return String(found.name).trim()
  return UNKNOWN_DELIVERABLE_NAME
}

/** @param {unknown} todo */
export function resolveTodoParentTaskId(todo) {
  if (!todo || typeof todo !== 'object') return null
  const raw = todo.parent_task ?? todo.parentTask ?? todo.parent_task_id ?? todo.parentTaskId
  if (raw == null || raw === '') return null
  if (typeof raw === 'object' && raw.id != null) return String(raw.id)
  return String(raw)
}

/**
 * 交付物类别是否为顶层（当前体系中 order 最小）。
 * @param {unknown[]} categories
 * @param {unknown} categoryId
 * @returns {boolean} 无类别或找不到时视为顶层（不强制上层交付物）
 */
export function isTopLevelDeliverableCategory(categories, categoryId) {
  if (categoryId == null || categoryId === '') return true
  const cats = normalizeDeliverableCategories(categories)
  if (!cats.length) return true
  const id = String(categoryId)
  const selected = cats.find((c) => c.id === id)
  if (!selected) return true
  const minOrder = Math.min(...cats.map((c) => c.order))
  return selected.order === minOrder
}

/**
 * 非顶层类别创建任务时，上层交付物候选 = 上一 order 层的任务。
 * @param {unknown[]} todos
 * @param {unknown[]} categories
 * @param {unknown} selectedCategoryId
 * @returns {Array<{ id: string, title: string, workspace_seq?: number }>}
 */
export function listParentDeliverableCandidates(todos, categories, selectedCategoryId) {
  if (isTopLevelDeliverableCategory(categories, selectedCategoryId)) return []
  const cats = normalizeDeliverableCategories(categories)
  const selected = cats.find((c) => c.id === String(selectedCategoryId))
  if (!selected) return []
  const parentOrder = selected.order - 1
  const parentCategoryIds = new Set(cats.filter((c) => c.order === parentOrder).map((c) => c.id))
  if (!parentCategoryIds.size) return []
  const list = Array.isArray(todos) ? todos : []
  const out = []
  const seen = new Set()
  for (const todo of list) {
    if (!todo || todo.id == null || todo.id === '') continue
    const catId = resolveTodoDeliverableObjId(todo)
    if (!catId || !parentCategoryIds.has(catId)) continue
    const content = todoToDeliverableContent(todo)
    if (!content || seen.has(content.id)) continue
    seen.add(content.id)
    out.push(content)
  }
  return out.sort((a, b) => a.title.localeCompare(b.title, 'zh-CN') || a.id.localeCompare(b.id))
}

/**
 * @param {Array<{id?: unknown, order?: unknown}>} categories
 * @returns {Array<{id: string, name: string, color: string, order: number}>}
 */
export function normalizeDeliverableCategories(categories) {
  const list = Array.isArray(categories) ? categories : []
  return list
    .map((c) => {
      if (!c || c.id == null || c.id === '') return null
      const orderNum = Number(c.order)
      return {
        id: String(c.id),
        name: String(c.name ?? ''),
        color: String(c.color ?? '#3b82f6'),
        order: Number.isFinite(orderNum) ? orderNum : 0,
      }
    })
    .filter(Boolean)
    .sort((a, b) => {
      if (a.order !== b.order) return a.order - b.order
      return a.id.localeCompare(b.id)
    })
}

/**
 * 选中类别及其子类别（order 更大）的 ID 集合。
 * @param {ReturnType<typeof normalizeDeliverableCategories>} categories
 * @param {string|null|undefined} selectedCategoryId
 * @returns {Set<string>|null} null 表示不过滤类别
 */
export function resolveCategoryIdClosure(categories, selectedCategoryId) {
  if (selectedCategoryId == null || selectedCategoryId === '') return null
  const selectedId = String(selectedCategoryId)
  if (selectedId === UNCATEGORIZED_DELIVERABLE_ID) {
    return new Set([UNCATEGORIZED_DELIVERABLE_ID])
  }
  const cats = normalizeDeliverableCategories(categories)
  const selected = cats.find((c) => c.id === selectedId)
  if (!selected) return new Set([selectedId])
  const ids = new Set(
    cats.filter((c) => c.order >= selected.order).map((c) => c.id),
  )
  return ids
}

/**
 * @param {unknown[]} todos
 * @param {string|null|undefined} rootTaskId
 * @returns {Set<string>|null}
 */
export function resolveTaskSubtreeIds(todos, rootTaskId) {
  if (rootTaskId == null || rootTaskId === '') return null
  const root = String(rootTaskId)
  const list = Array.isArray(todos) ? todos : []
  const childrenByParent = new Map()
  for (const todo of list) {
    const id = todo?.id != null ? String(todo.id) : null
    if (!id) continue
    const parent = resolveTodoParentTaskId(todo)
    if (!parent) continue
    if (!childrenByParent.has(parent)) childrenByParent.set(parent, [])
    childrenByParent.get(parent).push(id)
  }
  const out = new Set([root])
  const queue = [root]
  while (queue.length) {
    const cur = queue.shift()
    const kids = childrenByParent.get(cur) || []
    for (const kid of kids) {
      if (out.has(kid)) continue
      out.add(kid)
      queue.push(kid)
    }
  }
  return out
}

/**
 * @typedef {{ type: 'root' } | { type: 'category', id: string, label: string } | { type: 'task', id: string, label: string }} DeliverablePathSegment
 */

/**
 * @param {DeliverablePathSegment[]|null|undefined} path
 * @returns {{ categoryId: string|null, taskId: string|null }}
 */
export function readFilterFromPath(path) {
  const segments = Array.isArray(path) ? path : []
  let categoryId = null
  let taskId = null
  for (const seg of segments) {
    if (!seg || typeof seg !== 'object') continue
    if (seg.type === 'category' && seg.id != null && seg.id !== '') {
      categoryId = String(seg.id)
    }
    if (seg.type === 'task' && seg.id != null && seg.id !== '') {
      taskId = String(seg.id)
    }
  }
  return { categoryId, taskId }
}

/**
 * @param {unknown[]} todos
 * @param {unknown[]} categories
 * @param {DeliverablePathSegment[]|null|undefined} path
 */
export function filterTodosByDeliverablePath(todos, categories, path) {
  const list = Array.isArray(todos) ? todos : []
  const { categoryId, taskId } = readFilterFromPath(path)
  const taskSubtree = resolveTaskSubtreeIds(list, taskId)

  // 交付物「内容」过滤优先：只按任务子树收窄，不再叠加类别闭包
  if (taskSubtree) {
    return list.filter((todo) => {
      const id = todo?.id != null ? String(todo.id) : null
      return id != null && taskSubtree.has(id)
    })
  }

  const categoryClosure = resolveCategoryIdClosure(categories, categoryId)
  return list.filter((todo) => {
    if (!categoryClosure) return true
    const did = resolveTodoDeliverableObjId(todo)
    const key = did == null ? UNCATEGORIZED_DELIVERABLE_ID : did
    return categoryClosure.has(key)
  })
}

/**
 * 当前路径下应展示的交付物分栏（含未分类占位）。
 * @param {unknown[]} categories
 * @param {DeliverablePathSegment[]|null|undefined} path
 * @param {unknown[]} todosForUncategorizedHint 用于判断是否需要「未分类」栏
 */
export function visibleDeliverableColumns(categories, path, todosForUncategorizedHint = []) {
  const cats = normalizeDeliverableCategories(categories)
  const { categoryId, taskId } = readFilterFromPath(path)
  // 内容过滤时不按类别收缩分栏，便于在子树跨类别时仍横向对照
  const effectiveCategoryId = taskId ? null : categoryId
  let visibleCats = cats
  if (effectiveCategoryId && effectiveCategoryId !== UNCATEGORIZED_DELIVERABLE_ID) {
    const selected = cats.find((c) => c.id === effectiveCategoryId)
    if (selected) {
      visibleCats = cats.filter((c) => c.order >= selected.order)
    } else {
      visibleCats = cats.filter((c) => c.id === effectiveCategoryId)
    }
  } else if (effectiveCategoryId === UNCATEGORIZED_DELIVERABLE_ID) {
    visibleCats = []
  }

  const columns = visibleCats.map((c) => ({
    id: c.id,
    name: c.name,
    color: c.color,
    order: c.order,
    kind: 'category',
  }))

  const showUncategorized =
    !effectiveCategoryId || effectiveCategoryId === UNCATEGORIZED_DELIVERABLE_ID
  if (showUncategorized) {
    const hasUncategorized = (Array.isArray(todosForUncategorizedHint) ? todosForUncategorizedHint : []).some(
      (t) => resolveTodoDeliverableObjId(t) == null,
    )
    if (hasUncategorized || effectiveCategoryId === UNCATEGORIZED_DELIVERABLE_ID || cats.length === 0) {
      columns.push({
        id: UNCATEGORIZED_DELIVERABLE_ID,
        name: UNCATEGORIZED_DELIVERABLE_NAME,
        color: '#94a3b8',
        order: Number.MAX_SAFE_INTEGER,
        kind: 'uncategorized',
      })
    }
  }

  return columns
}

/**
 * @param {unknown[]} todos
 * @param {string} columnId
 */
export function todosInDeliverableColumn(todos, columnId) {
  const list = Array.isArray(todos) ? todos : []
  const col = String(columnId)
  return list.filter((todo) => {
    const did = resolveTodoDeliverableObjId(todo)
    if (col === UNCATEGORIZED_DELIVERABLE_ID) return did == null
    return did === col
  })
}

/**
 * 根路径「全部」。
 * @returns {DeliverablePathSegment[]}
 */
export function rootDeliverablePath() {
  return [{ type: 'root' }]
}

/**
 * 追加或跳转到类别段。
 * @param {DeliverablePathSegment[]} path
 * @param {{id: string, name: string}} category
 */
export function appendCategoryToPath(path, category) {
  if (!category || category.id == null || category.id === '') {
    throw new Error('appendCategoryToPath: category.id required')
  }
  const base = Array.isArray(path) ? path.filter((s) => s && s.type !== 'task') : rootDeliverablePath()
  const withoutDup = base.filter(
    (s) => !(s.type === 'category' && String(s.id) === String(category.id)),
  )
  const next = withoutDup.length ? withoutDup.slice() : rootDeliverablePath()
  next.push({
    type: 'category',
    id: String(category.id),
    label: String(category.name ?? ''),
  })
  return next
}

/**
 * 点击面包屑第 index 段（含），截断后续。
 * @param {DeliverablePathSegment[]} path
 * @param {number} index
 */
export function truncateDeliverablePath(path, index) {
  const segments = Array.isArray(path) ? path : rootDeliverablePath()
  if (index < 0) return rootDeliverablePath()
  const sliced = segments.slice(0, index + 1)
  return sliced.length ? sliced : rootDeliverablePath()
}

/**
 * 某交付物类别分栏下，用作横向过滤的「交付物内容」列表。
 * 优先取该栏内根任务（无 parent，或 parent 不在本栏）；否则回退为栏内全部任务。
 *
 * @param {unknown[]} todos
 * @param {string} columnId
 * @returns {Array<{id: string, title: string, workspace_seq?: number}>}
 */
export function listDeliverableContentsForColumn(todos, columnId) {
  const inCol = todosInDeliverableColumn(todos, columnId)
  const byId = new Map()
  for (const todo of inCol) {
    if (todo?.id == null || todo.id === '') continue
    byId.set(String(todo.id), todo)
  }
  const roots = []
  for (const todo of inCol) {
    if (todo?.id == null || todo.id === '') continue
    const parent = resolveTodoParentTaskId(todo)
    if (!parent || !byId.has(parent)) {
      roots.push(todo)
    }
  }
  const source = roots.length > 0 ? roots : inCol
  return source.map((todo) => todoToDeliverableContent(todo)).filter(Boolean)
}

/**
 * 选中某类别下的交付物内容作为过滤路径：全部 › 类别 › 内容。
 * 实际过滤以内容（任务子树）为准。
 *
 * @param {{id: string, name?: string}|null|undefined} category
 * @param {{id: string, title?: string, name?: string}} content
 */
export function pathSelectingDeliverableContent(category, content) {
  if (!content || content.id == null || content.id === '') {
    throw new Error('pathSelectingDeliverableContent: content.id required')
  }
  const path = rootDeliverablePath()
  if (category && category.id != null && category.id !== '') {
    path.push({
      type: 'category',
      id: String(category.id),
      label: String(category.name ?? ''),
    })
  }
  path.push({
    type: 'task',
    id: String(content.id),
    label: deliverableContentOptionLabel(content) || String(content.id),
  })
  return path
}

/**
 * @param {DeliverablePathSegment[]|null|undefined} path
 * @param {string|null|undefined} contentId
 */
export function isDeliverableContentSelected(path, contentId) {
  if (contentId == null || contentId === '') return false
  const { taskId } = readFilterFromPath(path)
  return taskId != null && String(taskId) === String(contentId)
}

/**
 * 在 ancestor 任务子树内（不含自身）列出某类别下的交付物内容。
 * 无 ancestor 时等同 listDeliverableContentsForColumn。
 *
 * @param {unknown[]} todos
 * @param {string} columnId
 * @param {string|null|undefined} ancestorTaskId
 */
export function listDeliverableContentsScoped(todos, columnId, ancestorTaskId) {
  const list = Array.isArray(todos) ? todos : []
  if (ancestorTaskId == null || ancestorTaskId === '') {
    return listDeliverableContentsForColumn(list, columnId)
  }
  const subtree = resolveTaskSubtreeIds(list, ancestorTaskId)
  if (!subtree) {
    return listDeliverableContentsForColumn(list, columnId)
  }
  const scoped = list.filter((todo) => {
    const id = todo?.id != null ? String(todo.id) : null
    if (!id || !subtree.has(id)) return false
    return id !== String(ancestorTaskId)
  })
  const contents = listDeliverableContentsForColumn(scoped, columnId)
  if (contents.length > 0) return contents
  // 子树内无「根」形态时，回退为子树内该类别全部任务
  return todosInDeliverableColumn(scoped, columnId)
    .map((todo) => todoToDeliverableContent(todo))
    .filter(Boolean)
}

/**
 * 横向过滤轨迹：全部 › {类别标题 / 内容…} › {类别标题 / 内容…} › …
 * 「/」语义由 UI 用上下分割（标题在上、内容在下）表达。
 *
 * @param {unknown[]} categories
 * @param {unknown[]} todos
 * @param {DeliverablePathSegment[]|null|undefined} path
 * @returns {Array<{
 *   category: {id: string, name: string, color: string, order: number},
 *   contents: Array<{id: string, title: string, workspace_seq?: number}>,
 *   selectedContentId: string|null,
 * }>}
 */
export function buildDeliverableFilterTrail(categories, todos, path) {
  const cats = normalizeDeliverableCategories(categories)
  const { taskId } = readFilterFromPath(path)
  const selectedId = taskId

  return cats.map((category) => {
    // 若当前选中内容属于更浅类别，则用其约束更深类别的内容列表
    let ancestorId = null
    if (selectedId) {
      const selectedTodo = (Array.isArray(todos) ? todos : []).find(
        (t) => t?.id != null && String(t.id) === String(selectedId),
      )
      const selectedCatId = resolveTodoDeliverableObjId(selectedTodo)
      const selectedCat = cats.find((c) => c.id === selectedCatId)
      if (selectedCat && selectedCat.order < category.order) {
        ancestorId = selectedId
      }
    }
    const contents = listDeliverableContentsScoped(todos, category.id, ancestorId)
    const selectedContentId =
      selectedId && contents.some((c) => c.id === String(selectedId))
        ? String(selectedId)
        : selectedId &&
            (Array.isArray(todos) ? todos : []).some((t) => {
              const id = t?.id != null ? String(t.id) : null
              return (
                id === String(selectedId) &&
                resolveTodoDeliverableObjId(t) === category.id
              )
            })
          ? String(selectedId)
          : null
    return {
      category,
      contents,
      selectedContentId,
    }
  })
}
