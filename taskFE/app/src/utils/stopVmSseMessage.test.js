import { describe, expect, it } from 'vitest'
import { isCloudServerStopLogLine, isCloudServerStopSuccessMessage } from './stopVmSseMessage.js'

describe('isCloudServerStopSuccessMessage', () => {
  it('matches real ECS stop success', () => {
    expect(isCloudServerStopSuccessMessage('停止虚拟机成功')).toBe(true)
  })

  it('matches Mock instance stop success', () => {
    expect(isCloudServerStopSuccessMessage('Mock 实例已停止')).toBe(true)
  })

  it('rejects unrelated success copy', () => {
    expect(isCloudServerStopSuccessMessage('aliyun服务器启动成功！')).toBe(false)
    expect(isCloudServerStopSuccessMessage('')).toBe(false)
    expect(isCloudServerStopSuccessMessage(null)).toBe(false)
  })
})

describe('isCloudServerStopLogLine', () => {
  it('matches unlabeled stop-vm progress copy', () => {
    expect(isCloudServerStopLogLine('[18:37:16] 正在准备停止aliyun服务器...')).toBe(true)
    expect(isCloudServerStopLogLine('[18:37:18] 正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）')).toBe(true)
    expect(isCloudServerStopLogLine('正在初始化服务器停止…')).toBe(true)
    expect(isCloudServerStopLogLine('停止虚拟机成功')).toBe(true)
  })

  it('rejects start-vm aliyun copy', () => {
    expect(isCloudServerStopLogLine('[18:41:24] 正在准备启动aliyun服务器...')).toBe(false)
    expect(isCloudServerStopLogLine('aliyun服务器启动成功！')).toBe(false)
  })
})
