// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] containerComputeWaiting.test.js requires vitest runtime')
} else {
const { describe, expect, it } = await import('vitest')
const {
  CONTAINER_ENDPOINT_WAITING_HINT,
  LAYER_DIR_PENDING_HINT,
  isContainerComputeWaitingMessage,
  userFacingContainerComputeFailure,
} = await import('./containerComputeWaiting.js')

describe('userFacingContainerComputeFailure', () => {
  it('maps layer not found to a waiting hint, not the raw English detail', () => {
    const out = userFacingContainerComputeFailure('layer not found', 404)
    expect(out.tone).toBe('waiting')
    expect(out.message).toBe(LAYER_DIR_PENDING_HINT)
    expect(out.message).not.toBe('layer not found')
  })

  it('maps 409 missing business address to a waiting hint', () => {
    const out = userFacingContainerComputeFailure(
      '容器尚未注册可用业务地址，请先完成启动与 exchange-refresh',
      409,
    )
    expect(out.tone).toBe('waiting')
    expect(out.message).toBe(CONTAINER_ENDPOINT_WAITING_HINT)
  })

  it('maps the legacy frontend server_url copy to waiting', () => {
    const out = userFacingContainerComputeFailure(
      '容器 server_url 未就绪，无法拉取执行日志',
    )
    expect(out.tone).toBe('waiting')
    expect(out.message).toBe(CONTAINER_ENDPOINT_WAITING_HINT)
    expect(isContainerComputeWaitingMessage(out.message)).toBe(true)
  })

  it('keeps unrelated 404 detail as an error', () => {
    const out = userFacingContainerComputeFailure('not found', 404)
    expect(out.tone).toBe('error')
    expect(out.message).toBe('not found')
  })
})
}
