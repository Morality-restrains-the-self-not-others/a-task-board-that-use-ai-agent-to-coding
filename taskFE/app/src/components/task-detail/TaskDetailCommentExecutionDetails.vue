<template>
  <details
    ref="detailsRef"
    class="mt-2 rounded-md border border-gray-100 bg-gray-50/70"
    data-testid="comment-execution-details"
    :data-comment-id="commentId"
    :data-active="isActive ? 'true' : 'false'"
    :data-dependency-mode="displayMode"
    :data-binding-status="bindingStatus || ''"
  >
    <summary
      class="cursor-pointer select-none px-2.5 py-1.5 flex flex-wrap items-center gap-2 text-xs text-gray-700"
      data-testid="comment-execution-details-summary"
    >
      <span class="font-medium text-gray-800">执行细节</span>
      <span
        class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium"
        :class="displayMode === 'independent'
          ? 'bg-emerald-50 text-emerald-800 border border-emerald-100'
          : 'bg-amber-50 text-amber-900 border border-amber-100'"
        data-testid="comment-execution-dependency-badge"
      >
        {{ dependencyBadgeText }}
      </span>
      <span
        v-if="isActive"
        class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-sky-50 text-sky-900 border border-sky-100"
        data-testid="comment-execution-active-badge"
      >当前执行</span>
      <CommentExecutionGitIdentityBadge
        :repo-identities="repoIdentities"
        :git-identity-options="gitIdentityOptions"
        :fallback-repo-identities="fallbackRepoIdentities"
      />
      <CommentExecutionGitOauthBadge
        :repo-identities="repoIdentities"
        :fallback-repo-identities="fallbackRepoIdentities"
        :oauth-readiness="oauthReadiness"
        :last-push-error="lastPushError"
        @retry-oauth-probe="onRetryOauthProbe"
      />
      <span
        v-if="displayContainerName"
        class="inline-flex items-center gap-1 max-w-full min-w-0"
        data-testid="comment-execution-container-name-wrap"
      >
        <span
          class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-indigo-50 text-indigo-900 border border-indigo-100 font-mono max-w-[14rem] truncate"
          :title="displayContainerName"
          data-testid="comment-execution-container-name"
        >{{ displayContainerName }}</span>
        <button
          type="button"
          class="shrink-0 px-1.5 py-0.5 text-[10px] rounded border border-indigo-200 text-indigo-800 bg-white hover:bg-indigo-50"
          :title="copyDone ? '已复制' : '复制容器名'"
          data-testid="comment-execution-container-name-copy"
          @click.stop.prevent="copyContainerName"
        >{{ copyDone ? '已复制' : '复制' }}</button>
      </span>
      <button
        v-if="bindingBadgeText"
        type="button"
        class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium bg-slate-50 text-slate-700 border border-slate-200"
        :class="bindingStatus === 'waiting_previous' ? 'hover:bg-slate-100' : ''"
        data-testid="comment-execution-binding-status"
        :title="bindingStatus === 'waiting_previous' ? '查看前序评论及其执行状态' : ''"
        @click.stop.prevent="onBindingStatusClick"
        @dblclick.stop.prevent="onBindingStatusDblclick"
      >{{ bindingBadgeText }}<template v-if="waitingPreviousCount"> · {{ waitingPreviousCount }}</template></button>
      <CommentExecutionCancelWaitingButton
        :visible="bindingStatus === 'waiting_previous'"
        :busy="cancelWaitingBusy"
        :comment-id="commentId"
        @cancel-waiting="emit('cancel-waiting', { commentId })"
      />
    </summary>
    <div class="px-2.5 pb-2.5 space-y-2 border-t border-gray-100/90">
      <CommentExecutionTabList
        v-if="showTablist"
        :active-tab="activeTab"
        :layer-ztree-tab="layerZtreeTab"
        :server-runtime-status-tab="serverRuntimeStatusTab"
        :show-predecessor-list="showPredecessorList"
        :predecessor-tab-count="predecessorTabCount"
        @pick="pickTab"
      />
      <!-- 容器元信息在 Tab 面板外：切到「服务器运行状态」仍可见容器名 / CSC / 启动 TraceId -->
      <div
        v-if="displayContainerName || cloneProgressRows.length"
        class="pt-2 space-y-1"
        data-testid="comment-execution-container-meta"
        @click.stop
      >
        <div class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-[11px]">
          <span class="text-gray-500 shrink-0">容器名</span>
          <code
            class="font-mono text-gray-800 break-all"
            data-testid="comment-execution-container-name-full"
          >{{ displayContainerName }}</code>
        </div>
        <div
          v-if="displayCscId"
          class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-[11px]"
          data-testid="comment-execution-csc-row"
        >
          <span class="text-gray-500 shrink-0">CSC</span>
          <code
            class="font-mono text-gray-800 break-all"
            data-testid="comment-execution-csc-id"
          >{{ displayCscId }}</code>
        </div>
        <p
          v-else
          class="text-[11px] text-gray-500"
          data-testid="comment-execution-csc-pending"
        >CSC 尚未分配（创建独立实例后对照显示于此）</p>
        <div
          v-if="displayStartTraceId"
          class="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-[11px]"
          data-testid="comment-execution-start-trace-id"
          :data-traceId="displayStartTraceId"
        >
          <span class="text-gray-500 shrink-0">启动 TraceId：</span>
          <code
            class="font-mono text-gray-800 break-all"
            data-testid="comment-execution-start-trace-id-value"
          >{{ displayStartTraceId }}</code>
          <button
            type="button"
            class="shrink-0 px-1.5 py-0.5 text-[10px] rounded border border-gray-200 text-gray-700 bg-white hover:bg-gray-50"
            :title="copyTraceDone ? '已复制' : '复制启动 TraceId'"
            data-testid="comment-execution-start-trace-id-copy"
            @click.stop.prevent="copyStartTraceId"
          >{{ copyTraceDone ? '已复制' : '复制' }}</button>
        </div>
        <TaskDetailCommentCloneProgress
          :rows="cloneProgressRows"
          :reclone-loading-by-url="recloneLoadingByUrl"
          :reclone-status-by-url="recloneStatusByUrl"
          :reclone-error-trace-id-by-url="recloneErrorTraceIdByUrl"
          :repo-reclone-global-loading="repoRecloneGlobalLoading"
          @repo-reclone="onCloneRepoReclone"
        />
      </div>
      <div
        v-if="layerZtreeTab"
        v-show="activeTab === 'ztree'"
        class="pt-2"
        data-testid="comment-execution-panel-ztree"
      >
        <!-- OPT-20260816-036：切走 Tab 时用 v-show 保住 LayerGraphZtree 展开态，
             仅能力开关（layerZtreeTab）控制是否渲染 -->
        <slot name="layer-ztree" />
      </div>
      <div
        v-if="!showTablist || activeTab === 'details'"
        class="pt-2 space-y-2"
        data-testid="comment-execution-panel-details"
      >
      <div
        v-if="commentId && canEditMode"
        class="pt-2 flex flex-wrap items-center gap-2"
        data-testid="comment-execution-mode-controls"
        @click.stop
      >
        <span class="text-[11px] text-gray-500">依赖模式</span>
        <button
          type="button"
          class="px-2 py-0.5 text-[11px] rounded border"
          :class="displayMode === 'wait_previous'
            ? 'bg-amber-50 border-amber-300 text-amber-900'
            : 'bg-white border-gray-200 text-gray-600 hover:bg-gray-50'"
          :disabled="modeSaving"
          data-testid="comment-execution-mode-wait-previous"
          @click.stop.prevent="setMode('wait_previous')"
        >串行</button>
        <button
          type="button"
          class="px-2 py-0.5 text-[11px] rounded border"
          :class="displayMode === 'independent'
            ? 'bg-emerald-50 border-emerald-300 text-emerald-900'
            : 'bg-white border-gray-200 text-gray-600 hover:bg-gray-50'"
          :disabled="modeSaving || queuedAutoRun"
          data-testid="comment-execution-mode-independent"
          @click.stop.prevent="setMode('independent')"
        >可并行</button>
        <span v-if="modeSaving" class="text-[10px] text-gray-400">保存中…</span>
        <span v-if="modeError" class="text-[10px] text-red-600" data-testid="comment-execution-mode-error">{{ modeError }}</span>
        <p
          v-if="queuedAutoRun"
          class="w-full text-[11px] text-gray-500 leading-relaxed"
          data-testid="comment-execution-queue-serial-hint"
        >已加入自动执行队列：本任务轮到后，评论按提交顺序一条一条执行（等前序完成）。</p>
      </div>
      <p
        v-if="!isActive"
        class="pt-1 text-[11px] text-gray-500"
        data-testid="comment-execution-inactive-hint"
      >
        {{ inactiveHint }}
      </p>
      <div :class="isActive ? 'pt-2' : ''">
        <slot />
        <slot v-if="!layerZtreeTab" name="layer-ztree" />
      </div>
      </div>
      <div
        v-if="showPredecessorList && activeTab === 'predecessors'"
        class="pt-2"
        data-testid="comment-execution-panel-predecessors"
      >
        <TaskDetailCommentPredecessorList
          :predecessors="predecessors"
          @focus-predecessor="emit('focus-predecessor', $event)"
        />
      </div>
      <div
        v-if="serverRuntimeStatusTab && activeTab === 'serverRuntime'"
        class="pt-2"
        data-testid="comment-execution-panel-server-runtime"
      >
        <slot name="server-runtime-status" />
      </div>
      <div
        v-if="serverRuntimeStatusTab && activeTab === 'serverContent'"
        class="pt-2"
        data-testid="comment-execution-panel-server-content"
      >
        <slot name="server-content" />
      </div>
    </div>
  </details>
</template>

<script setup>
import { computed, ref, watch, onMounted } from 'vue'
import {
  buildCommentContainerName,
  normalizeCommentContainerName,
} from '../../utils/commentContainerName.js'
import {
  commentExecutionBindingBadgeText,
  commentExecutionInactiveHint,
  isCommentExecutionReleased,
} from '../../composables/taskDetail/useCommentExecutionContext.js'
import TaskDetailCommentCloneProgress from './TaskDetailCommentCloneProgress.vue'
import TaskDetailCommentPredecessorList from './TaskDetailCommentPredecessorList.vue'
import CommentExecutionCancelWaitingButton from './CommentExecutionCancelWaitingButton.vue'
import CommentExecutionTabList from './CommentExecutionTabList.vue'
import CommentExecutionGitIdentityBadge from './CommentExecutionGitIdentityBadge.vue'
import CommentExecutionGitOauthBadge from './CommentExecutionGitOauthBadge.vue'
import { useCommentExecutionClipboard } from '../../composables/taskDetail/useCommentExecutionClipboard.js'

const props = defineProps({
  commentId: { type: String, default: '' },
  dependencyMode: {
    type: String,
    default: 'wait_previous',
    validator: (v) => v === 'wait_previous' || v === 'independent',
  },
  isActive: { type: Boolean, default: false },
  defaultOpen: { type: Boolean, default: false },
  /** 已发出评论的依赖模式只读；仅 composer 提交前可选。默认 false。 */
  canEditMode: { type: Boolean, default: false },
  /**
   * 任务已加入自动执行队列（task.queued_auto_run）。队列逐条执行语义下
   * 已发出评论也不得再切 independent，与 composer hint（OPT-20260827-013）。
   */
  queuedAutoRun: { type: Boolean, default: false },
  modeSaving: { type: Boolean, default: false },
  modeError: { type: String, default: '' },
  /** comment_container_bindings.status */
  bindingStatus: { type: String, default: '' },
  /** 该评论 server-runtime-status 快照（Released / 已释放 覆盖滞后的 running binding） */
  serverRuntimeStatus: { type: String, default: '' },
  /** 评论绑定的容器名（规范：task_* 任务 ID 不二次加前缀） */
  containerName: { type: String, default: '' },
  /**
   * wait_previous 模式下是否存在有效前序（未完成的前序评论）。
   * false 时 badge 显示「串行（无前序）」，避免首条/无前序评论误导为等待前序
   * （OPT-20260811-085）。
   */
  hasEffectivePredecessors: { type: Boolean, default: false },
  /**
   * wait_previous 前序评论（id / summary / status / statusLabel / finished）。
   * waiting_previous 时展示列表，便于查看卡在哪些前序。
   */
  predecessors: { type: Array, default: () => [] },
  /** 评论绑定的 CSC id（与容器名对照） */
  cscId: { type: String, default: '' },
  /** 任务 ID，用于无 binding 时本地推导容器名 */
  taskId: { type: String, default: '' },
  /** 是否挂接任务级共享 CSC（非空 csc_id） */
  ownsSharedContainer: { type: Boolean, default: false },
  /**
   * 本次启动请求的 traceId（start-vm HTTP / SSE status 透传）。
   * 展示在容器名/CSC 旁，便于直接复制到 Grafana/Loki。
   */
  startTraceId: { type: String, default: '' },
  /**
   * 该评论的项目克隆进度行（日志解析或 SSE comment_id 镜像）。
   * 展示在启动 TraceId 下方。
   */
  cloneProgressRows: { type: Array, default: () => [] },
  recloneLoadingByUrl: { type: Object, default: () => ({}) },
  recloneStatusByUrl: { type: Object, default: () => ({}) },
  recloneErrorTraceIdByUrl: { type: Object, default: () => ({}) },
  repoRecloneGlobalLoading: { type: Boolean, default: false },
  /**
   * 启用「执行细节 | 服务器运行状态 | 服务器内容」Tab 形式（后两 Tab
   * 内容由对应插槽提供）。未启用时保持原有单面板展开行为。
   */
  serverRuntimeStatusTab: { type: Boolean, default: false },
  /**
   * 启用「任务关联」Tab（ztree，layer-ztree 插槽）。为 true 时该 Tab 在最左且默认选中。
   */
  layerZtreeTab: { type: Boolean, default: false },
  /** 终止等待前序请求进行中 */
  cancelWaitingBusy: { type: Boolean, default: false },
  /** 该评论发评时选定的仓库 Git 身份（repo_identities） */
  repoIdentities: { type: Array, default: () => [] },
  /** 公司 Git 身份目录，用于把 git_identity_id 解析成标签 */
  gitIdentityOptions: { type: Array, default: () => [] },
  /** 评论 repo_identities 为空时回退的任务级仓库身份（OPT-20260821-028） */
  fallbackRepoIdentities: { type: Array, default: () => [] },
  /** 任务级 Git OAuth 检查结果（关联项目已拉取，摘要行不再发请求） */
  oauthReadiness: { type: Object, default: null },
  /** 该评论层快照 last_push_error；已绑定但仓库拒绝写权限时 overlay OAuth 芯片 */
  lastPushError: { type: String, default: '' },
})

const emit = defineEmits(['update:dependency-mode', 'change-dependency-mode', 'repo-reclone', 'focus-predecessor', 'cancel-waiting', 'retry-oauth-probe'])

/** OPT-20260902-011：check_failed 徽标「重试」向上转发到评论区（持探测 composable 处） */
function onRetryOauthProbe(urls) {
  emit('retry-oauth-probe', urls)
}

function onCloneRepoReclone(payload) {
  const base = payload && typeof payload === 'object' ? { ...payload } : { repoUrl: payload }
  const cid = String(props.commentId || '').trim()
  emit('repo-reclone', cid ? { ...base, commentId: cid, comment_id: cid } : base)
}

const detailsRef = ref(null)
const localMode = ref(props.dependencyMode === 'independent' ? 'independent' : 'wait_previous')

/**
 * 队列已入队（queued_auto_run）时按「逐条执行」语义展示为串行，
 * 即使历史评论存储为 independent 也不显示「可并行」（OPT-20260827-013）。
 * setMode 同时拒绝切回 independent，双保险防打穿。
 */
const displayMode = computed(() =>
  props.queuedAutoRun ? 'wait_previous' : localMode.value,
)
const showPredecessorList = computed(() => {
  if (props.bindingStatus === 'waiting_previous') return true
  return localMode.value === 'wait_previous' && Array.isArray(props.predecessors) && props.predecessors.length > 0
})

/** Tab：ztree=任务关联 / details=执行细节 / predecessors=前序评论 / serverRuntime=运行状态 / serverContent=服务器内容 */
const showTablist = computed(() =>
  props.layerZtreeTab || props.serverRuntimeStatusTab || showPredecessorList.value,
)
const userPickedTab = ref(false)
const activeTab = ref(props.layerZtreeTab ? 'ztree' : 'details')

function pickTab(tab) {
  userPickedTab.value = true
  activeTab.value = tab
}

watch(
  () => props.layerZtreeTab,
  (enabled) => {
    if (enabled) {
      // OPT-20260820-003：bindings 晚到（layerZtreeTab 晚变 true）时，
      // 只要用户还没手动选过 Tab，就自动切到「任务关联」，
      // 避免失败态/可写层提示藏在未选中的「执行细节」。
      if (!userPickedTab.value && activeTab.value !== 'ztree') activeTab.value = 'ztree'
    } else if (activeTab.value === 'ztree') {
      activeTab.value = 'details'
    }
  },
)

watch(showPredecessorList, (enabled) => {
  if (!enabled && activeTab.value === 'predecessors') activeTab.value = 'details'
})

watch(
  () => props.serverRuntimeStatusTab,
  (enabled) => {
    if (!enabled && (activeTab.value === 'serverRuntime' || activeTab.value === 'serverContent')) {
      activeTab.value = 'details'
    }
  },
)

watch(
  () => props.dependencyMode,
  (v) => {
    localMode.value = v === 'independent' ? 'independent' : 'wait_previous'
  },
)

/** 依赖 badge 文案：独立=可并行；串行且有有效前序=等待前序；串行无前序=无前序 */
const dependencyBadgeText = computed(() => {
  if (displayMode.value === 'independent') return '可并行（不等待前序）'
  return props.hasEffectivePredecessors ? '串行（等待前序完成）' : '串行（无前序）'
})

const bindingBadgeText = computed(() =>
  commentExecutionBindingBadgeText(props.bindingStatus, props.serverRuntimeStatus),
)

/** 服务器已释放（云实例 Released/Terminated 或 binding released）时收起执行细节 */
const isReleased = computed(() =>
  isCommentExecutionReleased(props.bindingStatus, props.serverRuntimeStatus),
)

watch(isReleased, (released) => {
  if (!released) return
  const el = detailsRef.value
  if (el) el.open = false
})

const waitingPreviousCount = computed(() => {
  if (props.bindingStatus !== 'waiting_previous') return 0
  return Array.isArray(props.predecessors) ? props.predecessors.length : 0
})

const predecessorTabCount = computed(() =>
  Array.isArray(props.predecessors) ? props.predecessors.length : 0,
)

function onBindingStatusClick() {
  const el = detailsRef.value
  if (!el) return
  if (props.bindingStatus === 'waiting_previous') {
    el.open = true
    if (showPredecessorList.value) pickTab('predecessors')
    return
  }
  el.open = !el.open
}

function onBindingStatusDblclick() {
  const el = detailsRef.value
  if (el) el.open = false
}

/** 展示名：API 优先（并纠双前缀）；否则按规范推导 */
const displayContainerName = computed(() => {
  const fromProp = String(props.containerName || '').trim()
  const tid = String(props.taskId || '').trim()
  const cid = String(props.commentId || '').trim()
  if (fromProp) return normalizeCommentContainerName(tid, cid, fromProp)
  return buildCommentContainerName(tid, cid)
})

const displayCscId = computed(() => String(props.cscId || '').trim())

const displayStartTraceId = computed(() => String(props.startTraceId || '').trim())

const { copyDone, copyTraceDone, copyContainerName, copyStartTraceId } = useCommentExecutionClipboard({
  displayContainerName,
  displayStartTraceId,
})

const inactiveHint = computed(() => commentExecutionInactiveHint({
  bindingStatus: props.bindingStatus,
  ownsSharedContainer: props.ownsSharedContainer,
  dependencyMode: displayMode.value,
  serverRuntimeStatus: props.serverRuntimeStatus,
}))

function setMode(mode) {
  if (mode !== 'wait_previous' && mode !== 'independent') return
  // OPT-20260827-013：队列逐条执行语义下禁止切回 independent（UI 与逻辑双保险）
  if (props.queuedAutoRun && mode === 'independent') return
  if (mode === localMode.value) return
  localMode.value = mode
  emit('update:dependency-mode', mode)
  emit('change-dependency-mode', { commentId: props.commentId, executionMode: mode })
}

function applyOpenPrefer() {
  const el = detailsRef.value
  if (!el) return
  // 服务器已释放时保持折叠，不随 defaultOpen/isActive 自动展开
  if (isReleased.value) return
  if (props.defaultOpen || props.isActive) {
    el.open = true
  }
}

onMounted(applyOpenPrefer)
watch(() => [props.defaultOpen, props.isActive], applyOpenPrefer)
</script>
