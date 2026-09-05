<template>
  <div v-if="layerChanges" class="mb-2" data-testid="task-detail-layer-changes">
    <details class="rounded border border-gray-200 bg-gray-50">
      <TaskDetailExecLayerChangesSummary
        :display-count="layerChanges.displayCount"
        show-refresh-button
        :refreshing="refreshing"
        :refresh-disabled="refreshDisabled"
        @refresh="$emit('refresh')"
      />
      <p
        v-if="refreshError"
        class="mx-2 mb-0 text-xs text-red-600"
        data-testid="task-detail-layer-changes-refresh-error"
        v-bind="refreshErrorTraceId ? { 'data-traceId': refreshErrorTraceId } : {}"
      >
        {{ refreshError }}
      </p>
      <div class="px-2 pb-2">
        <TaskDetailExecLayerChangesHints
          :detail="layerChanges.detail"
          :truncated="layerChanges.truncated"
          :has-changes="layerChanges.changes.length > 0"
          :has-more="Boolean(layerChanges.has_more)"
          :load-more-busy="Boolean(layerChanges.loadMoreBusy)"
          :truncated-trace-id="String(layerChanges.truncatedTraceId || '')"
        />
        <div
          v-if="layerChanges.changes.length"
          class="mt-2 flex flex-wrap items-end gap-2 border-b border-gray-100 pb-2"
        >
          <div class="flex-1 min-w-[12rem]">
            <label class="text-[11px] text-gray-500 block mb-0.5" for="layer-changes-commit-msg">
              提交说明
              <button
                type="button"
                class="ml-2 text-xs text-blue-600 hover:text-blue-700 hover:bg-blue-50 rounded px-1.5 py-0.5 border border-transparent hover:border-blue-200 disabled:cursor-not-allowed disabled:text-gray-400"
                :disabled="suggestCommitMessageBusy || gitStagedActionsBlocked"
                data-testid="task-detail-layer-changes-suggest-commit-message"
                @click="handleSuggestCommitMessage"
                title="为暂存区文件生成变动说明"
              >
                {{ suggestCommitMessageBusy ? '生成中…' : '生成提交说明' }}
              </button>
            </label>
            <input
              id="layer-changes-commit-msg"
              v-model.trim="stagedCommitMessage"
              type="text"
              class="w-full text-sm border border-gray-300 rounded px-2 py-1"
              placeholder="输入本次提交说明"
              data-testid="task-detail-layer-changes-commit-message"
            >
          </div>
          <button
            type="button"
            class="shrink-0 px-3 py-1.5 text-sm rounded border border-primary text-primary bg-white hover:bg-primary/5 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="commitStagedDisabled"
            data-testid="task-detail-layer-changes-commit-staged"
            @click="emitCommitStaged"
          >
            {{ commitStagedBusy ? '提交中…' : '提交到仓库' }}
          </button>
        </div>
        <ResizableSplitPane
          v-if="layerChanges.changes.length"
          class="mt-1"
          storage-key="task-detail-layer-changes-split"
        >
          <template #left>
            <TaskDetailExecLayerChangesList
              :changes="layerChanges.changes"
              :selected-path="selectedPath"
              :layer-id="String(layerChanges.layer_id || '')"
              :tenant-id="requestContext.tenantId"
              :workspace-id="requestContext.workspaceId"
              :task-id="requestContext.taskId"
              :git-actions-disabled="gitStagedActionsBlocked"
              :container-page-url="containerPageUrl"
              :comment-id="commentId"
              :has-more="Boolean(layerChanges.has_more)"
              :load-more-busy="Boolean(layerChanges.loadMoreBusy)"
              @select="handleSelectPath"
              @preview-loading="handlePreviewLoading"
              @preview-loaded="handlePreviewLoaded"
              @preview-error="handlePreviewError"
              @staged-success="onStagedSuccess"
              @load-more="$emit('load-more')"
            />
          </template>
          <template #right>
            <TaskDetailExecLayerChangePreview
              :selected-path="selectedPath"
              :loading="previewLoading"
              :error="previewError"
              :error-trace-id="previewErrorTraceId"
              :payload="previewPayload"
            />
          </template>
        </ResizableSplitPane>
      </div>
    </details>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../../utils/apiUtils.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { jsonPostWithCommentId } from '../../utils/containerForwardCommentId.js'
import { containerComputeLegacyTaskUrl } from '../../composables/taskDetail/containerComputeRequest.js'
import { resolveTaskRouteIds } from '../../utils/resolveTaskRouteIds.js'
import ResizableSplitPane from '../ResizableSplitPane.vue'
import TaskDetailExecLayerChangesSummary from './TaskDetailExecLayerChangesSummary.vue'
import TaskDetailExecLayerChangesHints from './TaskDetailExecLayerChangesHints.vue'
import TaskDetailExecLayerChangesList from './TaskDetailExecLayerChangesList.vue'
import TaskDetailExecLayerChangePreview from './TaskDetailExecLayerChangePreview.vue'

const props = defineProps({
  layerChanges: {
    type: Object,
    default: null,
  },
  refreshing: { type: Boolean, default: false },
  refreshDisabled: { type: Boolean, default: false },
  refreshError: { type: String, default: '' },
  refreshErrorTraceId: { type: String, default: '' },
  /** 为 true 时禁止「添加」到暂存区、禁止页头「提交到仓库」 */
  gitStagedActionsBlocked: { type: Boolean, default: false },
  /** 为 true 时因未选 Git 身份等禁止仅提交到仓库 */
  gitCommitIdentityBlocked: { type: Boolean, default: false },
  /** 与任务详情 zTree 提交一致，供容器目标解析 */
  containerPageUrl: { type: String, default: '' },
  /** 评论级 CSC；两评论绑不同实例时必须带上 */
  commentId: { type: String, default: '' },
  /** work-panel 弹窗无完整任务路由时由父组件传入 */
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
})

const emit = defineEmits(['refresh', 'commit-staged', 'staged-refresh', 'load-more'])

const route = useRoute()
const selectedPath = ref('')
const previewLoading = ref(false)
const previewError = ref('')
const previewErrorTraceId = ref('')
const previewPayload = ref(null)
const stagedCommitMessage = ref('')
const commitStagedBusy = ref(false)
const suggestCommitMessageBusy = ref(false)

const requestContext = computed(() =>
  resolveTaskRouteIds({
    tenantId: props.tenantId,
    workspaceId: props.workspaceId,
    taskId: props.taskId,
    routeParams: route.params,
  }),
)

const commitStagedDisabled = computed(
  () =>
    commitStagedBusy.value ||
    !stagedCommitMessage.value ||
    props.gitStagedActionsBlocked ||
    props.gitCommitIdentityBlocked
)

watch(
  () => (props.layerChanges ? String(props.layerChanges.layer_id || '') : ''),
  () => {
    selectedPath.value = ''
    previewLoading.value = false
    previewError.value = ''
    previewErrorTraceId.value = ''
    previewPayload.value = null
    stagedCommitMessage.value = ''
  },
  { immediate: true }
)

function onStagedSuccess(payload) {
  const s =
    payload && typeof payload.suggestedCommitMessage === 'string'
      ? payload.suggestedCommitMessage.trim()
      : ''
  if (s) stagedCommitMessage.value = s
  emit('staged-refresh')
}

function emitCommitStaged() {
  if (commitStagedDisabled.value) return
  const message = String(stagedCommitMessage.value || '').trim()
  if (!message) return
  const layerId = String(props.layerChanges?.layer_id || '').trim()
  if (!layerId) return
  commitStagedBusy.value = true
  const finish = () => {
    commitStagedBusy.value = false
  }
  emit('commit-staged', { message, layerId, finish })
}

function handleSelectPath(path) {
  const relPath = String(path || '').trim()
  if (!relPath) return
  selectedPath.value = relPath
}

function handlePreviewLoading(payload) {
  const relPath = String(payload?.path || '').trim()
  if (relPath) selectedPath.value = relPath
  previewError.value = ''
  previewErrorTraceId.value = ''
  previewPayload.value = null
  previewLoading.value = true
}

function handlePreviewLoaded(payload) {
  const relPath = String(payload?.path || '').trim()
  if (relPath) selectedPath.value = relPath
  previewPayload.value = payload?.payload && typeof payload.payload === 'object' ? payload.payload : null
  previewError.value = ''
  previewErrorTraceId.value = ''
  previewLoading.value = false
}

function handlePreviewError(payload) {
  const relPath = String(payload?.path || '').trim()
  if (relPath) selectedPath.value = relPath
  previewError.value = String(payload?.error || '').trim() || '读取文件内容失败'
  previewErrorTraceId.value = String(payload?.traceId || '').trim()
  previewPayload.value = null
  previewLoading.value = false
}

async function handleSuggestCommitMessage() {
  if (suggestCommitMessageBusy.value || props.gitStagedActionsBlocked) return
  const layerId = String(props.layerChanges?.layer_id || '').trim()
  if (!layerId) {
    window.alert('缺少 layer_id，无法生成提交说明')
    return
  }
  const stagedFiles = props.layerChanges?.changes
    ? props.layerChanges.changes
        .filter((c) => c.git_staged)
        .map((c) => String(c.path || '').trim())
        .filter(Boolean)
    : []
  if (stagedFiles.length === 0) {
    window.alert('暂存区没有文件')
    return
  }
  const { tenantId, workspaceId, taskId } = requestContext.value
  if (!tenantId || !workspaceId || !taskId) {
    window.alert('缺少任务上下文，无法生成提交说明')
    return
  }
  suggestCommitMessageBusy.value = true
  try {
    const apiPath = containerComputeLegacyTaskUrl(
      tenantId,
      workspaceId,
      taskId,
      'container-layer-git-diff-log',
      props.commentId,
    )
    const response = await apiFetch(apiPath, jsonPostWithCommentId({ layer_id: layerId, files: stagedFiles }, props.commentId))
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const detail = data.detail
      const msg = typeof detail === 'string' ? detail : '生成提交说明失败'
      showRequestError(msg, data)
      return
    }
    if (data.summary && typeof data.summary === 'string') {
      stagedCommitMessage.value = data.summary.trim()
    } else if (data.log && typeof data.log === 'string') {
      stagedCommitMessage.value = data.log.trim()
    }
  } catch (e) {
    showRequestError(e?.message || '网络错误，生成提交说明失败', e)
  } finally {
    suggestCommitMessageBusy.value = false
  }
}
</script>
