<template>
  <div class="comments-section task-card-no-drag" @click.stop>
    <div class="flex items-center mb-2 justify-between">
      <div class="flex items-center">
        <button
          type="button"
          class="text-xs text-purple-600 hover:text-purple-800 flex items-center"
          data-testid="task-card-toggle-comments"
          :aria-expanded="showComments"
          @click.stop="toggleComments"
        >
          <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
          </svg>
          评论<span v-if="commentsCount > 0"> ({{ commentsCount }})</span>
        </button>
      </div>
      <button
        v-if="tenantId && workspaceId"
        class="text-xs bg-blue-100 hover:bg-blue-200 text-blue-800 flex items-center px-2 py-1 rounded"
        @click.stop="openTaskInNewTab"
      >
        <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
        </svg>
        新页面打开
      </button>
    </div>

    <div
      v-if="showComments"
      class="comment-list bg-gray-50 p-3 rounded-md mb-3"
      data-testid="task-card-comments-panel"
    >
      <div v-if="task.comments && task.comments.length > 0">
        <div
          v-for="comment in task.comments"
          :key="comment.id"
          class="comment-item mb-3 pb-3 border-b border-gray-100 last:border-b-0 last:mb-0 last:pb-0"
        >
          <div class="flex items-start">
            <img
              class="w-5 h-5 rounded-full border-2 border-white mr-2"
              :src="commentAuthorAvatar(comment)"
              :alt="comment.created_by?.username || '未知用户'"
            >
            <div class="flex-1">
              <div class="flex justify-between items-center mb-1">
                <span class="text-xs font-medium text-gray-800">{{ comment.created_by?.username || '未知用户' }}</span>
                <span class="text-xs text-gray-500">{{ formatCommentDate(comment.created_at) }}</span>
              </div>
              <p class="text-xs text-gray-700">{{ comment.content }}</p>
            </div>
          </div>
        </div>
      </div>
      <p v-else class="text-xs text-gray-500 text-center py-2">暂无评论</p>

      <div class="add-comment-form mt-3 pt-3 border-t border-gray-200" @click.stop>
        <TaskDetailCommentComposer
          v-model="newCommentContent"
          :tenant-id="tenantIdString"
          :workspace-id="workspaceIdString"
          :show-cancel="true"
          :use-legacy-dom-ids="false"
          :show-image-select="false"
          :show-dependency-picker="false"
          :task-projects-with-details="resolvedTaskProjectsWithDetails"
          :comments="taskCommentsList"
          root-test-id="task-card-comment-composer"
          input-test-id="task-card-comment-input"
          @submit="submitComment"
          @cancel="cancelComment"
          @keydown="onCardCommentKeydown"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../utils/apiUtils.js'
import { initialsAvatarDataUri } from '../utils/initialsAvatarDataUri.js'
import { resolveCommentAuthorAvatar } from '../utils/taskCardPeopleDisplay.js'
import TaskDetailCommentComposer from './task-detail/TaskDetailCommentComposer.vue'
import {
  clearPendingImageMention,
  pendingImageMention,
} from '../composables/taskDetail/commentImageMentionState.js'

const props = defineProps({
  task: { type: Object, required: true },
  tenantId: { type: [String, Number], default: null },
  workspaceId: { type: [String, Number], default: null },
  /** 任务关联项目详情（含 project.git_repos）；缺省时按 task.projects 拉取工作区项目。 */
  taskProjectsWithDetails: { type: Array, default: () => [] },
})

const taskCommentsList = computed(() => {
  const rows = props.task?.comments
  return Array.isArray(rows) ? rows : []
})

// 看板卡未传关联项目时，按 task.projects + /api/projects 拉取（OPT-20260816-009）。
const localProjectsWithDetails = ref([])
const resolvedTaskProjectsWithDetails = computed(() => {
  if (Array.isArray(props.taskProjectsWithDetails) && props.taskProjectsWithDetails.length) {
    return props.taskProjectsWithDetails
  }
  return localProjectsWithDetails.value
})

const fetchTaskProjectsWithDetails = async () => {
  if (resolvedTaskProjectsWithDetails.value.length) return
  const tid = tenantIdString.value
  const wid = workspaceIdString.value
  const apiRows = Array.isArray(props.task?.projects) ? props.task.projects : []
  if (!tid || !wid || apiRows.length === 0) return
  try {
    const response = await apiFetch(
      `/api/projects/tenant_id/${encodeURIComponent(tid)}?workspace_id=${encodeURIComponent(wid)}`,
      { credentials: 'include', headers: { Accept: 'application/json' } },
    )
    const data = await response.json().catch(() => ({}))
    const projects = response.ok
      ? Array.isArray(data)
        ? data
        : data.results || []
      : []
    const byId = {}
    for (const p of projects) {
      byId[String(p?.id || '')] = p
    }
    const out = []
    const seen = new Set()
    for (const row of apiRows) {
      const pid = String(row?.project_id || '')
      if (!pid || seen.has(pid)) continue
      seen.add(pid)
      const project = byId[pid]
      if (!project) continue
      out.push({
        project_id: pid,
        project_name: String(project?.name || '') || pid,
        project,
      })
    }
    localProjectsWithDetails.value = out
  } catch (error) {
    console.warn('[TaskCardCommentsSection] fetch task projects failed', error)
  }
}

const emit = defineEmits(['create-comment'])

const route = useRoute()
const showComments = ref(false)
const newCommentContent = ref('')
const commentAuthorAvatar = (comment) => {
  const real = resolveCommentAuthorAvatar(comment?.created_by, null)
  if (real) return real
  return initialsAvatarDataUri(comment?.created_by?.username || '?')
}

const tenantIdString = computed(() =>
  props.tenantId != null ? String(props.tenantId).trim() : '',
)
const workspaceIdString = computed(() =>
  props.workspaceId != null ? String(props.workspaceId).trim() : '',
)
const commentsCount = computed(() => props.task.comments?.length || 0)

const toggleComments = () => {
  showComments.value = !showComments.value
  if (!showComments.value) {
    newCommentContent.value = ''
    clearPendingImageMention()
  } else {
    void fetchTaskProjectsWithDetails()
  }
}

const formatCommentDate = (dateString) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return `${date.getMonth() + 1}-${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`
}

const onCardCommentKeydown = (e) => {
  if (e.key !== 'Enter') return
  if (e.metaKey || e.ctrlKey) {
    e.preventDefault()
    submitComment()
  }
}

const submitComment = () => {
  const content = String(newCommentContent.value || '').trim()
  if (!content) return
  const mention = pendingImageMention.value
  const payload = { content }
  if (mention && mention.id) {
    payload.mentions = [{
      type: 'installed_image',
      id: String(mention.id),
      name: String(mention.name || ''),
    }]
  }
  emit('create-comment', props.task.id, payload)
  newCommentContent.value = ''
  clearPendingImageMention()
}

const cancelComment = () => {
  newCommentContent.value = ''
  clearPendingImageMention()
  showComments.value = false
}

const openTaskInNewTab = () => {
  if (!props.tenantId || !props.workspaceId || !props.task?.id) return
  const taskDetailUrl = `/tenant/${props.tenantId}/workspace/${props.workspaceId}/task-detail/${props.task.id}/`
  const q = new URLSearchParams()
  const rq = route.query || {}
  for (const k of ['accessCode', 'github']) {
    const v = rq[k]
    if (v == null || v === '') continue
    q.set(k, Array.isArray(v) ? String(v[0]) : String(v))
  }
  const s = q.toString()
  window.open(`${window.location.origin}${taskDetailUrl}${s ? `?${s}` : ''}`, '_blank')
}
</script>

<style scoped>
.comments-section {
  margin-top: 12px;
  font-size: 12px;
}

.comment-list {
  max-height: 200px;
  overflow-y: auto;
}

.comment-list::-webkit-scrollbar {
  width: 4px;
}

.comment-list::-webkit-scrollbar-track {
  background: transparent;
}

.comment-list::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.1);
  border-radius: 2px;
}

.comment-item {
  margin-bottom: 12px;
}

.add-comment-form {
  margin-top: 12px;
}
</style>
