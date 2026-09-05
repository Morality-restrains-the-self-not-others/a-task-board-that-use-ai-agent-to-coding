// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] titleTranslationError.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  TITLE_TRANSLATION_USER_ERROR,
  TITLE_TRANSLATION_LOCAL_HINT,
  userFacingTitleTranslationError,
  isTitleTranslateAbortError,
} = await import('./titleTranslationError.js')

describe('userFacingTitleTranslationError', () => {
  it('maps empty fanyi JSON parse dump to a user-safe fallback hint', () => {
    const raw = '任务标题翻译失败: fanyi_agent 响应无效: unexpected end of JSON input'
    expect(userFacingTitleTranslationError(raw)).toBe(TITLE_TRANSLATION_USER_ERROR)
    expect(userFacingTitleTranslationError(raw)).not.toContain('unexpected end of JSON')
    expect(userFacingTitleTranslationError(raw)).not.toContain('fanyi_agent')
  })

  it('keeps non-technical backend copy such as method not allowed', () => {
    expect(userFacingTitleTranslationError('method not allowed')).toBe('method not allowed')
  })

  it('maps backend user-safe 502 copy to the fallback hint after local branch naming', () => {
    expect(userFacingTitleTranslationError('任务标题翻译失败，请稍后重试或手动填写分支名'))
      .toBe(TITLE_TRANSLATION_USER_ERROR)
  })

  it('preserves categorized why from backend and appends local-branch hint', () => {
    const raw = '任务标题自动翻译失败：翻译服务未返回可用译文'
    const got = userFacingTitleTranslationError(raw)
    expect(got).toContain('未返回可用译文')
    expect(got).toContain(TITLE_TRANSLATION_LOCAL_HINT)
    expect(got).not.toContain('fanyi_agent')
    expect(got).not.toContain('finish_reason')
  })

  it('does not double-append local hint when already present', () => {
    const raw = `任务标题自动翻译失败：翻译服务响应超时。${TITLE_TRANSLATION_LOCAL_HINT}`
    expect(userFacingTitleTranslationError(raw)).toBe(raw)
  })

  it('classifies fetch AbortError so overlapping title translates stay silent', () => {
    const err = new Error('aborted')
    err.name = 'AbortError'
    expect(isTitleTranslateAbortError(err)).toBe(true)
    expect(isTitleTranslateAbortError(new Error('任务标题翻译失败'))).toBe(false)
  })
})
}
