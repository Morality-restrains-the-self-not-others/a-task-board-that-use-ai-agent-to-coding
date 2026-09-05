// @vitest-environment node
import { describe, expect, it } from 'vitest'
import {
  classifyServerStartupError,
  enrichStartupStatusUpdate,
} from './serverStartupErrorDisplay.js'

describe('serverStartupErrorDisplay', () => {
  it('classifies Zone.NotOnSale', () => {
    const d = classifyServerStartupError('指定可用区 cn-hongkong-d 已停售或无可售资源')
    expect(d?.code).toBe('ZONE_NOT_ON_SALE')
    expect(d?.title).toContain('停售')
  })

  it('classifies NoStock without suggesting other zones', () => {
    const d = classifyServerStartupError(
      '启动服务器失败: 可用区 cn-hongkong-b 中实例规格 ecs.hfg7.large 暂无库存，请更换实例规格或地域后重试',
    )
    expect(d?.code).toBe('INSTANCE_NO_STOCK')
    expect(d?.hint).not.toMatch(/其他可用区/)
  })

  it('classifies insufficient balance by code', () => {
    const d = classifyServerStartupError('扣费失败', { code: 'INSUFFICIENT_BALANCE' })
    expect(d?.code).toBe('INSUFFICIENT_BALANCE')
    expect(d?.showRecharge).toBe(true)
  })

  it('classifies container reachability timeout', () => {
    const d = classifyServerStartupError(
      '云主机已运行，但容器服务未在时限内登记可达地址。不是阿里云 API 连不上：实例已 Running，请检查镜像 UserData / 安全组出站 / 容器 HTTP 进程后重试。',
      { code: 'CONTAINER_REACHABILITY_TIMEOUT' },
    )
    expect(d?.code).toBe('CONTAINER_REACHABILITY_TIMEOUT')
    expect(d?.severity).toBe('error')
    expect(d?.title).toContain('容器')
  })

  it('enriches error status updates', () => {
    const out = enrichStartupStatusUpdate({
      status: 'error',
      message: '配额不足',
      progress: 0,
      code: 'INSUFFICIENT_BALANCE',
    })
    expect(out.error_title).toBe('资源配额不足')
    expect(out.error_hint).toContain('购买')
    expect(out.show_recharge).toBe(true)
  })
})
