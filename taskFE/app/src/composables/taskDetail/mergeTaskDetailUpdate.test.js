// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] mergeTaskDetailUpdate.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { mergeTaskDetailUpdate } = await import('./mergeTaskDetailUpdate.js')

  describe('mergeTaskDetailUpdate', () => {
    it('同一任务 PATCH comments=[] 时保留已加载评论', () => {
      const current = {
        id: 'task_1',
        auto_run: true,
        comments: [{ id: 'c1' }],
        comments_feeds_loaded: true,
      }
      const incoming = {
        id: 'task_1',
        auto_run: true,
        auto_run_start_skipped: false,
        comments: [],
      }
      const next = mergeTaskDetailUpdate(current, incoming)
      expect(next.comments).toEqual([{ id: 'c1' }])
      expect(next.comments_feeds_loaded).toBe(true)
      expect(next.auto_run_start_skipped).toBe(false)
    })

    it('incoming 为 null 时清空（关闭详情）', () => {
      const current = { id: 'task_1', comments: [{ id: 'c1' }] }
      expect(mergeTaskDetailUpdate(current, null)).toBeNull()
    })

    it('切换任务时不把旧评论带到新任务', () => {
      const current = {
        id: 'task_1',
        comments: [{ id: 'c1' }],
        comments_feeds_loaded: true,
      }
      const incoming = {
        id: 'task_2',
        auto_run: true,
        comments: [],
      }
      const next = mergeTaskDetailUpdate(current, incoming)
      expect(next.id).toBe('task_2')
      expect(next.comments).toEqual([])
      expect(next.comments_feeds_loaded).toBeUndefined()
    })

    it('同一任务 PATCH projects=[] 时保留已加载关联项目', () => {
      const current = {
        id: 'task_881388002226499584',
        title: '用 js 写一个 hello world',
        projects: [{
          project_id: 'proj_881195029417193472',
          stored_repo_address: 'https://github.com/ruandao/helloworld',
          project_repo_url: 'https://github.com/test-ruandao/helloworld.git',
        }],
      }
      const incoming = {
        id: 'task_881388002226499584',
        title: '用 js 写一个 hello world',
        projects: [],
        feature_params_source: 'workspace',
      }
      const next = mergeTaskDetailUpdate(current, incoming)
      expect(next.projects).toEqual(current.projects)
      expect(next.feature_params_source).toBe('workspace')
    })

    it('无 id 的非任务 payload 不得覆盖已加载任务', () => {
      const current = {
        id: 'task_881388002226499584',
        projects: [{ project_id: 'proj_881195029417193472' }],
      }
      const next = mergeTaskDetailUpdate(current, { ok: true, status: 'started' })
      expect(next).toEqual(current)
    })
  })
}
