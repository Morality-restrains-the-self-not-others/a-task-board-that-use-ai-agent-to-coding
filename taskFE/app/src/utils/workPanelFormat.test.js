// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] workPanelFormat.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    humanizeGoTaskErrorMessage,
    resolveApiErrorMessage,
  } = await import('./workPanelFormat.js')

  describe('humanizeGoTaskErrorMessage', () => {
    it('maps known Go task errors to Chinese', () => {
      expect(humanizeGoTaskErrorMessage('forbidden')).toBe('您没有权限修改此任务')
      expect(humanizeGoTaskErrorMessage('projects: one task can only link one project'))
        .toBe('一个任务只能关联一个项目')
    })

    it('humanizes base_branch required errors', () => {
      expect(humanizeGoTaskErrorMessage('projects[0] base_branch required'))
        .toBe('请为关联项目第 1 个仓库配置基准分支')
    })
  })

  describe('resolveApiErrorMessage', () => {
    it('prefers detail string', () => {
      expect(resolveApiErrorMessage({ detail: '标题不能为空' }, { fallback: '保存失败' }))
        .toBe('标题不能为空')
    })

    it('uses message when detail missing (Go/proxy 501 body)', () => {
      expect(resolveApiErrorMessage(
        { message: 'compute action not yet ported to taskCloudService: compute/github-credential-approve' },
        { fallback: '保存 GitHub 账号失败' },
      )).toBe('compute action not yet ported to taskCloudService: compute/github-credential-approve')
    })

    it('falls back to Go error field', () => {
      expect(resolveApiErrorMessage({ error: 'forbidden' }, { fallback: '保存失败' }))
        .toBe('您没有权限修改此任务')
    })

    it('formats field validation objects', () => {
      expect(resolveApiErrorMessage({ title: ['该字段不能为空。'] }, { fallback: '保存失败' }))
        .toBe('标题: 该字段不能为空。')
    })

    it('includes HTTP status when body is empty', () => {
      expect(resolveApiErrorMessage({}, { fallback: '保存失败', httpStatus: 500 }))
        .toBe('保存失败（HTTP 500）')
    })
  })
}
