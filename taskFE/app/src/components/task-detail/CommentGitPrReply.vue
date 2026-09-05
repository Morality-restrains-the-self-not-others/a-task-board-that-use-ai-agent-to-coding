<template>
  <div
    class="mt-2 rounded-md border border-sky-200 bg-sky-50/70 px-2.5 py-2"
    data-testid="comment-git-pr-reply"
  >
    <a
      class="text-xs text-sky-800 hover:underline break-all"
      :href="htmlUrl"
      target="_blank"
      rel="noopener noreferrer"
      data-testid="comment-git-pr-link"
    >{{ htmlUrl }}</a>
    <div class="mt-1.5 flex flex-wrap items-center gap-2">
      <span
        class="text-[10px] font-medium px-1.5 py-0.5 rounded-full"
        :class="badgeClass"
        :title="badgeTitle"
        :data-merged-by="mergedBy || undefined"
        :data-merged-at="mergedAt || undefined"
        data-testid="comment-git-pr-state"
      >{{ stateLabel }}</span>
      <button
        v-if="showMerge"
        type="button"
        class="text-[10px] px-1.5 py-0.5 rounded border border-emerald-300 bg-white text-emerald-800 hover:bg-emerald-50 disabled:opacity-50"
        data-testid="comment-git-pr-merge-btn"
        :disabled="busy"
        :aria-busy="busy ? 'true' : 'false'"
        @click.stop.prevent="onMerge"
      >{{ busy ? '合并中…' : '一键合并' }}</button>
      <!-- Anti-Replay-OK: 真实 a[href] 跳转 OAuth 绑定，非副作用按钮 -->
      <a
        v-if="bindHref"
        class="text-[10px] px-1.5 py-0.5 rounded border border-blue-200 bg-white text-blue-800 hover:bg-blue-50"
        data-testid="comment-git-pr-oauth-bind"
        :href="bindHref"
      >去绑定 Git OAuth</a>
      <span
        v-if="inlineError"
        class="text-[10px] text-rose-700"
        data-testid="comment-git-pr-error"
        v-bind="errorTraceId ? { 'data-traceId': errorTraceId } : {}"
      >{{ inlineError }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { createClickGuard } from '../../utils/clickGuard.js'
import { gitPrHtmlUrlOf, mergeGitPullRequest } from '../../composables/taskDetail/taskDetailGitPrReply.js'
import { humanizeRequestErrorMessage, showRequestError } from '../../utils/requestErrorDisplay.js'
import { extractTraceId } from '../../utils/traceId.js'
import {
  buildRepoOAuthStartHref,
} from '../../utils/repoOAuthAuthorizeUtils.js'
import {
  gitPrOauthStatusErrorText,
  shouldShowGitPrOauthBind,
} from '../../utils/commentExecutionGitOauth.js'

const props = defineProps({
  comment: { type: Object, default: null },
  tenantId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  status: { type: Object, default: null },
  oauthReadiness: { type: Object, default: null },
})

const emit = defineEmits(['merged'])

const htmlUrl = computed(() => gitPrHtmlUrlOf(props.comment))
const state = computed(() => String(props.status?.state || 'unknown').toLowerCase())
/** 合并人展示名（审计回显）：已由谁点击了一键合并 */
const mergedBy = computed(() => String(props.status?.merged_by || '').trim())
/** 合并点击时间（审计回显）：RFC3339，悬停展示本地时间 */
const mergedAt = computed(() => String(props.status?.merged_at || '').trim())
const stateLabel = computed(() => {
  if (state.value === 'merged') {
    return mergedBy.value ? `已由 ${mergedBy.value} 合并` : '已合并'
  }
  if (state.value === 'closed') return '已关闭'
  if (state.value === 'open') return '未合并'
  return '状态未知'
})
const badgeTitle = computed(() => {
  if (state.value !== 'merged') return ''
  const who = mergedBy.value
  const when = formatLocalDateTime(mergedAt.value)
  if (who && when) return `由 ${who} 于 ${when} 合并`
  if (when) return `合并时间：${when}`
  if (who) return `由 ${who} 合并`
  return ''
})

function formatLocalDateTime(value) {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
const badgeClass = computed(() => {
  if (state.value === 'merged') return 'bg-emerald-200/90 text-emerald-900'
  if (state.value === 'closed') return 'bg-gray-200 text-gray-700'
  if (state.value === 'open') return 'bg-amber-200/90 text-amber-900'
  return 'bg-slate-200 text-slate-700'
})
const showMerge = computed(() => Boolean(htmlUrl.value) && state.value === 'open')
const busy = ref(false)
const errorText = ref('')
const errorTrace = ref('')
const displayError = computed(() => {
  const raw = String(errorText.value || props.status?.error || '').trim()
  if (!raw) return ''
  return humanizeRequestErrorMessage(raw)
})
const inlineError = computed(() => gitPrOauthStatusErrorText(displayError.value, props.oauthReadiness))
/** status.error 行展示的 trace：优先本次合并失败响应，其次 status 载荷内带的 trace */
const errorTraceId = computed(() => errorTrace.value || extractTraceId(props.status))
const bindHref = computed(() => {
  if (!shouldShowGitPrOauthBind(displayError.value, props.oauthReadiness)) return ''
  const nextPath = typeof window !== 'undefined'
    ? `${window.location.pathname || ''}${window.location.search || ''}`
    : ''
  return buildRepoOAuthStartHref(htmlUrl.value, nextPath ? { nextPath } : {})
})
const mergeGuard = createClickGuard({ debounceMs: 300 })

async function onMerge() {
  await mergeGuard.run(async ({ headers }) => {
    busy.value = true
    errorText.value = ''
    errorTrace.value = ''
    try {
      const data = await mergeGitPullRequest({
        tenantId: props.tenantId,
        htmlUrl: htmlUrl.value,
        taskId: props.taskId,
        commentId: String(props.comment?.id || ''),
        headers,
      })
      // 服务端合并成功即回显「已由谁合并 + 点击时间」（审计回显，无需等轮询）
      emit('merged', htmlUrl.value, {
        merged_by: data?.merged_by != null ? String(data.merged_by) : '',
        merged_at: data?.merged_at != null ? String(data.merged_at) : '',
      })
    } catch (err) {
      errorText.value = err?.message || '合并失败'
      errorTrace.value = extractTraceId(err)
      showRequestError(errorText.value, err, {
        additionalActions: bindHref.value
          ? [{ text: '去绑定 Git OAuth', href: bindHref.value }]
          : [],
      })
    } finally {
      busy.value = false
    }
  })
}
</script>
