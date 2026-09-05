import { describe, expect, it } from 'vitest'
import {
  coerceSnowflakeId,
  normalizeMarketplaceImageList,
  parseJsonPreservingSnowflakeIds,
} from './snowflakeId.js'

describe('snowflakeId', () => {
  it('parseJsonPreservingSnowflakeIds keeps wire-format digits for large ids', () => {
    const raw =
      '[{"id":859670982529273856,"vendor":{"id":859457200064331776},"name":"demo"}]'
    const parsed = parseJsonPreservingSnowflakeIds(raw)
    expect(parsed[0].id).toBe('859670982529273856')
    expect(parsed[0].vendor.id).toBe('859457200064331776')
  })

  it('coerceSnowflakeId returns trimmed string', () => {
    expect(coerceSnowflakeId(' 123 ')).toBe('123')
    expect(coerceSnowflakeId(456)).toBe('456')
  })

  it('normalizeMarketplaceImageList stringifies nested vendor id', () => {
    const rows = normalizeMarketplaceImageList([
      { id: 859670982529273900, vendor: { id: 859457200064331800, company_name: 'v' } },
    ])
    expect(typeof rows[0].id).toBe('string')
    expect(typeof rows[0].vendor.id).toBe('string')
  })
})
