import { describe, expect, it } from 'vitest'
import {
  mapBindingLifecycleToServerStatus,
  mapBindingStatusToLifecycle,
} from './bindingLifecycleMaps.js'

describe('mapBindingLifecycleToServerStatus', () => {
  it('keeps live start/run codes', () => {
    expect(mapBindingLifecycleToServerStatus('已启动', true, false)).toBe('success')
    expect(mapBindingLifecycleToServerStatus('启动中', false, true)).toBe('processing')
    expect(mapBindingLifecycleToServerStatus('启动失败', false, false)).toBe('error')
  })

  it('maps released/completed/cancelled so the start-log panel stays visible after refresh', () => {
    expect(mapBindingLifecycleToServerStatus(mapBindingStatusToLifecycle('released'), false, false)).toBe('stopped')
    expect(mapBindingLifecycleToServerStatus(mapBindingStatusToLifecycle('completed'), false, false)).toBe('stopped')
    expect(mapBindingLifecycleToServerStatus(mapBindingStatusToLifecycle('cancelled'), false, false)).toBe('stopped')
  })
})
