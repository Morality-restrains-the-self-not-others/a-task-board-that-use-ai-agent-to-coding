<template>
  <div
    v-if="visible"
    class="space-y-2"
    data-testid="create-task-repo-oauth-section"
  >
    <div
      v-for="url in visibleOauthRepoUrls"
      :key="url"
      class="rounded-md border border-gray-200 p-2 space-y-1"
      data-testid="create-task-repo-oauth-row"
    >
      <p class="text-xs text-gray-600 break-all">{{ url }}</p>
      <p
        v-if="isRepoOAuthLoading(url)"
        class="text-xs text-gray-500"
        data-testid="create-task-repo-oauth-loading"
      >
        正在检查 Git OAuth…
      </p>
      <p
        v-else-if="isRepoOAuthBound(url)"
        class="text-xs text-emerald-700"
        data-testid="create-task-repo-oauth-bound"
      >
        已绑定 Git OAuth
      </p>
      <div
        v-else
        class="flex items-center gap-2 flex-wrap"
      >
        <a
          class="text-xs px-2 py-1 border border-blue-200 text-blue-700 rounded hover:bg-blue-50"
          data-testid="create-task-repo-oauth-bind"
          :href="repoOAuthStartHref(url)"
        >
          OAuth 绑定
        </a>
        <span class="text-xs text-gray-500">
          开启自动运行前须完成 {{ repoOAuthSiteLabel(url) }} 授权
        </span>
      </div>
      <p
        v-if="repoOAuthError(url)"
        class="text-xs text-red-600"
        role="alert"
        data-testid="create-task-repo-oauth-error"
        :data-traceId="repoOAuthErrorTraceId(url) || undefined"
      >
        {{ repoOAuthError(url) }}
      </p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { collectCreateTaskRepoUrls } from '../utils/createTaskGitIdentityGate.js'
import {
  buildRepoOAuthStartHref,
  resolveRepoOAuthAuthorizeLabel,
  supportsRepoOAuthAuthorize,
} from '../utils/repoOAuthAuthorizeUtils.js'

const props = defineProps({
  editingTask: {
    type: Object,
    required: true,
  },
  projects: {
    type: Array,
    default: () => [],
  },
  oauthBoundByUrl: {
    type: Object,
    default: () => ({}),
  },
  oauthLoadingByUrl: {
    type: Object,
    default: () => ({}),
  },
  oauthErrorByUrl: {
    type: Object,
    default: () => ({}),
  },
  oauthErrorTraceIdByUrl: {
    type: Object,
    default: () => ({}),
  },
  oauthRepoUrls: {
    type: Array,
    default: null,
  },
})

const collectedUrls = computed(() => collectCreateTaskRepoUrls(props.editingTask, props.projects))
const sourceRepoUrls = computed(() =>
  Array.isArray(props.oauthRepoUrls) ? props.oauthRepoUrls : collectedUrls.value,
)
const visibleOauthRepoUrls = computed(() =>
  sourceRepoUrls.value.filter((url) => supportsRepoOAuthAuthorize(url)),
)
const visible = computed(() =>
  Boolean(props.editingTask?.auto_run) && visibleOauthRepoUrls.value.length > 0,
)

const urlKey = (url) => String(url || '').trim()
const isRepoOAuthLoading = (url) => Boolean(props.oauthLoadingByUrl[urlKey(url)])
const isRepoOAuthBound = (url) => props.oauthBoundByUrl[urlKey(url)] === true
const repoOAuthError = (url) => String(props.oauthErrorByUrl[urlKey(url)] || '').trim()
const repoOAuthErrorTraceId = (url) =>
  String(props.oauthErrorTraceIdByUrl[urlKey(url)] || '').trim()
const repoOAuthSiteLabel = (url) => resolveRepoOAuthAuthorizeLabel(url)
const repoOAuthStartHref = (url) => {
  if (typeof window === 'undefined') return buildRepoOAuthStartHref(url)
  return buildRepoOAuthStartHref(url, {
    nextPath: `${window.location.pathname || ''}${window.location.search || ''}`,
  })
}
</script>
