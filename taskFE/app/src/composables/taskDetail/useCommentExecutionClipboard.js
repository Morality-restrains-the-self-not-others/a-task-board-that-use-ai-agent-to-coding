/**
 * Copy container name / start TraceId from comment execution details summary.
 * Anti-Replay-OK: clipboard write only; no HTTP.
 */
import { ref } from 'vue'
import { writeClipboardText } from '../../utils/writeClipboardText.js'

export function useCommentExecutionClipboard({ displayContainerName, displayStartTraceId }) {
  const copyDone = ref(false)
  const copyTraceDone = ref(false)
  let copyDoneTimer = null
  let copyTraceDoneTimer = null

  async function copyContainerName() {
    const name = String(displayContainerName.value || '').trim()
    if (!name) return
    try {
      await writeClipboardText(name)
      copyDone.value = true
      if (copyDoneTimer) clearTimeout(copyDoneTimer)
      copyDoneTimer = setTimeout(() => { copyDone.value = false }, 1500)
    } catch {
      /* ignore clipboard failures */
    }
  }

  async function copyStartTraceId() {
    const id = String(displayStartTraceId.value || '').trim()
    if (!id) return
    try {
      await writeClipboardText(id)
      copyTraceDone.value = true
      if (copyTraceDoneTimer) clearTimeout(copyTraceDoneTimer)
      copyTraceDoneTimer = setTimeout(() => { copyTraceDone.value = false }, 1500)
    } catch {
      /* ignore clipboard failures */
    }
  }

  return { copyDone, copyTraceDone, copyContainerName, copyStartTraceId }
}
