import { describe, expect, it } from 'vitest'
import { buildCommentRuntimePanelDisplay } from './buildCommentRuntimePanelDisplay.js'

describe('buildCommentRuntimePanelDisplay', () => {
  it('maps Running slot to action-enabled display text', () => {
    const d = buildCommentRuntimePanelDisplay({
      status: 'Running',
      message: 'ok',
      traceId: 'tr-1',
      response: {
        instance_attribute: { body: { PublicIpAddress: { IpAddress: ['1.2.3.4'] } } },
      },
      loading: false,
    }, { serverUrl: 'http://example.invalid:8080' })
    expect(d.serverRuntimeStatusDisplayText).toBe('运行中')
    expect(d.serverRuntimeStatus).toBe('Running')
    expect(d.showRuntimeActionButtons).toBe(true)
    expect(d.serverRuntimeStatusMessage).toBe('ok')
    expect(d.serverRuntimeStatusTraceId).toBe('tr-1')
    expect(d.serverJumpUrl).toContain('1.2.3.4')
  })

  it('does not leak another comment\'s instance into empty slot', () => {
    const other = buildCommentRuntimePanelDisplay({
      status: 'Running',
      message: 'other',
      response: { instance_id: 'i-other' },
    })
    const empty = buildCommentRuntimePanelDisplay({})
    expect(other.serverRuntimeStatusDisplayText).toBe('运行中')
    expect(empty.serverRuntimeStatusDisplayText).toBe('未知')
    expect(empty.showRuntimeActionButtons).toBe(false)
    expect(empty.serverRuntimeStatusRawJson).toBe('')
  })

  it('maps start-vm marketplace failure message to 启动失败', () => {
    const d = buildCommentRuntimePanelDisplay({
      status: '',
      message: '调用镜像市场 API 失败: 镜像服务返回错误: 502',
    })
    expect(d.serverRuntimeStatusDisplayText).toBe('启动失败')
    expect(d.serverRuntimeStatusMessage).toContain('镜像服务返回错误: 502')
    expect(d.showRuntimeActionButtons).toBe(false)
  })

  it('maps start_failed payload without marketplace snippet to 启动失败', () => {
    const d = buildCommentRuntimePanelDisplay({
      status: '',
      message: '可用区已停售',
      response: { start_failed: true },
    })
    expect(d.serverRuntimeStatusDisplayText).toBe('启动失败')
    expect(d.serverRuntimeStatusMessage).toBe('可用区已停售')
  })

  it('exposes Released raw status for summary badge overlay', () => {
    const d = buildCommentRuntimePanelDisplay({
      status: 'Released',
      message: '服务器已停止并释放',
    })
    expect(d.serverRuntimeStatus).toBe('Released')
    expect(d.serverRuntimeStatusDisplayText).toBe('已释放')
  })
})
