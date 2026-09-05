// @vitest-environment node
if (!process.env.VITEST) {
  console.log('[skip] taskDetailFeatureParamsBridge.test.js requires vitest runtime')
} else {
  const { describe, it, expect } = await import('vitest')
  const { ref } = await import('vue')
  const {
    commentComposerFeatureParams,
    bindCommentComposerFeatureParams,
    useCommentComposerFeatureParamsView,
  } = await import('./taskDetailFeatureParamsBridge.js')

  describe('taskDetailFeatureParamsBridge', () => {
    it('登记后 composer 视图可读来源；卸载后清空', () => {
      commentComposerFeatureParams.value = null
      const featureParamsSource = ref('company')
      const stop = bindCommentComposerFeatureParams({
        tenantId: ref('t1'),
        featureParamsSource,
        personalConfigs: ref([]),
        selectedPersonalConfigId: ref(''),
        resolvedEnvPreview: ref({}),
        isEnvPreviewLoading: ref(false),
        envPreviewExpanded: ref(false),
        featureParamsPersistError: ref(''),
        featureParamsSourcesAvailable: ref(true),
        onSourceChange: () => {},
        fetchEnvPreview: () => {},
        onFeatureParamsSourceUpdate: () => {},
        onPersonalConfigIdUpdate: () => {},
      })
      const view = useCommentComposerFeatureParamsView()
      expect(view.value.tenantId).toBe('t1')
      expect(view.value.featureParamsSource).toBe('company')
      featureParamsSource.value = 'workspace'
      expect(view.value.featureParamsSource).toBe('workspace')
      stop()
      expect(commentComposerFeatureParams.value).toBeNull()
      expect(view.value).toBeNull()
    })
  })
}
