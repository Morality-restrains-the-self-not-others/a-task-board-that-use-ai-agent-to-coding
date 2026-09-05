// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  resolveAutoRunDisabledReasons,
  resolveAutoRunEnabledHint,
  resolveAutoRunGateState,
  resolveAutoRunStartSkippedMessage,
  resolveForceAutoRunHint,
} from './autoRunGateHints.js'

describe('autoRunGateHints', () => {
  it('lists all blockers when project and image missing', () => {
    const state = resolveAutoRunGateState({
      hasLinkedProject: false,
      hasConfiguredRunTemplate: false,
      hasInstalledImage: false,
    })
    expect(state.canEnable).toBe(false)
    expect(state.blockers.map((b) => b.code)).toEqual([
      'AUTO_RUN_PROJECT_REQUIRED',
      'AUTO_RUN_IMAGE_REQUIRED',
    ])
    expect(resolveAutoRunDisabledReasons({
      hasLinkedProject: false,
      hasInstalledImage: false,
    }).length).toBe(2)
  })

  it('requires run template when project linked', () => {
    const state = resolveAutoRunGateState({
      hasLinkedProject: true,
      hasConfiguredRunTemplate: false,
      hasInstalledImage: true,
      projectAllowsAutoRun: true,
    })
    expect(state.canEnable).toBe(false)
    expect(state.blockers[0].code).toBe('AUTO_RUN_RUN_TEMPLATE_REQUIRED')
  })

  it('blocks when project does not allow auto run', () => {
    const state = resolveAutoRunGateState({
      hasLinkedProject: true,
      hasConfiguredRunTemplate: true,
      hasInstalledImage: true,
      projectAllowsAutoRun: false,
    })
    expect(state.canEnable).toBe(false)
    expect(state.blockers[0].code).toBe('AUTO_RUN_PROJECT_NOT_ALLOWED')
    expect(state.blockers[0].message).toContain('是否允许自动运行')
  })

  it('canEnable when all prerequisites met', () => {
    const state = resolveAutoRunGateState({
      hasLinkedProject: true,
      hasConfiguredRunTemplate: true,
      hasInstalledImage: true,
      projectAllowsAutoRun: true,
    })
    expect(state.canEnable).toBe(true)
    expect(state.blockers).toEqual([])
  })

  it('enabled hint mentions server-side cloud gate', () => {
    const hint = resolveAutoRunEnabledHint({ templateSummary: 'aliyun · cn-hangzhou' })
    expect(hint).toContain('aliyun · cn-hangzhou')
    expect(hint).toContain('云启动服务')
  })

  it('force hint explains default no re-start', () => {
    expect(resolveForceAutoRunHint()).toContain('不会二次启动')
  })

  it('start-skipped message includes server reason', () => {
    const msg = resolveAutoRunStartSkippedMessage({
      auto_run_start_skipped: true,
      auto_run_start_skip_reason: '未检测到可用授权',
    })
    expect(msg).toContain('未自动启动服务器')
    expect(msg).toContain('未检测到可用授权')
  })

  it('start-skipped message empty when not skipped', () => {
    expect(resolveAutoRunStartSkippedMessage({ auto_run: true })).toBe('')
  })
})
