// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  isAutoCloneNestedReposEnabled,
  resolveAutoRunGitAuthBlock,
  resolveDefaultAutoRunDisplayLabel,
} from './projectDetailAutoRunGitGate.js'

describe('projectDetailAutoRunGitGate', () => {
  it('treats missing auto_clone_nested_repos as enabled', () => {
    expect(isAutoCloneNestedReposEnabled(undefined)).toBe(true)
    expect(isAutoCloneNestedReposEnabled(true)).toBe(true)
    expect(isAutoCloneNestedReposEnabled(false)).toBe(false)
    expect(isAutoCloneNestedReposEnabled(0)).toBe(false)
    expect(isAutoCloneNestedReposEnabled('false')).toBe(false)
  })

  it('blocks when any repo has token_error (授权异常)', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_available', 'token_error'],
      nestedError: '',
    })
    expect(gate.blocked).toBe(true)
    expect(gate.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')
    expect(gate.displayLabel).toBe('无法启动')
    expect(gate.message).toContain('授权异常')
  })

  it('blocks when nested repos fetch fails after loading', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_available'],
      nestedError: '无法获取子 Git 仓库列表：未检测到可用授权。',
      nestedErrorTraceId: 'tid-nested-gate-1',
      nestedLoading: false,
    })
    expect(gate.blocked).toBe(true)
    expect(gate.code).toBe('AUTO_RUN_NESTED_REPOS_UNAVAILABLE')
    expect(gate.displayLabel).toBe('无法启动')
    expect(gate.traceId).toBe('tid-nested-gate-1')
  })

  it('omits traceId for auth-token aggregation block', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_error'],
      nestedErrorTraceId: 'tid-should-not-leak',
    })
    expect(gate.blocked).toBe(true)
    expect(gate.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')
    expect(gate.traceId).toBe('')
  })

  it('does not block on nestedError while nestedLoading', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: [],
      nestedError: 'stale',
      nestedLoading: true,
    })
    expect(gate.blocked).toBe(false)
  })

  it('does not block for not_bound alone (仅授权异常/子仓失败)', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['not_bound', 'token_available'],
      nestedError: '',
    })
    expect(gate.blocked).toBe(false)
  })

  it('does not block nested token_error when auto clone nested repos is off', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_available'],
      nestedTokenStatuses: ['token_error', 'not_bound'],
      nestedError: '',
      autoCloneNestedRepos: false,
    })
    expect(gate.blocked).toBe(false)
    expect(gate.code).toBe('')
  })

  it('does not block nestedError when auto clone nested repos is off', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_available'],
      nestedError: '无法获取子 Git 仓库列表：未检测到可用授权。',
      nestedErrorTraceId: 'tid-nested-off-1',
      nestedLoading: false,
      autoCloneNestedRepos: false,
    })
    expect(gate.blocked).toBe(false)
    expect(gate.code).toBe('')
    expect(gate.traceId).toBe('')
  })

  it('still blocks parent token_error when auto clone nested repos is off', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_error'],
      nestedTokenStatuses: ['token_available'],
      autoCloneNestedRepos: false,
    })
    expect(gate.blocked).toBe(true)
    expect(gate.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')
  })

  it('blocks nested token_error when auto clone nested repos is on', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_available'],
      nestedTokenStatuses: ['token_error'],
      autoCloneNestedRepos: true,
    })
    expect(gate.blocked).toBe(true)
    expect(gate.code).toBe('AUTO_RUN_GIT_AUTH_ERROR')
    expect(gate.message).toContain('授权异常')
  })

  it('display label is 无法启动 even when default_auto_run is true', () => {
    const gate = resolveAutoRunGitAuthBlock({
      tokenStatuses: ['token_error'],
    })
    expect(
      resolveDefaultAutoRunDisplayLabel({
        enabled: true,
        gitAuthBlock: gate,
      }),
    ).toBe('无法启动')
  })

  it('display label falls back to 已启用/未启用 when not blocked', () => {
    expect(
      resolveDefaultAutoRunDisplayLabel({
        enabled: true,
        gitAuthBlock: { blocked: false },
      }),
    ).toBe('已启用')
    expect(
      resolveDefaultAutoRunDisplayLabel({
        enabled: false,
        gitAuthBlock: null,
      }),
    ).toBe('未启用')
  })
})
