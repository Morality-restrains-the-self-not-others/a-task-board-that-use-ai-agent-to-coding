import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import { fetchProjects, fetchTaskTypes, fetchInstalledImages, fetchTodos, shouldShowEmptyTaskHint } from './workPanelDataFetch.js'

describe('workPanelDataFetch error data-traceId', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  const baseDeps = (overrides = {}) => {
    const apiFetch = vi.fn()
    return {
      tenantId: ref('t1'),
      currentWorkspace: ref({ id: 'ws1' }),
      todos: ref([]),
      todosError: ref(null),
      todosErrorTraceId: ref(''),
      filterOptions: ref({ priority: null, search: '' }),
      taskTypes: ref([]),
      isTaskTypesLoading: ref(false),
      taskTypesError: ref(null),
      taskTypesErrorTraceId: ref(''),
      projects: ref([]),
      isProjectsLoading: ref(false),
      projectsError: ref(null),
      projectsErrorTraceId: ref(''),
      installedImages: ref([]),
      isInstalledImagesLoading: ref(false),
      installedImagesError: ref(null),
      installedImagesErrorTraceId: ref(''),
      apiFetch,
      parseJsonSafe: async (r) => {
        try {
          return await r.json()
        } catch {
          return null
        }
      },
      normalizeListPayload: (d) => (Array.isArray(d) ? d : d?.results || []),
      warnOptionalApiFailure: vi.fn(),
      warnNetworkFailure: vi.fn(),
      formatApiErrorMessage: () => '',
      extractTraceId: (s) => (s?.traceId ? String(s.traceId) : ''),
      ...overrides,
    }
  }

  it('stores projectsErrorTraceId on HTTP failure', async () => {
    const deps = baseDeps()
    deps.apiFetch.mockResolvedValue({
      ok: false,
      status: 500,
      traceId: 'tid-projects-1',
      json: async () => ({ error: 'boom' }),
    })
    await fetchProjects(deps)
    expect(deps.projectsError.value).toMatch(/获取项目失败/)
    expect(deps.projectsErrorTraceId.value).toBe('tid-projects-1')
  })

  it('stores taskTypesErrorTraceId on network failure', async () => {
    const deps = baseDeps()
    const err = new Error('offline')
    err.traceId = 'tid-types-net'
    deps.apiFetch.mockRejectedValue(err)
    await fetchTaskTypes(deps)
    expect(deps.taskTypesError.value).toBe('网络错误，请稍后重试')
    expect(deps.taskTypesErrorTraceId.value).toBe('tid-types-net')
  })

  it('stores installedImagesErrorTraceId on HTTP failure', async () => {
    const deps = baseDeps()
    deps.apiFetch.mockResolvedValue({
      ok: false,
      status: 403,
      traceId: 'tid-images-1',
      json: async () => ({}),
    })
    await fetchInstalledImages(deps)
    expect(deps.installedImagesError.value).toMatch(/获取已安装镜像失败/)
    expect(deps.installedImagesErrorTraceId.value).toBe('tid-images-1')
  })

  it('stores todosErrorTraceId on HTTP failure and does not treat as empty success', async () => {
    const deps = baseDeps()
    deps.formatApiErrorMessage = (body) => body?.message || ''
    deps.apiFetch.mockResolvedValue({
      ok: false,
      status: 404,
      traceId: 'tid-todos-404',
      json: async () => ({ message: 'workspace not found', trace_id: 'tid-todos-404' }),
    })
    await fetchTodos(deps)
    expect(deps.todos.value).toEqual([])
    expect(deps.todosError.value).toMatch(/workspace not found|任务列表/)
    expect(deps.todosErrorTraceId.value).toBe('tid-todos-404')
  })

  it('clears todosError after a successful list', async () => {
    const deps = baseDeps()
    deps.todosError.value = 'stale'
    deps.todosErrorTraceId.value = 'old'
    deps.apiFetch.mockResolvedValue({
      ok: true,
      json: async () => [{ id: 'task_1', title: 'alive' }],
    })
    await fetchTodos(deps)
    expect(deps.todos.value).toEqual([{ id: 'task_1', title: 'alive' }])
    expect(deps.todosError.value).toBeNull()
    expect(deps.todosErrorTraceId.value).toBe('')
  })
})

describe('shouldShowEmptyTaskHint', () => {
  it('is false when task list failed to load', () => {
    expect(shouldShowEmptyTaskHint({
      workspaceId: 'ws1',
      statusCount: 4,
      todoCount: 0,
      loadError: 'workspace not found',
    })).toBe(false)
  })

  it('is true when load succeeded and board is empty', () => {
    expect(shouldShowEmptyTaskHint({
      workspaceId: 'ws1',
      statusCount: 4,
      todoCount: 0,
      loadError: null,
    })).toBe(true)
  })
})
