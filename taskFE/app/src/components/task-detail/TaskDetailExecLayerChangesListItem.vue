<template>
  <li class="text-xs flex items-stretch gap-1">
    <button
      type="button"
      class="min-w-0 flex-1 flex items-center justify-between gap-2 rounded px-1.5 py-1 text-left hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-primary/40"
      :class="isSelected ? 'bg-blue-50 ring-1 ring-blue-200' : ''"
      @click="handleRowOpenPreview"
    >
      <span class="min-w-0 flex items-start gap-1.5">
        <TaskDetailFileTypeIcon class="mt-0.5" :path="change.path || ''" />
        <span class="font-mono text-gray-800 break-all">{{ change.path }}</span>
      </span>
      <TaskDetailExecLayerChangeKind :kind="change.kind || ''" />
    </button>
    <button
      v-if="showUnstageButton"
      type="button"
      class="shrink-0 self-center rounded border border-gray-300 bg-white px-1.5 py-0.5 text-[11px] text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="gitActionsDisabled || rowActionBusy"
      :title="gitActionsDisabled ? '容器不可用时无法操作' : '从暂存区移出'"
      data-testid="task-detail-layer-changes-unstage"
      @click.stop="handleUnstage"
    >
      {{ unstageBusy ? '…' : '撤销' }}
    </button>
    <button
      v-if="showStageButton"
      type="button"
      class="shrink-0 self-center rounded border border-gray-300 bg-white px-1.5 py-0.5 text-[11px] text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="gitActionsDisabled || rowActionBusy"
      :title="gitActionsDisabled ? '容器不可用时无法暂存' : '加入暂存区'"
      data-testid="task-detail-layer-changes-add-to-stage"
      @click.stop="handleAddToStage"
    >
      {{ stageBusy ? '…' : '添加' }}
    </button>
  </li>
</template>

<script setup>
import { ref, computed } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { extractTraceId } from '../../utils/traceId.js'
import { jsonPostWithCommentId } from '../../utils/containerForwardCommentId.js'
import { containerComputeFuncFirstUrl } from '../../composables/taskDetail/containerComputeRequest.js'
import TaskDetailExecLayerChangeKind from './TaskDetailExecLayerChangeKind.vue'
import TaskDetailFileTypeIcon from './TaskDetailFileTypeIcon.vue'

const props = defineProps({
  change: {
    type: Object,
    default: () => ({ path: '', kind: '' }),
  },
  isSelected: { type: Boolean, default: false },
  layerId: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  /** 为 true 时禁止「添加」（容器未就绪或不可达） */
  gitActionsDisabled: { type: Boolean, default: false },
  containerPageUrl: { type: String, default: '' },
  commentId: { type: String, default: '' },
})

const emit = defineEmits(['select', 'preview-loading', 'preview-loaded', 'preview-error', 'staged-success'])

const showStageButton = computed(() => {
  const c = props.change
  if (c.git_layer_diff_only) return false
  return !(c.git_staged && !c.git_unstaged)
})

/** 已在暂存区（可「撤销」）；与未暂存可同时为 true（部分已暂存仍有工作区改动） */
const showUnstageButton = computed(() => {
  if (props.change?.git_layer_diff_only) return false
  return Boolean(props.change?.git_staged)
})

const stageBusy = ref(false)
const unstageBusy = ref(false)
const rowActionBusy = computed(() => stageBusy.value || unstageBusy.value)

function handleRowOpenPreview() {
  void openPreview()
}

async function openPreview() {
  const relPath = String(props.change?.path || '').trim()
  emit('select', relPath)
  if (!relPath) {
    emit('preview-error', { path: '', error: '缺少文件路径，无法读取内容' })
    return
  }
  const layerId = String(props.layerId || '').trim()
  if (!layerId) {
    emit('preview-error', { path: relPath, error: '缺少 layer_id，无法读取内容' })
    return
  }
  const tenantId = String(props.tenantId || '').trim()
  const workspaceId = String(props.workspaceId || '').trim()
  const taskId = String(props.taskId || '').trim()
  if (!tenantId || !workspaceId || !taskId) {
    emit('preview-error', { path: relPath, error: '缺少路由上下文，无法读取内容' })
    return
  }
  if (props.change?.kind === 'removed') {
    emit('preview-loaded', { path: relPath, payload: { kind: 'text', text: '该文件已被删除' } })
    return
  }
  emit('preview-loading', { path: relPath })
  try {
    const apiPath = containerComputeFuncFirstUrl(
      tenantId,
      workspaceId,
      taskId,
      'container-layer-file-content',
      props.commentId,
      `layer_id=${encodeURIComponent(layerId)}&path=${encodeURIComponent(relPath)}`,
    )
    const resp = await apiFetch(apiPath, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      const err = new Error(
        (typeof data.detail === 'string' && data.detail) ||
          `读取文件内容失败（HTTP ${resp.status}）`,
      )
      err.traceId = extractTraceId(resp) || extractTraceId(data) || ''
      throw err
    }
    emit('preview-loaded', { path: relPath, payload: data && typeof data === 'object' ? data : null })
  } catch (err) {
    emit('preview-error', {
      path: relPath,
      error: err?.message || '读取文件内容失败',
      traceId: extractTraceId(err) || '',
    })
  }
}

async function handleAddToStage() {
  if (props.gitActionsDisabled) return
  const relPath = String(props.change?.path || '').trim()
  if (!relPath) return
  const layerId = String(props.layerId || '').trim()
  if (!layerId) {
    window.alert('缺少 layer_id，无法加入暂存区')
    return
  }
  const tenantId = String(props.tenantId || '').trim()
  const workspaceId = String(props.workspaceId || '').trim()
  const taskId = String(props.taskId || '').trim()
  if (!tenantId || !workspaceId || !taskId) {
    window.alert('缺少任务上下文，无法加入暂存区')
    return
  }
  stageBusy.value = true
  try {
    const u = String(props.containerPageUrl || '').trim()
    const body = { layer_id: layerId, path: relPath, ...(u ? { container_page_url: u } : {}) }
    const response = await apiFetch(
      containerComputeFuncFirstUrl(
        tenantId,
        workspaceId,
        taskId,
        'container-layer-git-add',
        props.commentId,
      ),
      jsonPostWithCommentId(body, props.commentId),
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const d = data.detail
      const msg = typeof d === 'string' ? d : '加入暂存区失败'
      showRequestError(msg, data)
      return
    }
    const sug =
      data.suggested_commit_message != null && typeof data.suggested_commit_message === 'string'
        ? data.suggested_commit_message.trim()
        : ''
    emit('staged-success', { path: relPath, ...(sug ? { suggestedCommitMessage: sug } : {}) })
  } catch (e) {
    showRequestError(e?.message || '网络错误，加入暂存区失败', e)
  } finally {
    stageBusy.value = false
  }
}

async function handleUnstage() {
  if (props.gitActionsDisabled) return
  const relPath = String(props.change?.path || '').trim()
  if (!relPath) return
  const layerId = String(props.layerId || '').trim()
  if (!layerId) {
    window.alert('缺少 layer_id，无法从暂存区移出')
    return
  }
  const tenantId = String(props.tenantId || '').trim()
  const workspaceId = String(props.workspaceId || '').trim()
  const taskId = String(props.taskId || '').trim()
  if (!tenantId || !workspaceId || !taskId) {
    window.alert('缺少任务上下文，无法从暂存区移出')
    return
  }
  unstageBusy.value = true
  try {
    const u = String(props.containerPageUrl || '').trim()
    const body = { layer_id: layerId, path: relPath, ...(u ? { container_page_url: u } : {}) }
    const response = await apiFetch(
      containerComputeFuncFirstUrl(
        tenantId,
        workspaceId,
        taskId,
        'container-layer-git-unstage',
        props.commentId,
      ),
      jsonPostWithCommentId(body, props.commentId),
    )
    const data = await response.json().catch(() => ({}))
    if (!response.ok) {
      const d = data.detail
      const msg = typeof d === 'string' ? d : '从暂存区移出失败'
      showRequestError(msg, data)
      return
    }
    emit('staged-success', { path: relPath })
  } catch (e) {
    showRequestError(e?.message || '网络错误，从暂存区移出失败', e)
  } finally {
    unstageBusy.value = false
  }
}
</script>
