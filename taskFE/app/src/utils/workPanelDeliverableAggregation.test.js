import { describe, expect, it } from 'vitest'
import {
  UNCATEGORIZED_DELIVERABLE_ID,
  appendCategoryToPath,
  filterTodosByDeliverablePath,
  normalizeDeliverableCategories,
  readFilterFromPath,
  resolveCategoryIdClosure,
  resolveDeliverableCategoryDisplayName,
  resolveTodoDeliverableObjId,
  resolveTodoParentTaskId,
  isTopLevelDeliverableCategory,
  listParentDeliverableCandidates,
  rootDeliverablePath,
  listDeliverableContentsForColumn,
  pathSelectingDeliverableContent,
  isDeliverableContentSelected,
  buildDeliverableFilterTrail,
  todosInDeliverableColumn,
  truncateDeliverablePath,
  visibleDeliverableColumns,
} from './workPanelDeliverableAggregation.js'
import {
  createDeliverableFilterBar,
  defaultDeliverableFilterBars,
  insertDeliverableFilterBarBefore,
  filterBarSectionLabel,
  partitionTodosByFilterBars,
} from './workPanelDeliverableFilterBars.js'
import { buildDraggedTaskPatchPayload } from './workPanelDeliverableDragPatch.js'

const cats = [
  { id: 'c1', name: '价值流', color: '#111', order: 1 },
  { id: 'c2', name: '业务流程', color: '#222', order: 2 },
  { id: 'c3', name: '活动', color: '#333', order: 3 },
]

describe('workPanelDeliverableAggregation', () => {
  it('resolves deliverable_obj_id from nested shapes', () => {
    expect(UNCATEGORIZED_DELIVERABLE_ID).toBe('__uncategorized__')
    expect(resolveTodoDeliverableObjId({ deliverable_obj_id: 'a' })).toBe('a')
    expect(resolveTodoDeliverableObjId({ deliverable_obj: { id: 'b' } })).toBe('b')
    expect(resolveTodoDeliverableObjId({ task_type: { id: 'c' } })).toBe('c')
    expect(resolveTodoDeliverableObjId({})).toBeNull()
  })

  it('resolves deliverable category display name', () => {
    expect(resolveDeliverableCategoryDisplayName({})).toBe('未分类')
    expect(resolveDeliverableCategoryDisplayName(
      { deliverable_obj_id: 'c1' },
      cats,
    )).toBe('价值流')
    expect(resolveDeliverableCategoryDisplayName(
      { deliverable_obj: { id: 'x', name: '嵌套名' } },
      cats,
    )).toBe('嵌套名')
    expect(resolveDeliverableCategoryDisplayName(
      { deliverable_obj_id: 'missing' },
      cats,
    )).toBe('未知类别')
  })

  it('resolves parent_task id', () => {
    expect(resolveTodoParentTaskId({ parent_task: 'p1' })).toBe('p1')
    expect(resolveTodoParentTaskId({ parent_task: { id: 'p2' } })).toBe('p2')
    expect(resolveTodoParentTaskId({})).toBeNull()
  })

  it('isTopLevelDeliverableCategory uses min order', () => {
    expect(isTopLevelDeliverableCategory(cats, 'c1')).toBe(true)
    expect(isTopLevelDeliverableCategory(cats, 'c2')).toBe(false)
    expect(isTopLevelDeliverableCategory(cats, '')).toBe(true)
  })

  it('listParentDeliverableCandidates returns previous-order tasks', () => {
    const todos = [
      { id: 'v1', title: '价值流A', deliverable_obj_id: 'c1' },
      { id: 'b1', title: '过程B', deliverable_obj_id: 'c2' },
      { id: 'a1', title: '活动C', deliverable_obj_id: 'c3' },
    ]
    expect(listParentDeliverableCandidates(todos, cats, 'c1')).toEqual([])
    expect(listParentDeliverableCandidates(todos, cats, 'c2')).toEqual([
      { id: 'v1', title: '价值流A' },
    ])
    expect(
      listParentDeliverableCandidates(
        [{ id: 'v1', title: '价值流A', deliverable_obj_id: 'c1', workspace_seq: 9 }],
        cats,
        'c2',
      ),
    ).toEqual([{ id: 'v1', title: '价值流A', workspace_seq: 9 }])
    expect(listParentDeliverableCandidates(todos, cats, 'c3').map((t) => t.id)).toEqual(['b1'])
  })

  it('normalizes and sorts categories by order', () => {
    const n = normalizeDeliverableCategories([
      { id: 'c3', name: '活动', order: 3 },
      { id: 'c1', name: '价值流', order: 1 },
    ])
    expect(n.map((c) => c.id)).toEqual(['c1', 'c3'])
  })

  it('category closure includes selected and deeper orders', () => {
    const set = resolveCategoryIdClosure(cats, 'c2')
    expect([...set].sort()).toEqual(['c2', 'c3'])
    expect(resolveCategoryIdClosure(cats, null)).toBeNull()
    expect([...resolveCategoryIdClosure(cats, UNCATEGORIZED_DELIVERABLE_ID)]).toEqual([
      UNCATEGORIZED_DELIVERABLE_ID,
    ])
  })

  it('filters by category path including child categories', () => {
    const todos = [
      { id: 't1', deliverable_obj_id: 'c1' },
      { id: 't2', deliverable_obj_id: 'c2' },
      { id: 't3', deliverable_obj_id: 'c3' },
      { id: 't4' },
    ]
    const path = appendCategoryToPath(rootDeliverablePath(), cats[1])
    const filtered = filterTodosByDeliverablePath(todos, cats, path)
    expect(filtered.map((t) => t.id).sort()).toEqual(['t2', 't3'])
  })

  it('filters by task subtree via parent_task', () => {
    const todos = [
      { id: 'root', deliverable_obj_id: 'c1' },
      { id: 'child', deliverable_obj_id: 'c2', parent_task: 'root' },
      { id: 'other', deliverable_obj_id: 'c1' },
    ]
    const path = [
      { type: 'root' },
      { type: 'task', id: 'root', label: 'Root' },
    ]
    const filtered = filterTodosByDeliverablePath(todos, cats, path)
    expect(filtered.map((t) => t.id).sort()).toEqual(['child', 'root'])
  })

  it('groups todos into deliverable columns and visible columns', () => {
    const todos = [
      { id: 't1', deliverable_obj_id: 'c1' },
      { id: 't2' },
    ]
    expect(todosInDeliverableColumn(todos, 'c1').map((t) => t.id)).toEqual(['t1'])
    expect(todosInDeliverableColumn(todos, UNCATEGORIZED_DELIVERABLE_ID).map((t) => t.id)).toEqual([
      't2',
    ])
    const cols = visibleDeliverableColumns(cats, rootDeliverablePath(), todos)
    expect(cols.map((c) => c.id)).toEqual(['c1', 'c2', 'c3', UNCATEGORIZED_DELIVERABLE_ID])
  })

  it('shrinks visible columns after category filter', () => {
    const path = appendCategoryToPath(rootDeliverablePath(), cats[1])
    const cols = visibleDeliverableColumns(cats, path, [])
    expect(cols.map((c) => c.id)).toEqual(['c2', 'c3'])
  })

  it('path helpers: root / append / truncate / readFilter', () => {
    const root = rootDeliverablePath()
    expect(readFilterFromPath(root)).toEqual({ categoryId: null, taskId: null })
    const withCat = appendCategoryToPath(root, { id: 'c1', name: '价值流' })
    expect(readFilterFromPath(withCat).categoryId).toBe('c1')
    const truncated = truncateDeliverablePath(withCat, 0)
    expect(readFilterFromPath(truncated).categoryId).toBeNull()
  })

  it('lists deliverable contents under a category (prefer roots)', () => {
    const todos = [
      { id: 'root', title: '价值流A', deliverable_obj_id: 'c1' },
      { id: 'child', title: '子项', deliverable_obj_id: 'c1', parent_task: 'root' },
      { id: 'other', title: '价值流B', deliverable_obj_id: 'c1' },
      { id: 'x', title: '流程X', deliverable_obj_id: 'c2' },
    ]
    const contents = listDeliverableContentsForColumn(todos, 'c1')
    expect(contents.map((c) => c.id).sort()).toEqual(['other', 'root'])
    expect(contents.find((c) => c.id === 'root').title).toBe('价值流A')
  })

  it('lists deliverable contents with workspace_seq when present', () => {
    const todos = [
      { id: 'root', title: 'hello', deliverable_obj_id: 'c1', workspace_seq: 12 },
      { id: 'other', title: 'hello', deliverable_obj_id: 'c1', workspace_seq: 13 },
    ]
    const contents = listDeliverableContentsForColumn(todos, 'c1')
    expect(contents.find((c) => c.id === 'root')).toEqual({
      id: 'root',
      title: 'hello',
      workspace_seq: 12,
    })
    expect(contents.find((c) => c.id === 'other').workspace_seq).toBe(13)
  })

  it('content path label includes task number when workspace_seq present', () => {
    const path = pathSelectingDeliverableContent(cats[0], {
      id: 'root',
      title: '价值流A',
      workspace_seq: 12,
    })
    expect(path.find((s) => s.type === 'task').label).toBe('#12 价值流A')
  })

  it('content path filters by subtree only (category in path is label)', () => {
    const todos = [
      { id: 'root', deliverable_obj_id: 'c1' },
      { id: 'child', deliverable_obj_id: 'c2', parent_task: 'root' },
      { id: 'other', deliverable_obj_id: 'c1' },
    ]
    const path = pathSelectingDeliverableContent(cats[0], { id: 'root', title: '价值流A' })
    expect(readFilterFromPath(path)).toEqual({ categoryId: 'c1', taskId: 'root' })
    const filtered = filterTodosByDeliverablePath(todos, cats, path)
    expect(filtered.map((t) => t.id).sort()).toEqual(['child', 'root'])
    expect(isDeliverableContentSelected(path, 'root')).toBe(true)
    expect(isDeliverableContentSelected(path, 'other')).toBe(false)
  })

  it('content filter does not shrink visible columns by category', () => {
    const path = pathSelectingDeliverableContent(cats[0], { id: 'root', title: 'A' })
    const cols = visibleDeliverableColumns(cats, path, [{ id: 'root', deliverable_obj_id: 'c1' }])
    expect(cols.map((c) => c.id)).toEqual(['c1', 'c2', 'c3'])
  })

  it('buildDeliverableFilterTrail: category title over contents per level', () => {
    const todos = [
      { id: 'v1', title: '价值流1', deliverable_obj_id: 'c1' },
      { id: 'v2', title: '价值流2', deliverable_obj_id: 'c1' },
      { id: 'b1', title: '业务过程1', deliverable_obj_id: 'c2', parent_task: 'v1' },
      { id: 'b2', title: '业务过程2', deliverable_obj_id: 'c2', parent_task: 'v2' },
    ]
    const trail = buildDeliverableFilterTrail(cats, todos, rootDeliverablePath())
    expect(trail.map((l) => l.category.name)).toEqual(['价值流', '业务流程', '活动'])
    expect(trail[0].contents.map((c) => c.title).sort()).toEqual(['价值流1', '价值流2'])
    expect(trail[1].contents.map((c) => c.id).sort()).toEqual(['b1', 'b2'])
  })

  it('buildDeliverableFilterTrail: scopes deeper contents by selected ancestor', () => {
    const todos = [
      { id: 'v1', title: '价值流1', deliverable_obj_id: 'c1' },
      { id: 'v2', title: '价值流2', deliverable_obj_id: 'c1' },
      { id: 'b1', title: '业务过程1', deliverable_obj_id: 'c2', parent_task: 'v1' },
      { id: 'b2', title: '业务过程2', deliverable_obj_id: 'c2', parent_task: 'v2' },
    ]
    const path = pathSelectingDeliverableContent(cats[0], { id: 'v1', title: '价值流1' })
    const trail = buildDeliverableFilterTrail(cats, todos, path)
    expect(trail[0].selectedContentId).toBe('v1')
    expect(trail[1].contents.map((c) => c.id)).toEqual(['b1'])
  })

  it('createDeliverableFilterBar / defaultDeliverableFilterBars', () => {
    const bars = defaultDeliverableFilterBars()
    expect(bars).toHaveLength(1)
    expect(bars[0].id).toBe('bar-0')
    expect(readFilterFromPath(bars[0].path)).toEqual({ categoryId: null, taskId: null })
    const extra = createDeliverableFilterBar()
    expect(extra.id).toMatch(/^bar-/)
    expect(extra.id).not.toBe('bar-0')
  })

  it('insertDeliverableFilterBarBefore: inserts above the target bar', () => {
    const bars = [
      createDeliverableFilterBar('bar-a'),
      createDeliverableFilterBar('bar-b'),
    ]
    const next = insertDeliverableFilterBarBefore(bars, 'bar-b', createDeliverableFilterBar('bar-new'))
    expect(next.map((b) => b.id)).toEqual(['bar-a', 'bar-new', 'bar-b'])
    expect(bars.map((b) => b.id)).toEqual(['bar-a', 'bar-b'])
  })

  it('insertDeliverableFilterBarBefore: inserts above first bar', () => {
    const bars = [createDeliverableFilterBar('bar-0')]
    const next = insertDeliverableFilterBarBefore(bars, 'bar-0', createDeliverableFilterBar('bar-top'))
    expect(next.map((b) => b.id)).toEqual(['bar-top', 'bar-0'])
  })

  it('insertDeliverableFilterBarBefore: appends when beforeBarId missing', () => {
    const bars = [createDeliverableFilterBar('bar-0')]
    const next = insertDeliverableFilterBarBefore(bars, 'missing', createDeliverableFilterBar('bar-end'))
    expect(next.map((b) => b.id)).toEqual(['bar-0', 'bar-end'])
  })

  it('partitionTodosByFilterBars: inactive bars leave all in 其他', () => {
    const todos = [
      { id: 'v1', title: '价值流1', deliverable_obj_id: 'c1' },
      { id: 'v2', title: '价值流2', deliverable_obj_id: 'c1' },
    ]
    const bars = defaultDeliverableFilterBars()
    const { other, sections } = partitionTodosByFilterBars(todos, cats, bars)
    expect(sections).toEqual([])
    expect(other.map((t) => t.id).sort()).toEqual(['v1', 'v2'])
  })

  it('partitionTodosByFilterBars: active bars split 其他 vs filter sections', () => {
    const todos = [
      { id: 'v1', title: '价值流1', deliverable_obj_id: 'c1' },
      { id: 'child', title: '子', deliverable_obj_id: 'c2', parent_task: 'v1' },
      { id: 'v2', title: '价值流2', deliverable_obj_id: 'c1' },
      { id: 'orphan', title: '游离', deliverable_obj_id: 'c2' },
    ]
    const bars = [
      {
        id: 'bar-a',
        path: pathSelectingDeliverableContent(cats[0], { id: 'v1', title: '价值流1' }),
      },
      createDeliverableFilterBar('bar-idle'),
    ]
    const { other, sections } = partitionTodosByFilterBars(todos, cats, bars)
    expect(sections).toHaveLength(1)
    expect(sections[0].barId).toBe('bar-a')
    expect(sections[0].label).toBe('价值流1')
    expect(sections[0].todos.map((t) => t.id).sort()).toEqual(['child', 'v1'])
    expect(other.map((t) => t.id).sort()).toEqual(['orphan', 'v2'])
  })

  it('partitionTodosByFilterBars: overlapping filters keep tasks in each section; 其他 is residual', () => {
    const todos = [
      { id: 'v1', title: '价值流1', deliverable_obj_id: 'c1' },
      { id: 'v2', title: '价值流2', deliverable_obj_id: 'c1' },
      { id: 'solo', title: '独立', deliverable_obj_id: 'c2' },
    ]
    const bars = [
      {
        id: 'b1',
        path: pathSelectingDeliverableContent(cats[0], { id: 'v1', title: '价值流1' }),
      },
      {
        id: 'b2',
        path: pathSelectingDeliverableContent(cats[0], { id: 'v2', title: '价值流2' }),
      },
    ]
    const { other, sections } = partitionTodosByFilterBars(todos, cats, bars)
    expect(sections.map((s) => s.label)).toEqual(['价值流1', '价值流2'])
    expect(sections[0].todos.map((t) => t.id)).toEqual(['v1'])
    expect(sections[1].todos.map((t) => t.id)).toEqual(['v2'])
    expect(other.map((t) => t.id)).toEqual(['solo'])
  })

  it('filterBarSectionLabel prefers deepest task label', () => {
    const path = pathSelectingDeliverableContent(cats[0], { id: 'v1', title: '价值流1' })
    expect(filterBarSectionLabel(path)).toBe('价值流1')
    expect(filterBarSectionLabel(rootDeliverablePath())).toBe('未命名过滤')
  })

  it('buildDraggedTaskPatchPayload: same deliverable only updates progress', () => {
    expect(
      buildDraggedTaskPatchPayload({
        draggedTaskId: 't1',
        cardTaskId: 't1',
        order: 2,
        targetProgressColumnId: 'p2',
        sourceDeliverableColumnId: 'c1',
        targetDeliverableColumnId: 'c1',
      }),
    ).toEqual({ order: 2, progress_column_id: 'p2' })
  })

  it('buildDraggedTaskPatchPayload: cross deliverable updates deliverable_obj_id', () => {
    expect(
      buildDraggedTaskPatchPayload({
        draggedTaskId: 't1',
        cardTaskId: 't1',
        order: 0,
        targetProgressColumnId: 'p1',
        sourceDeliverableColumnId: 'c1',
        targetDeliverableColumnId: 'c2',
      }),
    ).toEqual({
      order: 0,
      progress_column_id: 'p1',
      deliverable_obj_id: 'c2',
    })
  })

  it('buildDraggedTaskPatchPayload: drop on uncategorized clears deliverable_obj_id', () => {
    expect(
      buildDraggedTaskPatchPayload({
        draggedTaskId: 't1',
        cardTaskId: 't1',
        order: 1,
        targetProgressColumnId: 'p1',
        sourceDeliverableColumnId: 'c1',
        targetDeliverableColumnId: UNCATEGORIZED_DELIVERABLE_ID,
      }),
    ).toEqual({
      order: 1,
      progress_column_id: 'p1',
      deliverable_obj_id: null,
    })
  })

  it('buildDraggedTaskPatchPayload: sibling cards only get order', () => {
    expect(
      buildDraggedTaskPatchPayload({
        draggedTaskId: 't1',
        cardTaskId: 't2',
        order: 3,
        targetProgressColumnId: 'p1',
        sourceDeliverableColumnId: 'c1',
        targetDeliverableColumnId: 'c2',
      }),
    ).toEqual({ order: 3 })
  })

})
