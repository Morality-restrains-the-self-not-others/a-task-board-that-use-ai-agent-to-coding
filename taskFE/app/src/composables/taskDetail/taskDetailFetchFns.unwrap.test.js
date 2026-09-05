// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.unwrap.test.js requires vitest runtime')
} else {
  const { describe, expect, it, vi, beforeEach } = await import('vitest')
  const { ref } = await import('vue')

  vi.mock('../../utils/apiUtils.js', () => ({
    apiFetch: vi.fn(),
  }))

  const { apiFetch } = await import('../../utils/apiUtils.js')
  const { fetchTaskDetail } = await import('./taskDetailFetchFns.js')

  const TASK_ID = 'task_881388002226499584'
  const PROJECT_ROW = {
    project_id: 'proj_881195029417193472',
    project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
    base_branch: 'master',
  }

  function emptyFeedOk() {
    return { ok: true, json: async () => ({ results: [], next_cursor: null, has_more: false }) }
  }

  describe('fetchTaskDetail — unwrap list/envelope GET', () => {
    beforeEach(() => {
      apiFetch.mockReset()
    })

    it('does not assign a workspace list as localTask; picks the matching task with projects', async () => {
      const localTask = ref(null)
      apiFetch.mockImplementation(async (url) => {
        const u = String(url)
        if (u.includes('/todos/')) {
          return {
            ok: true,
            json: async () => ([
              { id: 'task_other', title: 'other', projects: [] },
              { id: TASK_ID, title: '用 js 写一个 hello world', projects: [PROJECT_ROW] },
            ]),
          }
        }
        return emptyFeedOk()
      })

      await fetchTaskDetail({
        effectiveTenantId: ref('881024523581812736'),
        effectiveWorkspaceId: ref('ws_881024527847419904'),
        effectiveTaskId: ref(TASK_ID),
        localTask,
        taskDetailLoading: ref(false),
        taskDetailLoadError: ref(''),
        syncRepoCloneIdentityMapFromTask: vi.fn(),
      })

      expect(Array.isArray(localTask.value)).toBe(false)
      expect(localTask.value.id).toBe(TASK_ID)
      expect(localTask.value.projects).toEqual([PROJECT_ROW])
      const called = apiFetch.mock.calls.map((c) => String(c[0] || ''))
      expect(called.some((u) => u.includes('validate-git-repos'))).toBe(false)
    })

    it('rejects a list that does not contain the requested task', async () => {
      const localTask = ref(null)
      const loadError = ref('')
      apiFetch.mockImplementation(async (url) => {
        if (String(url).includes('/todos/')) {
          return { ok: true, json: async () => ([{ id: 'task_other', projects: [] }]) }
        }
        return emptyFeedOk()
      })

      await fetchTaskDetail({
        effectiveTenantId: ref('t1'),
        effectiveWorkspaceId: ref('ws1'),
        effectiveTaskId: ref(TASK_ID),
        localTask,
        taskDetailLoading: ref(false),
        taskDetailLoadError: loadError,
        syncRepoCloneIdentityMapFromTask: vi.fn(),
      })

      expect(localTask.value).toBeNull()
      expect(loadError.value).toContain('任务对象')
    })
  })
}
