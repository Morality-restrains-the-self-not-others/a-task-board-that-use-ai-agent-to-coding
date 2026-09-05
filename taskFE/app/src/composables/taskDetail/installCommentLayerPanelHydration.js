import { watch } from 'vue'
import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'

/**
 * 每条已挂 CSC 的评论拉一次 ui-context + 层图，写入该评论槽。禁止 setInterval。
 */
export function installCommentLayerPanelHydration(store, deps) {
  watch(
    () => deps.listCscCommentIds(),
    (ids) => {
      const seen = new Set()
      for (const id of ids || []) {
        const cid = trimCommentId(id)
        if (!cid || seen.has(cid)) continue
        seen.add(cid)
        const slot = store.get(cid)
        if (slot.uiContextFetched || slot.hydrateInFlight) continue
        void deps.fetchContainerTaskUiContext(cid)
      }
    },
    { immediate: true },
  )
}
