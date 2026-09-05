// @vitest-environment jsdom
if (!process.env.VITEST) {
  console.log('[skip] gitlabRegionCapacity.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    bandwidthShareLabel,
    bandwidthUsedMbps,
    remainingBandwidthMbps,
    bandwidthPercent,
  } = await import('./gitlabRegionCapacity.js')

  describe('gitlabRegionCapacity', () => {
    it('共享与独立文案', () => {
      expect(bandwidthShareLabel(true)).toBe('带宽共享分区')
      expect(bandwidthShareLabel(false)).toBe('独立带宽')
    })

    it('总带宽与剩余', () => {
      const r = { total_bandwidth_mbps: 200, remaining_bandwidth_mbps: 80 }
      expect(remainingBandwidthMbps(r)).toBe(80)
      expect(bandwidthUsedMbps(r)).toBe(120)
      expect(bandwidthPercent(r)).toBe(60)
    })

    it('未配置为 0', () => {
      expect(remainingBandwidthMbps({})).toBe(0)
      expect(bandwidthPercent({})).toBe(0)
    })
  })
}
