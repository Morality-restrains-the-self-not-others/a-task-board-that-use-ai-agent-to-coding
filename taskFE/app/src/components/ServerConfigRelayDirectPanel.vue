<template>
  <div v-show="visible" class="p-6 bg-white border border-gray-200 rounded-lg space-y-4">
    <h3 class="text-sm font-medium text-gray-500">直接启动</h3>
    <p class="text-[11px] text-gray-500">
      将环境变量转发至本机 relayToTrae 服务，由其在本地启动 trae-agent/onlineServiceJS（Node）。
    </p>
    <p class="text-[11px] text-gray-500">
      启动前需完成两步：先由任务创建者在创建或编辑任务时完成仓库 OAuth 绑定，再在添加评论（提交并运行）时为每个仓库选择提交身份。
    </p>
    <div
      v-if="effectiveStartBlockedByUnboundOAuth"
      class="rounded border border-amber-200 bg-amber-50 px-2 py-1.5 text-[11px] text-amber-800 leading-snug"
      data-testid="relay-to-trae-oauth-unbound-guide"
    >
      检测到仓库 OAuth 尚未全部完成绑定。请由任务创建者在创建或编辑任务时为全部仓库完成 OAuth 绑定后再启动。
    </div>
    <div
      v-else-if="repoCredentialGuideVisible"
      class="rounded border border-amber-200 bg-amber-50 px-2 py-1.5 text-[11px] text-amber-800 leading-snug"
      data-testid="relay-to-trae-repo-credential-guide"
    >
      检测到仓库克隆凭证不完整。请回到下方添加评论区域，为每个仓库选择本次运行身份后再启动。
    </div>

    <div
      v-if="staleRepoMismatches.length > 0"
      class="rounded border border-orange-200 bg-orange-50 px-2 py-2 text-[11px] text-orange-900 leading-snug space-y-2"
      data-testid="relay-to-trae-stale-repo-mismatch-banner"
    >
      <p class="m-0 font-medium">任务记录的仓库地址与项目当前地址不一致</p>
      <ul class="m-0 pl-4 list-disc space-y-1">
        <li v-for="(row, idx) in staleRepoMismatches" :key="`${row.projectId}-${idx}`">
          <span class="text-gray-600">任务记录：</span>
          <span class="font-mono break-all">{{ row.storedRepoAddress }}</span>
          <span class="mx-1">→</span>
          <span class="text-gray-600">项目当前：</span>
          <span class="font-mono break-all">{{ row.projectRepoUrl }}</span>
        </li>
      </ul>
      <p class="m-0 text-orange-800">
        直接启动将按项目当前地址克隆。请更新任务记录，或确认不更新后再点击「启动」。
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="px-2 py-1 text-[11px] rounded border border-orange-300 bg-white hover:bg-orange-100 disabled:opacity-60"
          data-testid="relay-to-trae-sync-stale-repo-btn"
          :disabled="staleRepoSyncLoading"
          @click="$emit('syncStaleRepoAddresses')"
        >
          {{ staleRepoSyncLoading ? '更新中…' : '更新任务仓库地址' }}
        </button>
        <button
          type="button"
          class="px-2 py-1 text-[11px] rounded border border-gray-300 bg-white hover:bg-gray-50 disabled:opacity-60"
          data-testid="relay-to-trae-ack-stale-repo-btn"
          :disabled="staleRepoSyncLoading || !startBlockedByStaleRepo"
          @click="$emit('acknowledgeStaleRepo')"
        >
          不更新，继续启动
        </button>
      </div>
    </div>

    <div
      class="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px]"
      data-testid="relay-to-trae-status-row"
    >
      <span :class="serviceOnline ? 'text-green-700' : 'text-amber-700'">
        relayToTrae 服务：{{ serviceOnline ? '在线' : '未连接' }}
      </span>
      <span :class="isOnlineServiceUp ? 'text-green-700' : 'text-gray-600'">
        onlineServiceJS：{{ onlineServiceStatusLabel }}
      </span>
      <button
        type="button"
        class="px-2 py-0.5 text-[11px] border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-60"
        data-testid="relay-to-trae-status-refresh"
        :disabled="isStatusLoading"
        @click="$emit('refreshStatus')"
      >
        {{ isStatusLoading ? '查询中…' : '刷新状态' }}
      </button>
    </div>

    <div class="space-y-2">
      <label class="block text-xs text-gray-600">环境变量</label>
      <p class="text-[11px] text-gray-500">
        ACCESS_TOKEN 由服务端在点击「启动」时自动签发（界面不展示明文）。
      </p>
      <div
        v-for="(item, index) in envItems"
        :key="'relay-env-' + item.key + '-' + index"
        class="grid grid-cols-[minmax(200px,1fr)_minmax(260px,2fr)] gap-2 items-center"
      >
        <div class="text-xs font-mono text-gray-700 break-all">{{ item.key }}</div>
        <input
          v-if="item.key === 'ACCESS_TOKEN'"
          :value="accessTokenMaskedLabel"
          type="text"
          readonly
          disabled
          class="w-full px-2 py-1.5 border border-gray-200 rounded-md font-mono text-xs bg-gray-100 text-gray-400 cursor-not-allowed"
          data-testid="relay-to-trae-access-token-masked"
        />
        <input
          v-else
          v-model="item.value"
          type="text"
          class="w-full px-2 py-1.5 border border-gray-300 rounded-md font-mono text-xs focus:outline-none focus:ring-primary focus:border-primary"
          :data-testid="'relay-to-trae-env-' + item.key"
        />
      </div>
    </div>

    <div class="flex items-center gap-3 flex-wrap">
      <button
        type="button"
        class="px-4 py-2 text-sm rounded-md text-white bg-primary hover:opacity-90 disabled:opacity-60"
        data-testid="relay-to-trae-start-btn"
        :disabled="isStartDisabled"
        :title="startDisabledTitle"
        @click="$emit('start')"
      >
        {{ isStarting ? '启动中...' : '启动' }}
      </button>
      <button
        v-if="showStopButton"
        type="button"
        class="px-4 py-2 text-sm rounded-md text-white bg-red-600 hover:opacity-90 disabled:opacity-60"
        data-testid="relay-to-trae-stop-btn"
        :disabled="isStopping"
        @click="$emit('stop')"
      >
        {{ isStopping ? '停止中...' : '停止' }}
      </button>
      <a
        v-if="uiUrl"
        :href="uiUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="text-xs text-primary underline"
        data-testid="relay-to-trae-open-console"
      >
        打开容器页面
      </a>
      <span
        v-else-if="message && String(message).includes('等待 register-reachability')"
        class="text-xs text-amber-700"
        data-testid="relay-to-trae-console-pending"
      >
        等待容器登记地址…
      </span>
      <span v-if="message" class="text-xs text-gray-600">{{ message }}</span>
    </div>
    <p
      v-if="isStartDisabled && startDisabledTitle"
      class="text-xs text-amber-700"
      data-testid="relay-to-trae-start-disabled-hint"
      role="status"
    >
      {{ startDisabledTitle }}
    </p>

    <div class="space-y-2">
      <div class="flex items-center justify-between gap-2">
        <h4 class="text-xs font-medium text-gray-500">启动日志</h4>
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="px-2 py-1 text-[11px] border border-gray-300 rounded-md hover:bg-gray-50"
            data-testid="relay-to-trae-logs-toggle"
            :aria-expanded="logsExpanded ? 'true' : 'false'"
            @click="$emit('toggleLogs')"
          >
            {{ logsExpanded ? '折叠日志' : '展开日志' }}
          </button>
          <button
            type="button"
            class="px-2 py-1 text-[11px] border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-60"
            data-testid="relay-to-trae-logs-clear"
            @click="$emit('clearLogs')"
          >
            清理日志
          </button>
          <button
            type="button"
            class="px-2 py-1 text-[11px] border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-60"
            data-testid="relay-to-trae-logs-copy"
            :disabled="logs.length === 0"
            @click="$emit('copyLogs')"
          >
            {{ logCopyState === 'copied' ? '已复制' : '复制日志' }}
          </button>
        </div>
      </div>
      <div
        v-if="bootstrapStatus.kind !== 'idle'"
        class="flex flex-wrap items-center gap-2 rounded-md border px-2 py-1.5 text-[11px] leading-snug"
        :class="bootstrapStatusBarClass"
        data-testid="relay-to-trae-bootstrap-status"
        :data-bootstrap-status="bootstrapStatus.kind"
      >
        <span
          class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-semibold tracking-wide"
          :class="bootstrapStatusBadgeClass"
          data-testid="relay-to-trae-bootstrap-badge"
        >{{ bootstrapStatusBadgeText }}</span>
        <span class="min-w-0 break-words" data-testid="relay-to-trae-bootstrap-status-label">
          {{ bootstrapStatus.label }}
        </span>
        <span
          v-if="bootstrapStatus.kind === 'failed' && bootstrapStatus.detail"
          class="w-full font-mono text-[10px] opacity-90 break-all"
          data-testid="relay-to-trae-bootstrap-status-detail"
        >{{ bootstrapStatus.detail }}</span>
      </div>
      <div
        v-show="logsExpanded"
        class="p-3 bg-gray-900 text-gray-100 rounded-md overflow-x-auto text-xs min-h-32 space-y-0.5"
        data-testid="relay-to-trae-logs"
      >
        <p v-if="!logs.length" class="m-0 text-gray-400 whitespace-pre-wrap break-words">{{ logsText || '暂无日志' }}</p>
        <p
          v-for="(line, index) in logs"
          :key="index"
          class="m-0 whitespace-pre-wrap break-words"
          :class="bootstrapLogLineClass(line)"
          :data-bootstrap-milestone="bootstrapLogLineKind(line) || undefined"
        >{{ line }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, inject, unref } from 'vue'
import {
  classifyRelayStartupLogLine,
  resolveBootstrapStatusFromLogs,
} from '../utils/relayToTraeUtils.js'

function bootstrapLogLineKind(line) {
  return classifyRelayStartupLogLine(line)
}

function bootstrapLogLineClass(line) {
  const kind = bootstrapLogLineKind(line)
  if (kind === 'complete') return 'text-emerald-300 font-semibold'
  if (kind === 'failed' || kind === 'token_persist_failed') return 'text-red-300 font-semibold'
  if (kind === 'phase') return 'text-sky-300'
  return ''
}

const props = defineProps({
  visible: { type: Boolean, default: false },
  serviceOnline: { type: Boolean, default: false },
  isOnlineServiceUp: { type: Boolean, default: false },
  onlineServiceStatusLabel: { type: String, default: '' },
  isStatusLoading: { type: Boolean, default: false },
  envItems: { type: Array, default: () => [] },
  accessTokenMaskedLabel: { type: String, default: '' },
  isStarting: { type: Boolean, default: false },
  isStopping: { type: Boolean, default: false },
  showStopButton: { type: Boolean, default: false },
  hasTaskId: { type: Boolean, default: false },
  uiUrl: { type: String, default: '' },
  message: { type: String, default: '' },
  repoCredentialGuideVisible: { type: Boolean, default: false },
  staleRepoMismatches: { type: Array, default: () => [] },
  staleRepoSyncLoading: { type: Boolean, default: false },
  startBlockedByStaleRepo: { type: Boolean, default: false },
  startBlockedByUnboundOAuth: { type: Boolean, default: false },
  startBlockedByOAuthCheckLoading: { type: Boolean, default: false },
  hasImage: { type: Boolean, default: false },
  envParamsSourceRequiredHint: { type: String, default: '' },
  logs: { type: Array, default: () => [] },
  logsText: { type: String, default: '' },
  logCopyState: { type: String, default: 'idle' },
  /** 默认折叠，与模拟启动日志区一致；需要排障时再展开 */
  logsExpanded: { type: Boolean, default: false },
})

const injectedStartBlockedByUnboundOAuth = inject('taskDetailRelayStartBlockedByUnboundOAuth', null)
const injectedStartBlockedByOAuthCheckLoading = inject('taskDetailRelayStartBlockedByOAuthCheckLoading', null)

const effectiveStartBlockedByUnboundOAuth = computed(
  () => Boolean(props.startBlockedByUnboundOAuth || unref(injectedStartBlockedByUnboundOAuth)),
)
const effectiveStartBlockedByOAuthCheckLoading = computed(
  () => Boolean(props.startBlockedByOAuthCheckLoading || unref(injectedStartBlockedByOAuthCheckLoading)),
)

defineEmits([
  'refreshStatus',
  'start',
  'stop',
  'toggleLogs',
  'clearLogs',
  'copyLogs',
  'acknowledgeStaleRepo',
  'syncStaleRepoAddresses',
])

const isStartDisabled = computed(
  () =>
    props.isStarting
    || !props.hasTaskId
    || !props.hasImage
    || Boolean(props.envParamsSourceRequiredHint)
    || props.startBlockedByStaleRepo
    || effectiveStartBlockedByUnboundOAuth.value
    || effectiveStartBlockedByOAuthCheckLoading.value,
)

const startDisabledTitle = computed(() => {
  if (!props.hasImage) return '请先在评论区选择镜像'
  if (props.envParamsSourceRequiredHint) return props.envParamsSourceRequiredHint
  if (effectiveStartBlockedByOAuthCheckLoading.value) return '正在检测仓库 OAuth 绑定状态'
  if (effectiveStartBlockedByUnboundOAuth.value) {
    return '请先在创建或编辑任务时绑定 Git OAuth。未绑定仍可发评，私有仓克隆可能失败'
  }
  if (props.startBlockedByStaleRepo) return '请先更新任务仓库地址，或确认不更新后再启动'
  return ''
})

const bootstrapStatus = computed(() => resolveBootstrapStatusFromLogs(props.logs))

const bootstrapStatusBarClass = computed(() => {
  switch (bootstrapStatus.value.kind) {
    case 'failed':
      return 'border-red-300 bg-red-50 text-red-900'
    case 'complete':
      return 'border-emerald-300 bg-emerald-50 text-emerald-900'
    case 'running':
      return 'border-sky-300 bg-sky-50 text-sky-900'
    default:
      return 'border-gray-200 bg-gray-50 text-gray-700'
  }
})

const bootstrapStatusBadgeClass = computed(() => {
  switch (bootstrapStatus.value.kind) {
    case 'failed':
      return 'bg-red-600 text-white'
    case 'complete':
      return 'bg-emerald-600 text-white'
    case 'running':
      return 'bg-sky-600 text-white'
    default:
      return 'bg-gray-500 text-white'
  }
})

const bootstrapStatusBadgeText = computed(() => {
  switch (bootstrapStatus.value.kind) {
    case 'failed':
      return bootstrapStatus.value.phaseKey === 'token_persist' ? '落盘失败' : '引导失败'
    case 'complete':
      return '引导完成'
    case 'running':
      return '引导中'
    default:
      return ''
  }
})
</script>
