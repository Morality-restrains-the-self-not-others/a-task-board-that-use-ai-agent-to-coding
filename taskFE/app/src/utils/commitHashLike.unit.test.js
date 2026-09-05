import { describe, expect, it } from 'vitest'
import { isCommitHashLike } from './commitHashLike.js'

describe('isCommitHashLike', () => {
  it('rejects branch names and short junk', () => {
    expect(isCommitHashLike('main')).toBe(false)
    expect(isCommitHashLike('develop')).toBe(false)
    expect(isCommitHashLike('abc123')).toBe(false)
    expect(isCommitHashLike('')).toBe(false)
    expect(isCommitHashLike(null)).toBe(false)
  })

  it('accepts 7–40 hex', () => {
    expect(isCommitHashLike('abc1234')).toBe(true)
    expect(isCommitHashLike('ABCDEF1')).toBe(true)
    expect(isCommitHashLike('0123456789abcdef0123456789abcdef01234567')).toBe(true)
    expect(isCommitHashLike(' abc1234 ')).toBe(true)
  })

  it('rejects overlong or non-hex', () => {
    expect(isCommitHashLike('0123456789abcdef0123456789abcdef012345678')).toBe(false)
    expect(isCommitHashLike('gbc1234')).toBe(false)
  })
})
