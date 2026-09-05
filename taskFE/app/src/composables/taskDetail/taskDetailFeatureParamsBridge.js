import { computed, shallowRef, unref } from 'vue'

/**
 * 任务详情：ServerConfig 持有智能体资源配置状态，评论 composer 就地渲染该块。
 * 二者为兄弟树，无法 provide/inject，用模块级 ref 桥接（同 image selection bridge）。
 */
export const commentComposerFeatureParams = shallowRef(null)

/**
 * @param {object} bindings refs + handlers from useServerConfigFeatureParams
 * @returns {() => void} stop
 */
export function bindCommentComposerFeatureParams(bindings) {
  commentComposerFeatureParams.value = bindings
  return () => {
    if (commentComposerFeatureParams.value === bindings) {
      commentComposerFeatureParams.value = null
    }
  }
}

/** Composer 模板用的已解包视图（bindings 未登记时为 null）。 */
export function useCommentComposerFeatureParamsView() {
  return computed(() => {
    const b = commentComposerFeatureParams.value
    if (!b) return null
    return {
      tenantId: String(unref(b.tenantId) || ''),
      featureParamsSource: unref(b.featureParamsSource) ?? '',
      personalConfigs: unref(b.personalConfigs) || [],
      selectedPersonalConfigId: unref(b.selectedPersonalConfigId) ?? '',
      resolvedEnvPreview: unref(b.resolvedEnvPreview) || {},
      isEnvPreviewLoading: Boolean(unref(b.isEnvPreviewLoading)),
      envPreviewExpanded: Boolean(unref(b.envPreviewExpanded)),
      persistError: unref(b.featureParamsPersistError) ?? '',
      sourcesAvailable: unref(b.featureParamsSourcesAvailable) !== false,
      onSourceChange: b.onSourceChange,
      fetchEnvPreview: b.fetchEnvPreview,
      onFeatureParamsSourceUpdate: b.onFeatureParamsSourceUpdate,
      onPersonalConfigIdUpdate: b.onPersonalConfigIdUpdate,
      setEnvPreviewExpanded: (v) => {
        if (b.envPreviewExpanded && typeof b.envPreviewExpanded === 'object' && 'value' in b.envPreviewExpanded) {
          b.envPreviewExpanded.value = v
        }
      },
    }
  })
}
