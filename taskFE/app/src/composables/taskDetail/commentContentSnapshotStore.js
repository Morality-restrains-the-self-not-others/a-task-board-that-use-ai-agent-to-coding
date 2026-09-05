import { ref } from 'vue'
import { trimCommentId } from '../../utils/cloudComputeCommentQuery.js'

export function emptyCommentContentSlot() {
  return {
    loading: false,
    message: '',
    messageTraceId: '',
    targetUrl: '',
    text: '',
  }
}

/**
 * 评论级 server-content 快照。两评论并发拉取不得互相覆盖。
 */
export function createCommentContentSnapshotStore() {
  const state = ref({})

  function get(commentId) {
    const cid = trimCommentId(commentId)
    if (!cid) return emptyCommentContentSlot()
    return state.value[cid] || emptyCommentContentSlot()
  }

  function patch(commentId, partial) {
    const cid = trimCommentId(commentId)
    if (!cid) return
    state.value = {
      ...state.value,
      [cid]: { ...emptyCommentContentSlot(), ...state.value[cid], ...partial },
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
      isServerContentLoading: make('loading'),
      serverContentMessage: make('message'),
      serverContentMessageTraceId: make('messageTraceId'),
      serverContentTargetUrl: make('targetUrl'),
      serverContentText: make('text'),
    }
  }

  function clear() {
    state.value = {}
  }

  return { state, get, patch, refsFor, clear }
}
