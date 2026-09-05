import { ref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { appendCommentIdPath } from '../../utils/containerForwardCommentId.js'

/**
 * 任务身份区：自动运行步骤缓存 / live 拉取。
 * @param {{
 *   task: import('vue').Ref|object,
 *   tenantId: () => string,
 *   workspaceId: () => string,
 *   taskId: () => string,
 *   commentId: () => string,
 *   containerEndpointRegistered: () => boolean,
 *   containerHeartbeatStatus: () => string,
 * }} opts
 */
export function useTaskIdentityAutoRunSteps(opts) {
  const cachedAutoRunStepsMd = ref('')
  const cachedAutoRunStepsStatus = ref('')
  const liveAutoRunStepsMd = ref(null)
  const liveAutoRunStepsError = ref('')

  const resolveImageId = () => {
    const t = typeof opts.task === 'function' ? opts.task() : opts.task?.value ?? opts.task
    return String(t?.container_image_id || t?.container_image?.id || '').trim()
  }

  const loadCachedAutoRunSteps = async () => {
    cachedAutoRunStepsMd.value = ''
    cachedAutoRunStepsStatus.value = ''
    const t = typeof opts.task === 'function' ? opts.task() : opts.task?.value ?? opts.task
    if (t?.auto_run !== true) return
    const tenant = String(opts.tenantId() || '').trim()
    const imageId = resolveImageId()
    if (!tenant || !imageId) return
    try {
      const resp = await apiFetch(`/api/cloud/installed-images/${encodeURIComponent(imageId)}/tenant_id/${tenant}/`, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!resp.ok) return
      const body = await resp.json().catch(() => ({}))
      cachedAutoRunStepsMd.value = String(body.auto_run_steps_md || '')
      cachedAutoRunStepsStatus.value = String(body.auto_run_steps_extract_status || '')
    } catch {
      /* ignore cache miss */
    }
  }

  const loadLiveAutoRunSteps = async () => {
    liveAutoRunStepsMd.value = null
    liveAutoRunStepsError.value = ''
    const t = typeof opts.task === 'function' ? opts.task() : opts.task?.value ?? opts.task
    if (t?.auto_run !== true || !opts.containerEndpointRegistered()) return
    const hb = String(opts.containerHeartbeatStatus() || '')
    if (hb && hb !== 'connected' && hb !== 'connecting') return
    const tenant = String(opts.tenantId() || '').trim()
    const workspace = String(opts.workspaceId() || '').trim()
    const taskId = String(opts.taskId() || t?.id || '').trim()
    if (!tenant || !workspace || !taskId) return
    const commentId = typeof opts.commentId === 'function' ? opts.commentId() : opts.commentId
    try {
      const path = appendCommentIdPath(
        `/api/cloud/compute/container-auto-run-steps/tenant_id/${tenant}/workspace_id/${workspace}/task_id/${taskId}/`,
        commentId,
      )
      const resp = await apiFetch(path, {
        credentials: 'include',
        headers: { Accept: 'application/json' },
      })
      if (!resp.ok) {
        const j = await resp.json().catch(() => ({}))
        if ((resp.status === 409 || resp.status === 404) && String(cachedAutoRunStepsMd.value || '').trim()) {
          return
        }
        liveAutoRunStepsError.value = String(j.detail || `HTTP ${resp.status}`)
        return
      }
      const body = await resp.json().catch(() => ({}))
      liveAutoRunStepsMd.value = typeof body.markdown === 'string' ? body.markdown : ''
    } catch (e) {
      liveAutoRunStepsError.value = String(e?.message || e)
    }
  }

  watch(
    () => {
      const t = typeof opts.task === 'function' ? opts.task() : opts.task?.value ?? opts.task
      return [t?.auto_run, t?.container_image_id, opts.tenantId()]
    },
    () => { void loadCachedAutoRunSteps() },
    { immediate: true },
  )

  watch(
    () => {
      const t = typeof opts.task === 'function' ? opts.task() : opts.task?.value ?? opts.task
      return [
        t?.auto_run,
        opts.containerEndpointRegistered(),
        opts.containerHeartbeatStatus(),
        opts.tenantId(),
        opts.workspaceId(),
        opts.taskId(),
        typeof opts.commentId === 'function' ? opts.commentId() : opts.commentId,
      ]
    },
    () => { void loadLiveAutoRunSteps() },
    { immediate: true },
  )

  return {
    cachedAutoRunStepsMd,
    cachedAutoRunStepsStatus,
    liveAutoRunStepsMd,
    liveAutoRunStepsError,
  }
}
