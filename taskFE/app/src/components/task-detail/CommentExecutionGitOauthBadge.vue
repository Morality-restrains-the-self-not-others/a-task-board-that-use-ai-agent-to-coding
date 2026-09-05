<template>
  <span
    v-if="summary"
    class="inline-flex items-center gap-1 max-w-full min-w-0"
    data-testid="comment-execution-git-oauth-wrap"
  >
    <span
      class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium max-w-[14rem] truncate"
      :class="chipClass"
      :title="summary.title"
      data-testid="comment-execution-git-oauth"
      :data-kind="summary.kind"
      v-bind="summary.traceId ? { 'data-traceId': summary.traceId } : {}"
    >{{ summary.text }}</span>
    <!-- Anti-Replay-OK: real <a href> to OAuth start; stop only prevents summary toggle -->
    <a
      v-if="bindHref"
      class="shrink-0 px-1.5 py-0.5 text-[10px] rounded border bg-white"
      :class="bindLinkClass"
      data-testid="comment-execution-git-oauth-bind"
      :href="bindHref"
      @click.stop
    >{{ summary.bindLabel }}</a>
    <!-- OPT-20260902-011：check_failed 仅记一次，不再自动轮询；由用户点「重试」再探测一次 -->
    <button
      v-if="retryableCheckFailedUrls.length > 0"
      type="button"
      class="shrink-0 px-1.5 py-0.5 text-[10px] rounded border border-amber-300 text-amber-900 bg-white hover:bg-amber-50"
      :title="'重新检查 ' + retryableCheckFailedUrls.join('\n')"
      data-testid="comment-execution-git-oauth-retry"
      @click.stop.prevent="onRetry"
    >重试</button>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import {
  collectCommentOauthRepoUrls,
  commentGitOauthBindHref,
  commentGitOauthSummary,
} from '../../utils/commentExecutionGitOauth.js'

const props = defineProps({
  repoIdentities: { type: Array, default: () => [] },
  fallbackRepoIdentities: { type: Array, default: () => [] },
  oauthReadiness: { type: Object, default: null },
  /** 该评论层快照 git_remote.last_push_error；已绑定但仓库拒绝写权限时 overlay */
  lastPushError: { type: String, default: '' },
})

const emit = defineEmits(['retry-oauth-probe'])

const summary = computed(() =>
  commentGitOauthSummary(
    collectCommentOauthRepoUrls(props.repoIdentities, props.fallbackRepoIdentities),
    props.oauthReadiness,
    props.lastPushError,
  ),
)

const chipClass = computed(() => {
  const kind = summary.value?.kind
  if (kind === 'bound') return 'bg-emerald-50 text-emerald-800 border border-emerald-100'
  if (kind === 'bound_no_write') return 'bg-red-50 text-red-800 border border-red-100'
  if (kind === 'unreachable') return 'bg-red-50 text-red-800 border border-red-100'
  if (kind === 'unbound') return 'bg-amber-50 text-amber-900 border border-amber-100'
  if (kind === 'check_failed') return 'bg-amber-50 text-amber-900 border border-amber-100'
  return 'bg-slate-50 text-slate-700 border border-slate-200'
})

const bindLinkClass = computed(() => {
  if (summary.value?.kind === 'bound_no_write') {
    return 'border-red-200 text-red-900 hover:bg-red-50'
  }
  return 'border-amber-200 text-amber-900 hover:bg-amber-50'
})

const bindHref = computed(() => {
  const kind = summary.value?.kind
  if (kind !== 'unbound' && kind !== 'bound_no_write') return ''
  const repoUrl = String(summary.value.unboundRepoUrls?.[0] || '').trim()
  if (!repoUrl) return ''
  const nextPath = typeof window === 'undefined'
    ? ''
    : `${window.location?.pathname || ''}${window.location?.search || ''}`
  return commentGitOauthBindHref(repoUrl, { nextPath })
})

/** 本评论展示的仓库里真正 check_failed 的 URL；只有它们才值得「重试」（OPT-20260902-011） */
const retryableCheckFailedUrls = computed(() => {
  const failed = new Set(
    (Array.isArray(props.oauthReadiness?.checkFailedRepoUrls)
      ? props.oauthReadiness.checkFailedRepoUrls
      : []
    ).map((url) => String(url || '').trim()).filter(Boolean),
  )
  if (failed.size === 0 || summary.value?.kind !== 'check_failed') return []
  return collectCommentOauthRepoUrls(props.repoIdentities, props.fallbackRepoIdentities)
    .filter((url) => failed.has(url))
})

function onRetry() {
  emit('retry-oauth-probe', retryableCheckFailedUrls.value)
}
</script>
