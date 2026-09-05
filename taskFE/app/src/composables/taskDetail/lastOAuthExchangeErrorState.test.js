// @vitest-environment node
import { describe, it, expect, beforeEach } from 'vitest'
import {
  clearLastOAuthExchangeError,
  isOAuthExchangeFailureMessage,
  lastOAuthExchangeError,
  setLastOAuthExchangeError,
} from './lastOAuthExchangeErrorState.js'

describe('lastOAuthExchangeErrorState', () => {
  beforeEach(() => {
    clearLastOAuthExchangeError()
  })

  it('detects oauth / bridge / refresh failure messages', () => {
    expect(isOAuthExchangeFailureMessage('bridge secret mismatch')).toBe(true)
    expect(isOAuthExchangeFailureMessage('未能换取 access_token')).toBe(true)
    expect(isOAuthExchangeFailureMessage('refresh token expired')).toBe(true)
    expect(isOAuthExchangeFailureMessage('network timeout')).toBe(false)
  })

  it('stores truncated failure summary for linked-projects panel', () => {
    setLastOAuthExchangeError('oauth unauthorized: refresh failed', { maxLen: 20 })
    expect(lastOAuthExchangeError.value).toBe('oauth unauthorized: …')
    clearLastOAuthExchangeError()
    expect(lastOAuthExchangeError.value).toBe('')
  })
})
