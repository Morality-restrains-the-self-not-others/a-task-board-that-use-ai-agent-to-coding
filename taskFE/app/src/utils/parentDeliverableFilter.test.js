// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] parentDeliverableFilter.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    parentDeliverableMatchesQuery,
    filterParentDeliverableCandidates,
  } = await import('./parentDeliverableFilter.js')

  const candidate = { id: '850256677331014872', title: '写一个 hello world', workspace_seq: 12 }

  describe('parentDeliverableMatchesQuery', () => {
    it('空查询匹配全部', () => {
      expect(parentDeliverableMatchesQuery(candidate, '')).toBe(true)
      expect(parentDeliverableMatchesQuery(candidate, '  ')).toBe(true)
    })

    it('按任务名称子串匹配', () => {
      expect(parentDeliverableMatchesQuery(candidate, 'hello')).toBe(true)
      expect(parentDeliverableMatchesQuery(candidate, 'HELLO')).toBe(true)
      expect(parentDeliverableMatchesQuery(candidate, '不存在')).toBe(false)
    })

    it('按 workspace_seq 或 #序号匹配', () => {
      expect(parentDeliverableMatchesQuery(candidate, '12')).toBe(true)
      expect(parentDeliverableMatchesQuery(candidate, '#12')).toBe(true)
      expect(parentDeliverableMatchesQuery(candidate, '999999')).toBe(false)
    })

    it('按完整 id 匹配', () => {
      expect(parentDeliverableMatchesQuery(candidate, '850256677331014872')).toBe(true)
    })
  })

  describe('filterParentDeliverableCandidates', () => {
    const list = [
      { id: '850256677331014872', title: '价值流A' },
      { id: '111111111111222222', title: '价值流B' },
    ]

    it('按名称过滤', () => {
      const filtered = filterParentDeliverableCandidates(list, '流B')
      expect(filtered.map((c) => c.id)).toEqual(['111111111111222222'])
    })

    it('已选项即使不匹配查询也保留', () => {
      const filtered = filterParentDeliverableCandidates(list, '流B', '850256677331014872')
      expect(filtered.map((c) => c.id)).toEqual(['850256677331014872', '111111111111222222'])
    })
  })
}
