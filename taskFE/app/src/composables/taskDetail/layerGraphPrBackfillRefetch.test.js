// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] layerGraphPrBackfillRefetch.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const {
    collectLayerPrHtmlUrls,
    missingLayerPrHtmlUrls,
    shouldRefetchCommentsForLayerPrBackfill,
  } = await import('./layerGraphPrBackfillRefetch.js')

  describe('collectLayerPrHtmlUrls', () => {
    it('collects git_remote.pr_html_url and pr.html_url and dedups', () => {
      const urls = collectLayerPrHtmlUrls([
        { layer_id: 'L1', git_remote: { pr_html_url: 'https://github.com/a/b/pull/1' } },
        { layer_id: 'L2', pr: { html_url: 'https://github.com/a/b/pull/2' } },
        { layer_id: 'L3', git_remote: { pr_html_url: 'https://github.com/a/b/pull/1' } },
      ])
      expect(urls).toEqual([
        'https://github.com/a/b/pull/1',
        'https://github.com/a/b/pull/2',
      ])
    })

    it('defensively walks children and nested layers', () => {
      const urls = collectLayerPrHtmlUrls([
        {
          layer_id: 'root',
          children: [{ layer_id: 'c1', git_remote: { pr_html_url: 'https://gitlab.example/g/r/-/merge_requests/7' } }],
        },
        { layers: [{ layer_id: 'n1', pr: { html_url: 'https://gitlab.example/g/r2/-/merge_requests/8' } }] },
      ])
      expect(urls).toContain('https://gitlab.example/g/r/-/merge_requests/7')
      expect(urls).toContain('https://gitlab.example/g/r2/-/merge_requests/8')
    })

    it('ignores empty / non-http values', () => {
      expect(collectLayerPrHtmlUrls([
        { layer_id: 'L1', git_remote: { pr_html_url: '' } },
        { layer_id: 'L2', git_remote: { ahead: 1 } },
      ])).toEqual([])
      expect(collectLayerPrHtmlUrls(null)).toEqual([])
    })
  })

  describe('missingLayerPrHtmlUrls / shouldRefetchCommentsForLayerPrBackfill', () => {
    const layers = [
      { layer_id: 'L1', git_remote: { pr_html_url: 'https://github.com/a/b/pull/9' } },
      { layer_id: 'L2', git_remote: { pr_html_url: 'https://github.com/a/b/pull/10' } },
    ]

    it('flags PR URLs absent from the current comment feed', () => {
      expect(missingLayerPrHtmlUrls(layers, ['https://github.com/a/b/pull/9'])).toEqual([
        'https://github.com/a/b/pull/10',
      ])
      expect(shouldRefetchCommentsForLayerPrBackfill(layers, ['https://github.com/a/b/pull/9'])).toBe(true)
    })

    it('returns false when the feed already has every layer PR URL', () => {
      const feed = ['https://github.com/a/b/pull/9', 'https://github.com/a/b/pull/10']
      expect(missingLayerPrHtmlUrls(layers, feed)).toEqual([])
      expect(shouldRefetchCommentsForLayerPrBackfill(layers, feed)).toBe(false)
    })

    it('returns false when there are no layer PR URLs', () => {
      expect(shouldRefetchCommentsForLayerPrBackfill([], [])).toBe(false)
      expect(shouldRefetchCommentsForLayerPrBackfill(null, [])).toBe(false)
    })

    it('normalizes scheme/host casing and trailing whitespace', () => {
      const feed = ['HTTPS://GitHub.com/a/b/pull/9 ']
      expect(missingLayerPrHtmlUrls(layers, feed)).toEqual([
        'https://github.com/a/b/pull/10',
      ])
    })
  })
}
