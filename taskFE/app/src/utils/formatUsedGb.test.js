import { describe, expect, it } from 'vitest'
import { formatUsedGb, usagePercent } from './formatUsedGb.js'

describe('formatUsedGb', () => {
  it('treats non-finite and non-positive as 0', () => {
    expect(formatUsedGb(0)).toBe('0')
    expect(formatUsedGb(-1)).toBe('0')
    expect(formatUsedGb(Number.NaN)).toBe('0')
    expect(formatUsedGb(undefined)).toBe('0')
  })

  it('keeps integers as-is', () => {
    expect(formatUsedGb(1)).toBe('1')
    expect(formatUsedGb(12)).toBe('12')
  })

  it('keeps up to 6 decimal places and strips trailing zeros', () => {
    expect(formatUsedGb(0.25)).toBe('0.25')
    expect(formatUsedGb(0.100000)).toBe('0.1')
    expect(formatUsedGb(0.1234567)).toBe('0.123457')
  })
})

describe('usagePercent', () => {
  it('returns 0 when quota is missing or used is empty', () => {
    expect(usagePercent(1, 0)).toBe(0)
    expect(usagePercent(0, 10)).toBe(0)
  })

  it('clamps used over quota to 100', () => {
    expect(usagePercent(5, 4)).toBe(100)
    expect(usagePercent(1, 4)).toBe(25)
  })
})
