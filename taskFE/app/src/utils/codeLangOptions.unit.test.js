// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] codeLangOptions.unit.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    DEFAULT_CODE_LANG_OPTIONS,
    normalizeCodeLangOptions,
    codeLangOptionsFromResponse,
  } = await import('./codeLangOptions.js')

  describe('codeLangOptions', () => {
    it('默认值为 go / rust / js', () => {
      expect(DEFAULT_CODE_LANG_OPTIONS).toEqual(['go', 'rust', 'js'])
      expect(normalizeCodeLangOptions([])).toEqual(['go', 'rust', 'js'])
    })

    it('去重并保留首次大小写', () => {
      expect(normalizeCodeLangOptions([' Go ', 'GO', 'rust', ''])).toEqual(['Go', 'rust'])
    })

    it('从 API 响应解析 options', () => {
      expect(codeLangOptionsFromResponse({ options: ['python'] })).toEqual(['python'])
      expect(codeLangOptionsFromResponse(null)).toEqual(['go', 'rust', 'js'])
    })
  })
}
