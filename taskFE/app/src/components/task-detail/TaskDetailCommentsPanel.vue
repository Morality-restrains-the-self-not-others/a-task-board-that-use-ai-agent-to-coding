<template>
  <div class="p-3 bg-white border border-gray-200 rounded-lg">
    <h3 class="text-sm font-medium text-gray-500 mb-2">评论</h3>
    <div
      v-if="showContainerCloneProgressBanner"
      class="mb-3 p-2 rounded-lg bg-sky-50 border border-sky-100"
      data-testid="container-clone-progress-banner"
    >
      <p class="text-sm font-medium text-sky-900">容器项目克隆进度</p>
      <div
        v-for="(row, cloneRowIdx) in containerCloneProgressEntries"
        :key="row.key"
        :class="cloneRowIdx === 0 ? 'mt-2' : 'mt-3'"
        :data-testid="row.repoUrl ? `container-clone-progress-row-${repoCloneFieldId(row.repoUrl)}` : 'container-clone-progress-row-global'"
      >
        <p v-if="row.repoUrl" class="text-xs font-medium text-sky-900/90 truncate" :title="row.repoUrl">
          {{ shortCloneRepoLabel(row.repoUrl) }}
        </p>
        <template v-if="cloneProgressRowHasSubPhases(row)">
          <div :class="row.repoUrl ? 'mt-1' : 'mt-2'" class="flex gap-3 min-w-0">
            <div class="min-w-0 flex-1">
              <div class="flex justify-between text-[10px] text-sky-800/90 mb-0.5">
                <span>接收数据包</span>
                <span>{{ cloneProgressRecvPct(row) }}%</span>
              </div>
              <div class="w-full bg-sky-100 rounded-full h-2.5 overflow-hidden">
                <div
                  class="bg-sky-600 h-2.5 rounded-full"
                  :class="cloneProgressBarWidthTransitionClass"
                  :style="{ width: cloneProgressRecvPct(row) + '%' }"
                />
              </div>
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex justify-between text-[10px] text-sky-800/90 mb-0.5">
                <span>解压</span>
                <span>{{ cloneProgressUnpackPct(row) }}%</span>
              </div>
              <div class="w-full bg-sky-100 rounded-full h-2.5 overflow-hidden">
                <div
                  class="bg-indigo-500 h-2.5 rounded-full"
                  :class="cloneProgressBarWidthTransitionClass"
                  :style="{ width: cloneProgressUnpackPct(row) + '%' }"
                />
              </div>
            </div>
          </div>
        </template>
        <div
          v-else
          :class="row.repoUrl ? 'mt-1' : 'mt-2'"
          class="w-full bg-sky-100 rounded-full h-2.5 overflow-hidden"
        >
          <div
            class="bg-sky-600 h-2.5 rounded-full"
            :class="cloneProgressBarWidthTransitionClass"
            :style="{ width: Math.min(100, Math.max(0, row.progress)) + '%' }"
          />
        </div>
        <p class="mt-1 text-xs text-sky-800 whitespace-pre-wrap">{{ row.message }}</p>
        <details
          class="mt-1.5 rounded border border-sky-200/80 bg-white/80"
          :data-testid="row.repoUrl ? `container-clone-progress-log-${repoCloneFieldId(row.repoUrl)}` : 'container-clone-progress-log-global'"
        >
          <summary class="cursor-pointer select-none text-xs font-medium text-sky-900 px-2 py-1 hover:bg-sky-100/70">
            克隆日志
          </summary>
          <pre
            class="mt-0 max-h-52 overflow-y-auto whitespace-pre-wrap break-words px-2 pb-2 pt-1 text-[11px] font-mono leading-snug border-t border-sky-100/90"
            :class="cloneProgressRowLogIsPlaceholder(row, cloneRowIdx) ? 'text-slate-500' : 'text-slate-800'"
            >{{ cloneProgressRowLogDisplayText(row, cloneRowIdx) }}</pre>
        </details>
      </div>
    </div>

    <div id="comments-container" class="space-y-4">
      <div v-if="commentsHasMore" class="flex justify-center">
        <button
          type="button"
          class="text-xs text-gray-600 hover:text-gray-900 border border-gray-200 rounded-md px-3 py-1.5 disabled:opacity-50"
          data-testid="comments-load-more"
          :disabled="commentsLoadingMore"
          @click="emit('load-more-comments')"
        >
          {{ commentsLoadingMore ? '加载中…' : '加载更早的评论' }}
        </button>
      </div>
      <TaskDetailConversationFeed
        :comments="displayComments"
        :collaborator-name-by-id="collaboratorNameById"
        :collaborator-avatar-by-id="collaboratorAvatarById"
        :stream-busy="aiStreamBusy"
        :stream-text="aiStreamBuffer"
        :active-container-agent-id="activeContainerAgentId"
        :active-execution-comment-id="activeExecutionCommentId"
        :has-agent-steps="hasZtreeAgentSteps"
        :agent-step-count="agentStepCount"
        :tenant-id="tenantId"
        :task-id="taskId"
        :oauth-readiness="oauthReadiness"
      >
        <template #execution-details="slotProps">
          <slot name="execution-details" v-bind="slotProps" />
        </template>
      </TaskDetailConversationFeed>
      <TaskDetailApplyPatchBar
        :tenant-id="tenantId"
        :workspace-id="workspaceId"
        :task-id="taskId"
      />
    </div>

    <div class="mt-4 pt-4 border-t border-gray-200">
      <h4 class="text-sm font-medium text-gray-500 mb-2">添加评论</h4>
      <div v-if="commentComposerChips.length" class="flex flex-wrap gap-1 mb-2">
        <span
          v-for="(chip, idx) in commentComposerChips"
          :key="`cq-${idx}-${chip}`"
          class="inline-flex items-center gap-1 rounded-full border border-violet-200 bg-violet-50 px-2 py-0.5 text-[11px] text-violet-900"
        >
          {{ chip }}
          <button
            type="button"
            class="text-violet-600 hover:text-violet-900"
            :aria-label="'移除引用'"
            @click="emit('remove-comment-composer-chip', idx)"
          >
            ×
          </button>
        </span>
      </div>
      <TaskDetailCommentComposer
        v-model="newComment"
        :tenant-id="tenantId"
        :workspace-id="workspaceId"
        :task="task"
        :viewer-task-id="taskId"
        :predecessor-options="predecessorOptions"
        :project-server-run-template="projectServerRunTemplate"
        :task-projects-with-details="taskProjectsWithDetails"
        :comments="displayComments"
        @keydown="emit('comment-textarea-keydown', $event)"
        @submit="emit('submit-comment')"
        @task-updated="onComposerTaskUpdated"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import TaskDetailConversationFeed from './TaskDetailConversationFeed.vue'
import TaskDetailApplyPatchBar from './TaskDetailApplyPatchBar.vue'
import TaskDetailCommentComposer from './TaskDetailCommentComposer.vue'
import { buildCommentPredecessorOptions } from '../../composables/taskDetail/useCommentExecutionContext.js'
import { resolveCommentAuthorDisplayName } from '../../utils/taskCardPeopleDisplay.js'

const newComment = defineModel('newComment', { type: String, required: true })

const props = defineProps({
  showContainerCloneProgressBanner: { type: Boolean, required: true },
  containerCloneProgressEntries: { type: Array, required: true },
  repoCloneFieldId: { type: Function, required: true },
  shortCloneRepoLabel: { type: Function, required: true },
  cloneProgressRowHasSubPhases: { type: Function, required: true },
  cloneProgressRecvPct: { type: Function, required: true },
  cloneProgressUnpackPct: { type: Function, required: true },
  cloneProgressBarWidthTransitionClass: { type: String, required: true },
  cloneProgressRowLogIsPlaceholder: { type: Function, required: true },
  cloneProgressRowLogDisplayText: { type: Function, required: true },
  displayComments: { type: Array, required: true },
  collaboratorNameById: { type: Object, default: () => ({}) },
  collaboratorAvatarById: { type: Object, default: () => ({}) },
  commentsFeedErrors: { type: Array, default: () => [] },
  commentsHasMore: { type: Boolean, default: false },
  commentsLoadingMore: { type: Boolean, default: false },
  aiStreamBusy: { type: Boolean, required: true },
  aiStreamBuffer: { type: String, required: true },
  activeContainerAgentId: { type: String, default: '' },
  activeExecutionCommentId: { type: String, default: '' },
  hasZtreeAgentSteps: { type: Boolean, required: true },
  agentStepCount: { type: Number, required: true },
  tenantId: { type: String, required: true },
  workspaceId: { type: String, required: true },
  taskId: { type: String, required: true },
  /** 当前任务对象，透传给评论区镜像选择 hints */
  task: { type: Object, default: null },
  commentComposerChips: { type: Array, required: true },
  projectServerRunTemplate: { type: Object, default: null },
  taskProjectsWithDetails: { type: Array, default: () => [] },
  oauthReadiness: { type: Object, default: null },
})

const emit = defineEmits([
  'remove-comment-composer-chip',
  'comment-textarea-keydown',
  'submit-comment',
  'load-more-comments',
  'task-updated',
])

function onComposerTaskUpdated(data) {
  if (data && typeof data === 'object' && props.task && typeof props.task === 'object') {
    Object.assign(props.task, data)
  }
  emit('task-updated', data)
}

const predecessorOptions = computed(() =>
  buildCommentPredecessorOptions(props.displayComments, (c) =>
    resolveCommentAuthorDisplayName(c?.created_by, props.collaboratorNameById, ''),
  ),
)
</script>
