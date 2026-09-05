// @vitest-environment jsdom
if (!process.env.VITEST) {
  // pre-commit 会以 node 直跑已暂存 *.test.js，非 Vitest 环境下跳过。
  console.log('[skip] rgDeepLink.test.js requires vitest runtime')
} else {
  const { describe, it, expect, vi, beforeEach, afterEach } = await import('vitest')

  const {
    parseRgKey,
    findRgTarget,
    scrollToRgKey,
    handleRgDeepLink,
  } = await import('./rgDeepLink.js')

  describe('rgDeepLink', () => {
    beforeEach(() => {
      vi.useFakeTimers()
      document.body.innerHTML = ''
      // jsdom 不实现 scrollIntoView 行为，测试中以 no-op stub 代替
      Element.prototype.scrollIntoView = vi.fn()
    })
    afterEach(() => {
      vi.useRealTimers()
    })

    it('parseRgKey 解析 #rg= 深链（含 &rg= 与空值）', () => {
      expect(parseRgKey('#rg=settings.cloud.main')).toBe('settings.cloud.main')
      expect(parseRgKey('#foo=1&rg=people.access')).toBe('people.access')
      expect(parseRgKey('#tab=1')).toBe('')
      expect(parseRgKey('')).toBe('')
      expect(parseRgKey('#rg=settings.cloud%2Fmain')).toBe('settings.cloud/main')
    })

    it('findRgTarget 按 data-rg-key 属性匹配元素', () => {
      const a = document.createElement('section')
      a.setAttribute('data-rg-key', 'people.access.subject_list')
      document.body.appendChild(a)
      expect(findRgTarget('people.access.subject_list')).toBe(a)
      expect(findRgTarget('no.such.key')).toBeNull()
    })

    it('scrollToRgKey 添加高亮 class 并在 1.5s 后移除', () => {
      const el = document.createElement('div')
      el.setAttribute('data-rg-key', 'settings.cloud.main')
      document.body.appendChild(el)
      expect(scrollToRgKey('settings.cloud.main')).toBe(true)
      expect(el.classList.contains('rg-deep-link-highlight')).toBe(true)
      vi.advanceTimersByTime(1500)
      expect(el.classList.contains('rg-deep-link-highlight')).toBe(false)
    })

    it('scrollToRgKey 未命中返回 false 且不抛错', () => {
      expect(scrollToRgKey('no.such.key')).toBe(false)
    })

    it('handleRgDeepLink 根据当前 hash 定位并高亮目标区块', () => {
      const el = document.createElement('div')
      el.setAttribute('data-rg-key', 'nav.projects')
      document.body.appendChild(el)
      window.location.hash = '#rg=nav.projects'
      expect(handleRgDeepLink()).toBe(true)
      expect(el.classList.contains('rg-deep-link-highlight')).toBe(true)
    })
  })
}
