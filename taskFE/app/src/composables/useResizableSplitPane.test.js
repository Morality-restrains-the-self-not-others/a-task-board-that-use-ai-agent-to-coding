// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  DEFAULT_LEFT_WIDTH_PX,
  MIN_LEFT_WIDTH_PX,
  MAX_LEFT_WIDTH_PX,
  clampLeftWidth,
  leftWidthFromPointer,
  readStoredLeftWidth,
  writeStoredLeftWidth,
} from './useResizableSplitPane.js'

function memoryStorage() {
  const map = new Map()
  return {
    getItem: (k) => (map.has(k) ? map.get(k) : null),
    setItem: (k, v) => {
      map.set(k, String(v))
    },
  }
}

describe('clampLeftWidth', () => {
  it('clamps to min/max', () => {
    expect(clampLeftWidth(10, 1000)).toBe(MIN_LEFT_WIDTH_PX)
    expect(clampLeftWidth(9999, 1000)).toBe(Math.min(MAX_LEFT_WIDTH_PX, Math.floor(1000 * 0.7)))
  })

  it('respects container ratio cap', () => {
    expect(clampLeftWidth(500, 400)).toBe(Math.floor(400 * 0.7))
  })

  it('returns min for non-finite input', () => {
    expect(clampLeftWidth(NaN, 800)).toBe(MIN_LEFT_WIDTH_PX)
  })

  it('rounds to integer', () => {
    expect(clampLeftWidth(256.7, 800)).toBe(257)
  })
})

describe('leftWidthFromPointer', () => {
  it('computes width from clientX relative to container', () => {
    expect(leftWidthFromPointer(300, 100, 800)).toBe(200)
  })
})

describe('storage helpers', () => {
  it('round-trips stored width', () => {
    const s = memoryStorage()
    writeStoredLeftWidth('split-key', DEFAULT_LEFT_WIDTH_PX, s)
    expect(readStoredLeftWidth('split-key', s)).toBe(DEFAULT_LEFT_WIDTH_PX)
  })

  it('returns null for missing or invalid', () => {
    const s = memoryStorage()
    expect(readStoredLeftWidth('', s)).toBeNull()
    expect(readStoredLeftWidth('k', null)).toBeNull()
    s.setItem('k', 'abc')
    expect(readStoredLeftWidth('k', s)).toBeNull()
  })
})
