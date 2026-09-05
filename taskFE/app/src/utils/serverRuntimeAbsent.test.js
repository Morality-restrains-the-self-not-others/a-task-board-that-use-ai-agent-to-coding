// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  isServerRuntimeAbsentPayload,
  isServerRuntimeNoInstanceMessage,
  isServerRuntimeStartFailedMessage,
} from './serverRuntimeAbsent.js'

describe('isServerRuntimeNoInstanceMessage', () => {
  it('识别尚未创建 / 未找到配置', () => {
    expect(isServerRuntimeNoInstanceMessage('该任务尚未创建云实例')).toBe(true)
    expect(isServerRuntimeNoInstanceMessage('未找到服务器配置记录')).toBe(true)
    expect(isServerRuntimeNoInstanceMessage('启动失败且未创建评论级服务器配置。请用启动 TraceId 排查后重试。')).toBe(true)
    expect(isServerRuntimeNoInstanceMessage('启动已发起，但尚未创建评论级服务器配置。请用启动 TraceId 排查。')).toBe(true)
    expect(isServerRuntimeNoInstanceMessage('调用镜像市场 API 失败: 镜像服务返回错误: 502')).toBe(true)
    expect(isServerRuntimeNoInstanceMessage('查询失败')).toBe(false)
    expect(isServerRuntimeNoInstanceMessage('')).toBe(false)
  })

  it('启动中创建文案不算「无实例」', () => {
    expect(isServerRuntimeNoInstanceMessage('云实例创建中，等待分配')).toBe(false)
  })
})

describe('isServerRuntimeStartFailedMessage', () => {
  it('识别启机失败文案', () => {
    expect(isServerRuntimeStartFailedMessage('启动失败且未创建评论级服务器配置。请用启动 TraceId 排查后重试。')).toBe(true)
    expect(isServerRuntimeStartFailedMessage('调用镜像市场 API 失败: 镜像服务返回错误: 502')).toBe(true)
    expect(isServerRuntimeStartFailedMessage('启动已发起，但尚未创建评论级服务器配置。请用启动 TraceId 排查。')).toBe(false)
  })

  it('识别 start_failed 契约字段', () => {
    expect(isServerRuntimeStartFailedMessage('可用区已停售', { start_failed: true })).toBe(true)
    expect(isServerRuntimeAbsentPayload({
      status: 'success',
      runtime_status: null,
      message: '可用区已停售',
      start_failed: true,
    })).toBe(true)
  })
})

describe('isServerRuntimeAbsentPayload', () => {
  it('success + 空 runtime + 无实例文案 → true', () => {
    expect(
      isServerRuntimeAbsentPayload({
        status: 'success',
        runtime_status: null,
        message: '该任务尚未创建云实例',
        instance_id: null,
      }),
    ).toBe(true)
  })

  it('有 runtime_status 或 instance_id → false', () => {
    expect(
      isServerRuntimeAbsentPayload({
        status: 'success',
        runtime_status: 'Running',
        message: '该任务尚未创建云实例',
      }),
    ).toBe(false)
    expect(
      isServerRuntimeAbsentPayload({
        status: 'success',
        runtime_status: null,
        instance_id: 'i-1',
        message: '该任务尚未创建云实例',
      }),
    ).toBe(false)
  })

  it('启动中 Starting 载荷 → false（不得回落已停止）', () => {
    expect(
      isServerRuntimeAbsentPayload({
        status: 'success',
        runtime_status: 'Starting',
        message: '云实例创建中，等待分配',
        instance_id: null,
      }),
    ).toBe(false)
  })
})
