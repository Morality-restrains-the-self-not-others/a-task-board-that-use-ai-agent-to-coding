import { describe, expect, it } from 'vitest'
import { appendAssetHashQuery, assetUrl, ASSET_HASH_QUERY_PARAM } from './assetUrl.js'

describe('assetUrl', () => {
  it('appends ?h= when absent', () => {
    expect(appendAssetHashQuery('/utils/foo.js', 'abc123')).toBe('/utils/foo.js?h=abc123')
  })

  it('appends &h= when query already exists', () => {
    expect(appendAssetHashQuery('/a.js?vite=1', 'deadbeef')).toBe('/a.js?vite=1&h=deadbeef')
  })

  it('does not duplicate h param', () => {
    expect(appendAssetHashQuery('/a.js?h=old', 'new')).toBe('/a.js?h=old')
  })

  it('assetUrl normalizes leading slash', () => {
    expect(assetUrl('utils/bar.js', 'ff00')).toBe('/utils/bar.js?h=ff00')
  })

  it('exports param name h', () => {
    expect(ASSET_HASH_QUERY_PARAM).toBe('h')
  })
})
