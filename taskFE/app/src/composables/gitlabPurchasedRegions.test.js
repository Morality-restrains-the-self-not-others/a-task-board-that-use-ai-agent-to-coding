// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] gitlabPurchasedRegions.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { normalizePurchasedGitlabRegions } = await import('./gitlabPurchasedRegions.js')

  describe('normalizePurchasedGitlabRegions', () => {
    it('keeps rows with region + gitlab_web_url and drops pending empty URL', () => {
      const out = normalizePurchasedGitlabRegions([
        {
          region: 'tencent-sh-1',
          region_name: '腾讯上海一区',
          gitlab_web_url: 'https://gitlab-sh1.example',
          provisioning_status: 'active',
        },
        {
          region: 'aliyun-cn-hangzhou',
          region_name: '阿里杭州',
          gitlab_web_url: '',
          provisioning_status: 'pending_admin',
        },
        { region: '', gitlab_web_url: 'https://x.example' },
      ])
      expect(out).toHaveLength(1)
      expect(out[0].region).toBe('tencent-sh-1')
      expect(out[0].gitlab_web_url).toBe('https://gitlab-sh1.example')
    })

    it('returns empty array for non-array input', () => {
      expect(normalizePurchasedGitlabRegions(null)).toEqual([])
      expect(normalizePurchasedGitlabRegions(undefined)).toEqual([])
    })
  })
}
