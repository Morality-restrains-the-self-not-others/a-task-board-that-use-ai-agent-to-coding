import { ref } from 'vue'
import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'

export function emptyCommentRuntimeSlot() {
  return {
    status: '',
    message: '',
    traceId: '',
    response: null,
    loading: false,
  }
}

/**
 * 评论级 server-runtime-status 快照。两评论并发刷新不得互相覆盖。
 */
export function createCommentRuntimeSnapshotStore() {
  const state = ref({})

  function get(commentId) {
    const cid = trimCommentId(commentId)
    if (!cid) return emptyCommentRuntimeSlot()
    return state.value[cid] || emptyCommentRuntimeSlot()
  }

  function patch(commentId, partial) {
    const cid = trimCommentId(commentId)
    if (!cid) return
    state.value = {
      ...state.value,
      [cid]: { ...emptyCommentRuntimeSlot(), ...state.value[cid], ...partial },
    }
  }

  function refsFor(commentId) {
    const cid = trimCommentId(commentId)
    const make = (key) => ({
      get value() {
        return get(cid)[key]
      },
      set value(v) {
        patch(cid, { [key]: v })
      },
    })
    return {
      serverRuntimeStatus: make('status'),
      serverRuntimeStatusMessage: make('message'),
      serverRuntimeStatusTraceId: make('traceId'),
      serverRuntimeStatusResponse: make('response'),
      isServerRuntimeStatusLoading: make('loading'),
    }
  }

  function clear() {
    state.value = {}
  }

  return { state, get, patch, refsFor, clear }
}
