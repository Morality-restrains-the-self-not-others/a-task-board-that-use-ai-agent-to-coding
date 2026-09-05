<template>
  <div
    v-if="visible"
    class="mb-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950"
    data-testid="auto-run-start-skipped-banner"
    role="status"
  >
    <p class="m-0 font-medium">自动运行已开启，但未自动启动服务器</p>
    <p
      class="m-0 mt-1 text-xs leading-relaxed text-amber-900/90"
      data-testid="auto-run-start-skipped-reason"
    >{{ reasonText }}</p>
    <div class="mt-2 flex flex-wrap items-center gap-2">
      <button
        type="button"
        class="inline-flex items-center rounded-md border border-amber-300 bg-white px-2.5 py-1 text-xs font-medium text-amber-950 hover:bg-amber-100 disabled:opacity-50"
        data-testid="auto-run-force-restart"
        :disabled="busy"
        :aria-busy="busy ? 'true' : 'false'"
        @click="onForceRestart"
      >{{ busy ? '正在重新启动…' : '强制重新启动' }}</button>
      <!-- Anti-Replay-OK: 真实 a[href] 跳转 OAuth 绑定，非副作用按钮 -->
      <a
        v-if="bindHref"
        class="inline-flex items-center rounded-md border border-blue-200 bg-white px-2.5 py-1 text-xs font-medium text-blue-800 hover:bg-blue-50"
        data-testid="auto-run-skip-oauth-bind"
        :href="bindHref"
      >去绑定 Git OAuth</a>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { queryClientPublicIpForAutoSg } from '../../utils/publicClientIp.js'
import { humanizeRequestErrorMessage, showRequestError } from '../../utils/requestErrorDisplay.js'
import { resolveApiErrorMessage } from '../../utils/workPanelFormat.js'
import { alertAutoRunStartSkippedIfNeeded } from '../../utils/autoRunGateHints.js'
import modalService from '../../utils/modalService.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../../utils/clickGuard.js'
import { fetchGitOAuthUserAppConnected } from '../../utils/gitOAuthUserAppConnection.js'
import {
  buildRepoOAuthStartHref,
  shouldShowRepoOAuthAuthorizeButton,
} from '../../utils/repoOAuthAuthorizeUtils.js'

const props = defineProps({
  task: { type: Object, default: null },
  tenantId: { type: String, required: true },
  workspaceId: { type: String, required: true },
  taskId: { type: String, required: true },
  repoUrl: { type: String, default: '' },
})

const emit = defineEmits(['updated'])

const busy = ref(false)
const restartGuard = createClickGuard({ debounceMs: 300 })

const skipReason = computed(() =>
  String(props.task?.auto_run_start_skip_reason || '').trim(),
)

function commentFeedLength(key) {
  const list = props.task?.[key]
  return Array.isArray(list) ? list.length : 0
}

/** 评论独立 feed 已拉取后，auto_run=true 仍无任何评论：视为未写出【自动运行】评论。 */
const legacyEmptyAutoRun = computed(() => {
  if (props.task?.auto_run !== true) return false
  if (props.task?.comments_feeds_loaded !== true) return false
  if (skipReason.value || props.task?.auto_run_start_skipped === true) return false
  return (
    commentFeedLength('comments') === 0
    && commentFeedLength('ai_comments') === 0
    && commentFeedLength('container_agent_comments') === 0
  )
})

const visible = computed(() =>
  props.task?.auto_run === true && (
    props.task?.auto_run_start_skipped === true
    || Boolean(skipReason.value)
    || legacyEmptyAutoRun.value
  ),
)

const persistedReasonText = computed(() => {
  if (skipReason.value) return humanizeRequestErrorMessage(skipReason.value)
  if (legacyEmptyAutoRun.value) {
    return '尚未出现自动运行评论，服务器未自动启动。可检查 Git 网站绑定与网络后强制重新启动。'
  }
  return '因无法获取 Git / 子 Git 仓库列表，已跳过自动启服。请检查 Git 网站绑定与网络后重试。'
})

const wantsOauthBindLink = computed(() =>
  shouldShowRepoOAuthAuthorizeButton(persistedReasonText.value)
  || shouldShowRepoOAuthAuthorizeButton(skipReason.value),
)

const reasonText = computed(() => {
  if (oauthBound.value && wantsOauthBindLink.value) {
    return 'Git 网站已绑定。自动运行仍因先前授权失败未启动，请强制重新启动。'
  }
  return persistedReasonText.value
})

const oauthBound = ref(false)
let oauthCheckSeq = 0

watch(
  () => [visible.value, wantsOauthBindLink.value, String(props.repoUrl || '').trim()],
  async ([isVisible, wantsBind, repoUrl]) => {
    const seq = ++oauthCheckSeq
    if (!isVisible || !wantsBind || !repoUrl) {
      oauthBound.value = false
      return
    }
    try {
      const connected = await fetchGitOAuthUserAppConnected(repoUrl)
      if (seq !== oauthCheckSeq) return
      oauthBound.value = Boolean(connected)
    } catch {
      if (seq !== oauthCheckSeq) return
      oauthBound.value = false
    }
  },
  { immediate: true },
)

const bindHref = computed(() => {
  if (oauthBound.value) return ''
  if (!wantsOauthBindLink.value) return ''
  const repoUrl = String(props.repoUrl || '').trim()
  if (!repoUrl) return ''
  const nextPath = typeof window !== 'undefined'
    ? `${window.location.pathname || ''}${window.location.search || ''}`
    : ''
  return buildRepoOAuthStartHref(repoUrl, nextPath ? { nextPath } : {})
})

async function onForceRestart() {
  const tid = String(props.tenantId || '').trim()
  const wid = String(props.workspaceId || '').trim()
  const taskId = String(props.taskId || '').trim()
  if (!tid || !wid || !taskId) return
  await restartGuard.run(async ({ idempotencyKey }) => {
    if (busy.value) return
    busy.value = true
    try {
      const payload = { auto_run: true, force_auto_run: true }
      const ip = await queryClientPublicIpForAutoSg()
      if (ip) payload.client_public_ip = ip
      const resp = await apiFetch(
        `/api/tasks/todos/tenant_id/${tid}/workspace_id/${wid}/${taskId}/`,
        {
          method: 'PATCH',
          credentials: 'include',
          headers: mergeIdempotencyHeaders(
            { 'Content-Type': 'application/json', Accept: 'application/json' },
            idempotencyKey,
          ),
          body: JSON.stringify(payload),
        },
      )
      const body = await resp.json().catch(() => ({}))
      if (!resp.ok) {
        showRequestError(
          resolveApiErrorMessage(body, { fallback: '强制重新启动失败', httpStatus: resp.status }),
          resp,
        )
        return
      }
      alertAutoRunStartSkippedIfNeeded(body, modalService)
      emit('updated', body)
    } catch (e) {
      showRequestError('强制重新启动失败，请检查网络后重试', e)
    } finally {
      busy.value = false
    }
  })
}
</script>
