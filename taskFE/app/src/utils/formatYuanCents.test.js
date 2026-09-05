import { describe, expect, it } from 'vitest'
import { formatYuanFromCents } from './formatYuanCents.js'

describe('formatYuanFromCents', () => {
  it('converts cents to yuan with two decimals', () => {
    expect(formatYuanFromCents(55)).toBe('0.55')
    expect(formatYuanFromCents(100)).toBe('1.00')
    expect(formatYuanFromCents(0)).toBe('0.00')
  })

  it('supports 元 suffix', () => {
    expect(formatYuanFromCents(55, { suffix: '元' })).toBe('0.55元')
  })

  it('returns empty placeholder for missing values', () => {
    expect(formatYuanFromCents(null)).toBe('—')
    expect(formatYuanFromCents(undefined)).toBe('—')
    expect(formatYuanFromCents('')).toBe('—')
    expect(formatYuanFromCents('x')).toBe('—')
  })
})
