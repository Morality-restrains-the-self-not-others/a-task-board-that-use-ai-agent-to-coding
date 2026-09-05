// @vitest-environment node
/**
 * 创建项目 Git 仓库批量校验：失败文案必须按原因分流，且带上请求 traceId。
 */
if (!process.env.VITEST) {
  console.log('[skip] gitRepoValidateError.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    VALIDATE_GIT_REPO_NETWORK_MSG,
    VALIDATE_GIT_REPO_TIMEOUT_MSG,
    VALIDATE_GIT_REPO_MISSING_RESULT_MSG,
    REPO_INACCESSIBLE_MSG,
    isGitRepoValidateRetryable,
    indexValidateResultsByUrl,
    resolveValidateGitReposRequestFailure,
  } = await import('./gitRepoValidateError.js')
  const { INVALID_REPO_URL_MSG } = await import('./gitRepoUrlUtils.js')

  describe('indexValidateResultsByUrl', () => {
    it('indexes by url or repo_url', () => {
      const byUrl = indexValidateResultsByUrl([
        { url: 'https://github.com/a/b.git', is_accessible: true },
        { repo_url: 'https://gitlab.com/g/p.git', is_accessible: false },
        { url: '  ', is_accessible: true },
      ])
      expect(byUrl['https://github.com/a/b.git'].is_accessible).toBe(true)
      expect(byUrl['https://gitlab.com/g/p.git'].is_accessible).toBe(false)
      expect(Object.keys(byUrl)).toHaveLength(2)
    })
  })

  describe('resolveValidateGitReposRequestFailure', () => {
    it('maps TimeoutError to timeout copy and keeps traceId', () => {
      const err = new Error('请求超时（30 秒），请检查网络后重试')
      err.name = 'TimeoutError'
      err.traceId = 'trace-timeout-1'
      const got = resolveValidateGitReposRequestFailure({ err })
      expect(got.message).toBe(VALIDATE_GIT_REPO_TIMEOUT_MSG)
      expect(got.traceId).toBe('trace-timeout-1')
    })

    it('maps transport errors to network copy and keeps traceId', () => {
      const err = new Error('Failed to fetch')
      err.traceId = 'trace-net-1'
      const got = resolveValidateGitReposRequestFailure({ err })
      expect(got.message).toBe(VALIDATE_GIT_REPO_NETWORK_MSG)
      expect(got.traceId).toBe('trace-net-1')
    })

    it('uses server error/message on HTTP failure instead of blaming the network', () => {
      const response = {
        ok: false,
        status: 400,
        traceId: 'trace-http-1',
      }
      const data = { error: 'urls 不能为空', message: 'urls 不能为空', trace_id: 'trace-http-1' }
      const got = resolveValidateGitReposRequestFailure({ response, data })
      expect(got.message).toBe('urls 不能为空')
      expect(got.traceId).toBe('trace-http-1')
      expect(got.message).not.toBe(VALIDATE_GIT_REPO_NETWORK_MSG)
    })

    it('uses missing-result copy when HTTP 200 but the row URL is absent', () => {
      const response = { ok: true, status: 200, traceId: 'trace-miss-1' }
      const got = resolveValidateGitReposRequestFailure({ response, data: { results: [] } })
      expect(got.message).toBe(VALIDATE_GIT_REPO_MISSING_RESULT_MSG)
      expect(got.traceId).toBe('trace-miss-1')
    })
  })

  describe('isGitRepoValidateRetryable', () => {
    it('is false for format and inaccessible messages', () => {
      expect(isGitRepoValidateRetryable(INVALID_REPO_URL_MSG)).toBe(false)
      expect(isGitRepoValidateRetryable(REPO_INACCESSIBLE_MSG)).toBe(false)
      expect(isGitRepoValidateRetryable('')).toBe(false)
    })

    it('is true for request-failure copies', () => {
      expect(isGitRepoValidateRetryable(VALIDATE_GIT_REPO_NETWORK_MSG)).toBe(true)
      expect(isGitRepoValidateRetryable(VALIDATE_GIT_REPO_TIMEOUT_MSG)).toBe(true)
      expect(isGitRepoValidateRetryable(VALIDATE_GIT_REPO_MISSING_RESULT_MSG)).toBe(true)
      expect(isGitRepoValidateRetryable('unauthenticated')).toBe(true)
    })
  })
}
