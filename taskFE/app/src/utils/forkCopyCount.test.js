import { describe, expect, it } from 'vitest'
import {
  FORK_COPY_COUNT_MAX,
  FORK_COPY_COUNT_MIN,
  clampForkCopyCount,
  forkCopyIdempotencyKey,
} from './forkCopyCount.js'

describe('clampForkCopyCount', () => {
  it('defaults invalid values to 1', () => {
    expect(clampForkCopyCount(undefined)).toBe(FORK_COPY_COUNT_MIN)
    expect(clampForkCopyCount(null)).toBe(FORK_COPY_COUNT_MIN)
    expect(clampForkCopyCount('')).toBe(FORK_COPY_COUNT_MIN)
    expect(clampForkCopyCount('abc')).toBe(FORK_COPY_COUNT_MIN)
    expect(clampForkCopyCount(Number.NaN)).toBe(FORK_COPY_COUNT_MIN)
  })

  it('clamps below min to 1 and above max to 99', () => {
    expect(clampForkCopyCount(0)).toBe(1)
    expect(clampForkCopyCount(-3)).toBe(1)
    expect(clampForkCopyCount(1)).toBe(1)
    expect(clampForkCopyCount(99)).toBe(FORK_COPY_COUNT_MAX)
    expect(clampForkCopyCount(100)).toBe(99)
    expect(clampForkCopyCount('99')).toBe(99)
  })
})

describe('forkCopyIdempotencyKey', () => {
  it('joins batch key with 1-based index', () => {
    expect(forkCopyIdempotencyKey('batch-a', 1)).toBe('batch-a:1')
    expect(forkCopyIdempotencyKey('batch-a', 99)).toBe('batch-a:99')
  })
})
