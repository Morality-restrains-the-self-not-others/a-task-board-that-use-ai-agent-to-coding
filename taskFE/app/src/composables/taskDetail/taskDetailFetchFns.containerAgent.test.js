// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFetchFns.containerAgent.test.js requires vitest runtime')
} else {
const { describe, expect, it, vi, beforeEach, afterEach } = await import('vitest')
const { ref } = await import('vue')

vi.mock('../../utils/apiUtils.js', () => ({
  apiFetch: vi.fn(),
}))

const { apiFetch } = await import('../../utils/apiUtils.js')
const {
  COMMENT_FEED_PAGE_LIMIT,
  fetchTaskDetail,
  loadMoreCommentFeeds,
} = await import('./taskDetailFetchFns.js')

describe('fetchTaskDetail — comment feeds', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('parallel-fetches human, ai, and container-agent comments with page limit', async () => {
    const localTask = ref(null)
    apiFetch.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('/todos/')) {
        return {
          ok: true,
          json: async () => ({
            id: 'task1',
            title: 'T',
            comments: [],
            ai_comments: [],
          }),
        }
      }
      if (u.includes('/tasks/') && u.includes('/comments/')) {
        expect(u).toContain(`limit=${COMMENT_FEED_PAGE_LIMIT}`)
        return {
          ok: true,
          json: async () => ({
            results: [{ id: 'h1', content: 'human', created_at: '2026-07-15T09:00:00Z' }],
            next_cursor: 'cur-h',
            has_more: true,
          }),
        }
      }
      if (u.includes('/ai-comments/')) {
        expect(u).toContain(`limit=${COMMENT_FEED_PAGE_LIMIT}`)
        return {
          ok: true,
          json: async () => ({
            results: [{ id: 'a1', content: 'ai', created_at: '2026-07-15T10:00:00Z' }],
            next_cursor: null,
            has_more: false,
          }),
        }
      }
      if (u.includes('/container-agent-comments')) {
        expect(u).toContain(`limit=${COMMENT_FEED_PAGE_LIMIT}`)
        return {
          ok: true,
          json: async () => ({
            results: [
              { id: 'ca1', content: '@img', assistant_response: 'done', created_at: '2026-07-15T11:00:00Z' },
            ],
            next_cursor: 'cur-ca',
            has_more: true,
          }),
        }
      }
      return { ok: false, status: 404 }
    })

    await fetchTaskDetail({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask,
      taskDetailLoading: ref(false),
      taskDetailLoadError: ref(''),
      syncRepoCloneIdentityMapFromTask: vi.fn(),
    })

    expect(localTask.value.comments).toHaveLength(1)
    expect(localTask.value.ai_comments).toHaveLength(1)
    expect(localTask.value.container_agent_comments).toHaveLength(1)
    expect(localTask.value.comments_feeds_loaded).toBe(true)
    expect(localTask.value.comments_next_cursor).toBe('cur-h')
    expect(localTask.value.comments_has_more).toBe(true)
    expect(localTask.value.container_agent_comments_has_more).toBe(true)
    expect(apiFetch).toHaveBeenCalledTimes(4)
  })

  it('prefers successful feed over detail payload even when feed is empty', async () => {
    const localTask = ref(null)
    apiFetch.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('/todos/')) {
        return {
          ok: true,
          json: async () => ({
            id: 'task1',
            comments: [{ id: 'pre', content: 'from detail' }],
            ai_comments: [{ id: 'ai-pre', content: 'ai detail' }],
            container_agent_comments: [{ id: 'pre' }],
          }),
        }
      }
      if (u.includes('/comments/') || u.includes('/ai-comments/') || u.includes('/container-agent-comments')) {
        return {
          ok: true,
          json: async () => ({ results: [], next_cursor: null, has_more: false }),
        }
      }
      return { ok: false, status: 404 }
    })

    await fetchTaskDetail({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask,
      taskDetailLoading: ref(false),
      taskDetailLoadError: ref(''),
      syncRepoCloneIdentityMapFromTask: vi.fn(),
    })

    expect(localTask.value.comments).toEqual([])
    expect(localTask.value.ai_comments).toEqual([])
    expect(localTask.value.container_agent_comments).toEqual([])
    expect(localTask.value.comments_has_more).toBe(false)
  })

  it('sets comments_feed_errors when a feed fails', async () => {
    const localTask = ref(null)
    apiFetch.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('/todos/')) {
        return {
          ok: true,
          json: async () => ({ id: 'task1', comments: [], ai_comments: [] }),
        }
      }
      if (u.includes('/ai-comments/')) {
        return { ok: false, status: 503 }
      }
      if (u.includes('/comments/')) {
        return { ok: true, json: async () => ({ results: [], next_cursor: null, has_more: false }) }
      }
      if (u.includes('/container-agent-comments')) {
        return { ok: true, json: async () => ({ results: [], next_cursor: null, has_more: false }) }
      }
      return { ok: false, status: 404 }
    })

    await fetchTaskDetail({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask,
      taskDetailLoading: ref(false),
      taskDetailLoadError: ref(''),
      syncRepoCloneIdentityMapFromTask: vi.fn(),
    })

    expect(localTask.value.comments_feed_errors).toEqual(
      expect.arrayContaining([expect.stringContaining('ai_comments')]),
    )
    expect(localTask.value.comments).toEqual([])
  })
})

describe('loadMoreCommentFeeds', () => {
  beforeEach(() => {
    apiFetch.mockReset()
  })

  it('appends older pages and updates cursors without duplicates', async () => {
    const localTask = ref({
      id: 'task1',
      comments: [{ id: 'h1', content: 'newest', created_at: '2026-07-15T12:00:00Z' }],
      comments_next_cursor: 'cur-h',
      comments_has_more: true,
      ai_comments: [],
      ai_comments_has_more: false,
      container_agent_comments: [],
      container_agent_comments_has_more: false,
    })
    const commentsFeedsLoadingMore = ref(false)

    apiFetch.mockImplementation(async (url) => {
      const u = String(url)
      if (u.includes('/tasks/') && u.includes('/comments/')) {
        expect(u).toContain('cursor=cur-h')
        expect(u).toContain(`limit=${COMMENT_FEED_PAGE_LIMIT}`)
        return {
          ok: true,
          json: async () => ({
            results: [
              { id: 'h0', content: 'older', created_at: '2026-07-15T08:00:00Z' },
              { id: 'h1', content: 'dup', created_at: '2026-07-15T12:00:00Z' },
            ],
            next_cursor: null,
            has_more: false,
          }),
        }
      }
      return { ok: false, status: 404 }
    })

    await loadMoreCommentFeeds({
      effectiveTenantId: ref('t1'),
      effectiveWorkspaceId: ref('ws1'),
      effectiveTaskId: ref('task1'),
      localTask,
      commentsFeedsLoadingMore,
    })

    expect(localTask.value.comments).toHaveLength(2)
    expect(localTask.value.comments.map((c) => c.id)).toEqual(['h1', 'h0'])
    expect(localTask.value.comments_has_more).toBe(false)
    expect(localTask.value.comments_next_cursor).toBeNull()
    expect(commentsFeedsLoadingMore.value).toBe(false)
  })
})

}
