// @vitest-environment node
/**
 * 公司切换导航 URL：须固定落到目标公司工作面板（非同路径替换）。
 */
if (!process.env.VITEST) {
  console.log('[skip] companySwitchNavigation.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { buildCompanySwitchHref } = await import('./companySwitchNavigation.js')

  describe('buildCompanySwitchHref', () => {
    it('空 companyId → null', () => {
      expect(buildCompanySwitchHref('')).toBeNull()
      expect(buildCompanySwitchHref(null)).toBeNull()
      expect(buildCompanySwitchHref(undefined)).toBeNull()
      expect(buildCompanySwitchHref('   ')).toBeNull()
    })

    it('任意当前路径 → 目标公司 /work-panel/（不保留旧路径）', () => {
      expect(buildCompanySwitchHref('222', 'https://www.daydaymoney.com/tenant/111/people/manage/'))
        .toBe('/tenant/222/work-panel/')
      expect(buildCompanySwitchHref('222', 'https://www.daydaymoney.com/user/999/profile/'))
        .toBe('/tenant/222/work-panel/')
      expect(buildCompanySwitchHref('222', 'https://www.daydaymoney.com/'))
        .toBe('/tenant/222/work-panel/')
      expect(buildCompanySwitchHref('222', 'https://www.daydaymoney.com/pricing/'))
        .toBe('/tenant/222/work-panel/')
    })

    it('任务详情页当前公司 → 仍落到该公司工作面板并保留 accessCode', () => {
      expect(
        buildCompanySwitchHref(
          '877397588196749312',
          'https://www.daydaymoney.com/tenant/877397588196749312/workspace/ws_-2309487803472456748/task-detail/task_878932440129761280/?accessCode=DR2AKvP9J9',
        ),
      ).toBe('/tenant/877397588196749312/work-panel/?accessCode=DR2AKvP9J9')
    })

    it('清除跨租户 workspace_id，不带入目标 URL', () => {
      expect(
        buildCompanySwitchHref(
          '222',
          'https://www.daydaymoney.com/tenant/111/work-panel/?workspace_id=ws-old&foo=1',
        ),
      ).toBe('/tenant/222/work-panel/')
    })

    it('保留 accessCode（与单公司工作面板链接一致）', () => {
      expect(
        buildCompanySwitchHref(
          '850256677331562496',
          'https://www.daydaymoney.com/tenant/111/work-panel/?accessCode=abc123&workspace_id=ws1',
        ),
      ).toBe('/tenant/850256677331562496/work-panel/?accessCode=abc123')
    })

    it('Snowflake 公司 id 原样写入路径', () => {
      expect(buildCompanySwitchHref('660256677331560000')).toBe(
        '/tenant/660256677331560000/work-panel/',
      )
    })
  })
}
