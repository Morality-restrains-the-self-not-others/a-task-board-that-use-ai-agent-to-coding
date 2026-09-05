import { ref, watch } from 'vue'

/**
 * 任务详情：评论区镜像选择 与 ServerConfig.selectedImageId 跨兄弟组件桥接。
 * （ServerConfig 与 CommentsSection 同级，无法用 provide/inject。）
 */

/** @type {import('vue').Ref<string>} */
export const bridgedSelectedImageId = ref('')

/**
 * 将 ServerConfig 的 selectedImageId 与桥接 ref 双向同步（幂等，防环）。
 * @param {import('vue').Ref<string>} selectedImageId
 * @returns {() => void} stop
 */
export function bindServerConfigSelectedImage(selectedImageId) {
  const syncFromPanel = (v) => {
    const next = v != null ? String(v) : ''
    if (bridgedSelectedImageId.value !== next) {
      bridgedSelectedImageId.value = next
    }
  }
  const syncToPanel = (v) => {
    const next = v != null ? String(v) : ''
    if (selectedImageId.value !== next) {
      selectedImageId.value = next
    }
  }
  syncFromPanel(selectedImageId.value)
  const stopA = watch(selectedImageId, syncFromPanel, { flush: 'sync' })
  const stopB = watch(bridgedSelectedImageId, syncToPanel, { flush: 'sync' })
  return () => {
    stopA()
    stopB()
  }
}
