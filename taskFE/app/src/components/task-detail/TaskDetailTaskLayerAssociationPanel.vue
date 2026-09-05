<template>
  <div class="mb-4" data-testid="comment-layer-ztree-panel">
    <div class="mb-2 flex flex-wrap items-center gap-2">
      <p class="text-xs font-medium text-gray-600">任务关联（可写层串行 · 容器推送）</p>
      <OpenContainerActionLinks
        v-if="!containerReleased"
        :container-page-url="containerPageUrl"
        :display-container-vscode-url="displayContainerVscodeUrl"
        :tenant-id="tenantId"
        :workspace-id="workspaceId"
        :task-id="taskId"
        :pending-reveal="containerPageLinkPendingReveal"
        :http-unreachable="containerHttpUnreachable"
      />
      <button
        v-if="!containerReleased"
        type="button"
        class="text-xs px-2 py-1 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50"
        :disabled="layerGraphRefreshing"
        @click="emit('refresh-layer-graph')"
      >
        <span v-if="layerGraphRefreshing" class="inline-block animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full mr-1"></span>
        刷新
      </button>
    </div>
    <TaskDetailCommentLayerZtreeErrorBanner
      v-if="commentLayerZtreeLoadingIsError && commentLayerZtreeLoadingHint"
      :hint="commentLayerZtreeLoadingHint"
      :trace-id="commentLayerZtreeLoadingErrorTraceId"
    />
    <LayerGraphZtree
      :flat-nodes="layerGraphZNodes"
      :meta-line="layerGraphMetaLine"
      :selected-id="selectedLayerGraphNode ? selectedLayerGraphNode.id : null"
      :action-busy-key="layerGraphBusyActionKey"
      @node-select="emit('layer-graph-node-select', $event)"
      @job-redo="emit('layer-graph-job-redo', $event)"
      @job-interrupt="emit('layer-graph-job-interrupt', $event)"
      @job-continue="emit('layer-graph-job-continue', $event)"
      @job-edit-run="emit('layer-graph-job-edit-run', $event)"
      @job-delete="emit('layer-graph-job-delete', $event)"
      @layer-delete="emit('layer-graph-layer-delete', $event)"
      @layer-submit="emit('layer-graph-layer-submit', $event)"
      @layer-push="emit('layer-graph-layer-push', $event)"
      @layer-submit-and-push="emit('layer-graph-layer-submit-and-push', $event)"
      @layer-submit-and-merge="emit('layer-graph-layer-submit-and-merge', $event)"
      @layer-merge="emit('layer-graph-layer-merge', $event)"
    />
    <TaskDetailLayerFilesTabs
      v-if="selectedLayerGraphFileTreeLayerId && !containerReleased"
      :layer-id="selectedLayerGraphFileTreeLayerId"
      :changes-count="Number(selectedZTreeLayerChangesPanel?.displayCount || 0)"
    >
      <template #tree>
        <TaskDetailProjectFileTree
          :key="selectedLayerGraphFileTreeLayerId"
          :layer-id="selectedLayerGraphFileTreeLayerId"
          :container-endpoint-registered="containerEndpointRegistered"
          :container-released="containerReleased"
          :file-tree-refresh-nonce="projectFileTreeRefreshNonce"
          :tenant-id="tenantId"
          :workspace-id="workspaceId"
          :task-id="taskId"
          :comment-id="commentId"
          expand-by-default
          data-testid="comment-layer-ztree-project-file-tree"
        />
      </template>
      <template #changes>
        <TaskDetailExecLayerChanges
          v-if="selectedZTreeLayerChangesPanel"
          :layer-changes="selectedZTreeLayerChangesPanel"
          :refreshing="layerChangesRefreshBusy"
          :refresh-disabled="!layerChangesRefreshEnabled"
          :refresh-error="layerChangesRefreshError"
          :refresh-error-trace-id="layerChangesRefreshErrorTraceId"
          :git-staged-actions-blocked="layerChangesGitStagedActionsBlocked"
          :git-commit-identity-blocked="layerChangesGitCommitIdentityBlocked"
          :container-page-url="containerPageUrl"
          :comment-id="commentId"
          @refresh="emit('layer-changes-refresh')"
          @staged-refresh="emit('layer-changes-staged-refresh')"
          @commit-staged="emit('layer-changes-commit-staged')"
          @load-more="emit('layer-changes-load-more')"
        />
        <p
          v-else
          class="text-xs text-gray-400 mt-2 px-2"
          data-testid="layer-files-changes-empty"
        >暂无文件变动数据</p>
      </template>
    </TaskDetailLayerFilesTabs>
    <div
      v-if="selectedLayerGraphNode"
      class="mt-3 rounded-lg border border-gray-200 bg-gray-50/80 p-3"
    >
      <div
        v-if="!containerReleased"
        data-testid="comment-layer-ztree-command-panel"
      >
      <p class="text-xs text-gray-600 mb-2">
        已选节点：<span class="font-medium text-gray-900">{{ selectedLayerGraphNode.name }}</span>
      </p>
      <div class="flex flex-wrap items-center gap-2 mb-2">
        <label class="text-xs text-gray-500" for="layer-graph-cmd-kind">指令类型</label>
        <select
          id="layer-graph-cmd-kind"
          v-model="layerGraphCommandKind"
          class="text-sm border border-gray-300 rounded-md px-2 py-1 bg-white"
        >
          <option value="trae">trae-cli</option>
          <option value="shell">shell</option>
        </select>
        <div class="flex items-center gap-2 ml-auto">
          <label class="text-xs text-gray-500 whitespace-nowrap">模型（单选）</label>
          <details
            ref="modelDetailsRef"
            class="relative"
            :class="layerGraphModelSelectDisabled ? 'pointer-events-none opacity-60' : ''"
          >
            <summary
              class="list-none inline-flex items-center gap-1 text-xs border border-gray-300 rounded-md px-2 py-1 bg-white text-gray-700 cursor-pointer select-none max-w-[14rem]"
            >
              <span>选择模型</span>
              <span class="text-violet-700 truncate" :title="layerGraphSelectedModel || '未选择'">{{
                layerGraphSelectedModel || '未选择'
              }}</span>
              <span class="text-gray-400 shrink-0">▾</span>
            </summary>
            <div class="absolute right-0 z-20 mt-1 w-72 max-h-56 overflow-auto rounded-md border border-gray-200 bg-white p-2 shadow-lg">
              <label class="flex items-center gap-2 py-1 text-xs text-gray-600">
                <input
                  v-model="layerGraphSelectedModel"
                  type="radio"
                  name="layer-graph-agent-model"
                  value=""
                  class="h-3.5 w-3.5 border-gray-300 text-primary focus:ring-primary"
                  :disabled="layerGraphModelSelectDisabled"
                >
                <span>不指定</span>
              </label>
              <label
                v-for="model in layerGraphModelOptions"
                :key="`layer-graph-model-radio-${model}`"
                class="flex items-center gap-2 py-1 text-xs text-gray-700"
              >
                <input
                  v-model="layerGraphSelectedModel"
                  type="radio"
                  name="layer-graph-agent-model"
                  :value="model"
                  class="h-3.5 w-3.5 border-gray-300 text-primary focus:ring-primary"
                  :disabled="layerGraphModelSelectDisabled"
                >
                <span class="truncate">{{ model }}</span>
                <span
                  v-if="layerGraphDefaultModel && layerGraphDefaultModel === model"
                  class="text-[10px] text-violet-600"
                >
                  默认
                </span>
              </label>
              <p v-if="layerGraphModelOptions.length === 0" class="text-xs text-gray-400 py-1">暂无可选模型</p>
            </div>
          </details>
          <label class="text-xs text-gray-500 whitespace-nowrap" for="layer-graph-auto-iteration-count">智能体自动迭代次数</label>
          <input
            id="layer-graph-auto-iteration-count"
            v-model.trim="layerGraphAutoIterationCount"
            type="number"
            min="1"
            step="1"
            class="text-sm border border-gray-300 rounded-md px-2 py-1 bg-white w-40"
            placeholder="如 200"
          >
        </div>
      </div>
      <textarea
        id="layer-graph-command-input"
        ref="layerGraphCommandInputRef"
        v-model="layerGraphCommandText"
        rows="3"
        class="w-full text-sm p-2 border border-gray-300 rounded-md resize-y focus:outline-none focus:ring-primary focus:border-primary"
        placeholder="输入要在该层/任务上执行的指令…"
      />
      <p v-if="layerGraphEditRunTargetJobId" class="mt-2 text-xs text-violet-700">
        当前为「修改指令后执行」模式：将基于任务 {{ layerGraphEditRunTargetJobId }} 派生新任务
      </p>
      <p
        v-if="layerGraphModelLoadError"
        class="mt-2 text-xs text-amber-700"
        v-bind="layerGraphModelLoadErrorTraceId ? { 'data-traceId': layerGraphModelLoadErrorTraceId } : {}"
      >{{ layerGraphModelLoadError }}</p>
      <p
        v-if="layerGraphCmdError"
        class="mt-2 text-xs text-red-600"
        v-bind="layerGraphCmdErrorTraceId ? { 'data-traceId': layerGraphCmdErrorTraceId } : {}"
      >{{ layerGraphCmdError }}</p>
      <div class="mt-2 flex flex-wrap items-center justify-between gap-2">
        <div class="flex flex-wrap items-center gap-1 min-h-[1.25rem]">
          <span
            v-if="layerGraphSelectedModel"
            class="inline-flex items-center rounded-full border border-violet-200 bg-violet-50 px-2 py-0.5 text-[11px] text-violet-700 max-w-full truncate"
            :title="layerGraphSelectedModel"
          >
            {{ layerGraphSelectedModel }}
          </span>
        </div>
        <button
          id="layer-graph-command-send-btn"
          type="button"
          :disabled="layerGraphCmdSending || containerActionsBlocked"
          class="px-4 py-2 text-sm font-medium rounded-md border border-transparent text-white bg-primary hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary disabled:opacity-50 disabled:cursor-not-allowed"
          @click="emit('submit-layer-graph-command')"
        >
          {{ layerGraphCmdSending ? '发送中…' : (layerGraphEditRunTargetJobId ? '修改后执行' : '发送给AI') }}
        </button>
      </div>
      <p v-if="!containerEndpointRegistered" class="mt-2 text-xs text-amber-700">
        容器业务端点尚未就绪时无法发送；请先完成启动并等待「打开容器页面」可用。
      </p>
      <p v-else-if="containerHttpUnreachable" class="mt-2 text-xs text-red-700">
        容器当前无法连接，请待容器恢复后再试。
      </p>
      <p class="mt-2 text-xs text-gray-600">
        层级推送：请在下方添加评论（提交并运行）时为各仓库选择 Git 提交身份；API/自动 PR 请用同页独立的「GitHub App 授权」分区，并在账号中心完成
        <span class="font-medium">Git 网站 OAuth</span> 绑定（平台用 refresh_token 换 access_token，由容器通过 HTTPS 执行
        <code class="text-[11px] bg-gray-100 px-1 rounded">git push</code>）。
      </p>
      </div>
      <p
        v-else
        class="text-xs text-gray-600"
        data-testid="comment-layer-ztree-released-selected-node"
      >
        已选节点：<span class="font-medium text-gray-900">{{ selectedLayerGraphNode.name }}</span>
        <span class="ml-1 text-gray-400">（服务器已释放，步骤来自归档）</span>
      </p>
      <!-- 已释放：隐藏指令/文件树，仍展示 SaaS/COS 归档步骤（ADR-0039） -->
      <div
        class="mt-4 pt-3 border-t border-gray-200"
        data-testid="comment-layer-ztree-exec-log-panel"
      >
        <TaskDetailExecLogPanel
          :loading="layerExecLogLoading"
          :copyable="layerExecLogCopyable"
          :copy-feedback="layerExecLogCopyFeedback"
          :top-error="layerExecLogTopError"
          :targets="zTreeLogTargets"
          :clone-log-fetch-error="layerCloneLogFetchError"
          :clone-log-fetch-error-trace-id="layerCloneLogFetchErrorTraceId"
          :clone-log-text="layerCloneLogText"
          :live-output-display="layerLiveOutputDisplay"
          :job-log-fetch-error="layerJobLogFetchError"
          :job-log-fetch-error-trace-id="layerJobLogFetchErrorTraceId"
          :job-execution-payload="layerJobExecutionPayload"
          :job-command-head="layerJobCommandHead"
          :job-output-display="layerJobOutputDisplay"
          :agent-step-copy-feedback-key="layerAgentStepCopyFeedbackKey"
          :agent-step-cards="layerAgentStepCards"
          @copy-log="emit('copy-layer-exec-log')"
          @clear-log="emit('clear-layer-exec-log')"
          @copy-agent-step-json="(step, stepIdx) => emit('copy-agent-step-json', step, stepIdx)"
          @rich-interact="emit('agent-step-rich-interact', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import LayerGraphZtree from '../LayerGraphZtree.vue'
import {
  TaskDetailExecLayerChanges,
  TaskDetailExecLogPanel,
  TaskDetailProjectFileTree,
} from './index.js'
import TaskDetailLayerFilesTabs from './TaskDetailLayerFilesTabs.vue'
import OpenContainerActionLinks from './OpenContainerActionLinks.vue'
import TaskDetailCommentLayerZtreeErrorBanner from './TaskDetailCommentLayerZtreeErrorBanner.vue'
import { useClickOutside } from '../../composables/useClickOutside.js'

const layerGraphCommandKind = defineModel('layerGraphCommandKind', { type: String, required: true })
const layerGraphSelectedModel = defineModel('layerGraphSelectedModel', { type: String, required: true })
const layerGraphAutoIterationCount = defineModel('layerGraphAutoIterationCount', { type: String, required: true })
const layerGraphCommandText = defineModel('layerGraphCommandText', { type: String, required: true })

const props = defineProps({
  layerGraphRefreshing: { type: Boolean, required: true },
  layerGraphZNodes: { type: Array, required: true },
  layerGraphMetaLine: { type: String, default: '' },
  layerGraphBusyActionKey: { type: String, default: '' },
  selectedLayerGraphNode: { type: Object, default: null },
  selectedZTreeLayerChangesPanel: { type: Object, default: null },
  selectedLayerGraphFileTreeLayerId: { type: String, default: '' },
  layerChangesRefreshBusy: { type: Boolean, required: true },
  layerChangesRefreshEnabled: { type: Boolean, required: true },
  layerChangesRefreshError: { type: String, required: true },
  layerChangesRefreshErrorTraceId: { type: String, default: '' },
  layerChangesGitStagedActionsBlocked: { type: Boolean, required: true },
  layerChangesGitCommitIdentityBlocked: { type: Boolean, required: true },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  commentId: { type: String, default: '' },
  containerPageUrl: { type: String, required: true },
  containerPageLinkPendingReveal: { type: Boolean, required: true },
  containerEndpointRegistered: { type: Boolean, required: true },
  containerHttpUnreachable: { type: Boolean, required: true },
  /** 容器内 code-server（VS Code Web）；有映射 URL 时在任务关联区显示 */
  displayContainerVscodeUrl: { type: String, default: '' },
  projectFileTreeRefreshNonce: { type: Number, required: true },
  layerGraphModelSelectDisabled: { type: Boolean, required: true },
  layerGraphModelOptions: { type: Array, required: true },
  layerGraphDefaultModel: { type: String, default: '' },
  layerGraphEditRunTargetJobId: { type: String, default: '' },
  layerGraphModelLoadError: { type: String, default: '' },
  layerGraphModelLoadErrorTraceId: { type: String, default: '' },
  layerGraphCmdError: { type: String, default: '' },
  layerGraphCmdErrorTraceId: { type: String, default: '' },
  layerGraphCmdSending: { type: Boolean, required: true },
  containerActionsBlocked: { type: Boolean, required: true },
  containerReleased: { type: Boolean, default: false },
  layerExecLogLoading: { type: Boolean, required: true },
  layerExecLogCopyable: { type: Boolean, required: true },
  layerExecLogCopyFeedback: { type: String, default: '' },
  layerExecLogTopError: { type: String, default: '' },
  zTreeLogTargets: { type: Object, required: true },
  layerCloneLogFetchError: { type: String, default: '' },
  layerCloneLogFetchErrorTraceId: { type: String, default: '' },
  layerCloneLogText: { type: String, default: '' },
  layerLiveOutputDisplay: { type: String, default: '' },
  layerJobLogFetchError: { type: String, default: '' },
  layerJobLogFetchErrorTraceId: { type: String, default: '' },
  layerJobExecutionPayload: { type: [Object, null], default: null },
  layerJobCommandHead: { type: String, default: '' },
  layerJobOutputDisplay: { type: String, default: '' },
  layerAgentStepCopyFeedbackKey: { type: String, default: '' },
  layerAgentStepCards: { type: Array, required: true },
  commentLayerZtreeLoadingHint: { type: String, default: '' },
  commentLayerZtreeLoadingIsError: { type: Boolean, default: false },
  commentLayerZtreeLoadingErrorTraceId: { type: String, default: '' },
})

const emit = defineEmits([
  'refresh-layer-graph',
  'layer-graph-node-select',
  'layer-graph-job-redo',
  'layer-graph-job-interrupt',
  'layer-graph-job-continue',
  'layer-graph-job-edit-run',
  'layer-graph-job-delete',
  'layer-graph-layer-delete',
  'layer-graph-layer-submit',
  'layer-graph-layer-push',
  'layer-graph-layer-merge',
  'layer-graph-layer-submit-and-push',
  'layer-graph-layer-submit-and-merge',
  'layer-changes-refresh',
  'layer-changes-staged-refresh',
  'layer-changes-commit-staged',
  'layer-changes-load-more',
  'submit-layer-graph-command',
  'copy-layer-exec-log',
  'clear-layer-exec-log',
  'copy-agent-step-json',
  'agent-step-rich-interact',
])

const layerGraphCommandInputRef = ref(null)
const modelDetailsRef = ref(null)

// 模型下拉：点击模型选择框外部时关闭 <details>；仅在已选中节点时生效（与旧 bind/unbind 语义一致）。
useClickOutside(
  modelDetailsRef,
  () => {
    if (modelDetailsRef.value?.open) modelDetailsRef.value.open = false
  },
  { enabled: () => !!props.selectedLayerGraphNode },
)

defineExpose({
  focusLayerGraphCommandInput: () => layerGraphCommandInputRef.value?.focus?.(),
  selectLayerGraphCommandInput: () => layerGraphCommandInputRef.value?.select?.(),
})
</script>
