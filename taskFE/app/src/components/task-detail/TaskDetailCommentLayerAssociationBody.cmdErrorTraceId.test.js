// @vitest-environment node
/**
 * 评论执行细节必须把 cmd 错误 traceId 传到 ztree 面板，DOM 才能带 data-traceId。
 */
if (!process.env.VITEST) {
  console.log('[skip] TaskDetailCommentLayerAssociationBody.cmdErrorTraceId.test.js requires vitest runtime')
} else {
  const { describe, expect, it } = await import('vitest')
  const { readFileSync } = await import('node:fs')
  const { dirname, join } = await import('node:path')
  const { fileURLToPath } = await import('node:url')
  const { buildLayerBodyBind } = await import('../../composables/taskDetail/taskDetailCommentsSectionHelpers.js')

  const here = dirname(fileURLToPath(import.meta.url))

  describe('comment layer association cmd error traceId', () => {
    it('buildLayerBodyBind forwards layerGraphCmdErrorTraceId for the Vue v-bind', () => {
      const bind = buildLayerBodyBind({
        layerGraphCmdError: '没有权限执行该操作，或登录态/容器授权已失效。请刷新页面后重试。',
        layerGraphCmdErrorTraceId: 'tid-cmt-403',
      })
      expect(bind.layerGraphCmdError).not.toBe('HTTP 403')
      expect(bind.layerGraphCmdErrorTraceId).toBe('tid-cmt-403')
    })

    it('TaskDetailCommentsSection declares the prop and v-binds per-comment body', () => {
      const src = readFileSync(join(here, 'TaskDetailCommentsSection.vue'), 'utf8')
      expect(src).toMatch(/layerGraphCmdErrorTraceId:\s*\{\s*type:\s*String/)
      expect(src).toContain('v-bind="buildPerCommentLayerBodyBind(props, comment.id)"')
    })

    it('TaskDetailCommentLayerAssociationBody passes traceId into the ztree panel', () => {
      const src = readFileSync(join(here, 'TaskDetailCommentLayerAssociationBody.vue'), 'utf8')
      expect(src).toContain(':layer-graph-cmd-error-trace-id="layerGraphCmdErrorTraceId"')
      expect(src).toMatch(/layerGraphCmdErrorTraceId:\s*\{\s*type:\s*String/)
    })
  })
}
