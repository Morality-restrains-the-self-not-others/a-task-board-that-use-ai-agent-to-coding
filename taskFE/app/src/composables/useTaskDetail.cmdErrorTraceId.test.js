// @vitest-environment node
/**
 * useTaskDetail 提交层图命令时必须把评论槽 cmdErrorTraceId 交给 submitLayerGraphCommand。
 */
if (!process.env.VITEST) {
  console.log('[skip] useTaskDetail.cmdErrorTraceId.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readUseTaskDetailBundle } = await import('./useTaskDetailBundle.js')

  describe('useTaskDetail layerGraphCmdErrorTraceId wiring', () => {
    it('submitLayerGraphCommand uses the comment-slot traceId ref', () => {
      const src = readUseTaskDetailBundle()
      expect(src).toContain(
        'layerGraphCmdErrorTraceId: slice?.layerGraphCmdErrorTraceId || layerGraphCmdErrorTraceId',
      )
      expect(src).toMatch(/layerGraphCmdError,\s*layerGraphCmdErrorTraceId,\s*layerGraphCmdSending/)
    })

    it('wires zlog.containerReleased into submit and layer-graph deps（OPT-20260823-038）', () => {
      const src = readUseTaskDetailBundle()
      // submitLayerGraphCommand 与 createTaskDetailLayerGraphState 都须拿到 released 标志
      expect(src).toContain('containerReleased: zlog.containerReleased')
      const submitCount = src.split('containerReleased: zlog.containerReleased').length - 1
      expect(submitCount).toBeGreaterThanOrEqual(2)
    })
  })
}
