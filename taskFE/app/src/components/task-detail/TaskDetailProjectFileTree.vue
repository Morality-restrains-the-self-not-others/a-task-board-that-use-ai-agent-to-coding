<template>
  <details
    class="mt-3 rounded border border-gray-200 bg-gray-50"
    :open="isDetailsOpen"
    @toggle="onToggle"
  >
    <summary class="flex items-center gap-1 cursor-pointer select-none px-2 py-1 text-xs text-gray-700">
      <span>项目文件树（所有拉取仓库）</span>
      <button
        type="button"
        class="flex items-center justify-center w-5 h-5 rounded hover:bg-gray-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="filesLoading || !layerId"
        title="刷新文件列表"
        @click.stop="handleRefresh"
      >
        <svg
          class="w-3.5 h-3.5 text-gray-500"
          :class="{ 'animate-spin': filesLoading }"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
      </button>
    </summary>
    <div class="px-2 pb-2">
      <!-- 仅首次无数据时显示加载文案；刷新时保留旧树，避免高度塌陷导致页面抖动 -->
      <p v-if="filesLoading && !treeNodes.length" class="text-xs text-gray-500 mt-2">加载文件列表中…</p>
      <p
        v-if="filesError"
        class="text-xs mt-2 whitespace-pre-wrap break-words"
        :class="filesErrorPending ? 'text-amber-700' : 'text-red-600'"
        :data-testid="filesErrorPending ? 'project-file-tree-waiting' : 'project-file-tree-error'"
        v-bind="filesErrorTraceId ? { 'data-traceId': filesErrorTraceId } : {}"
      >{{ filesError }}</p>
      <p
        v-if="filesTruncated && treeNodes.length"
        class="text-xs text-amber-700 mt-2"
        data-testid="project-file-tree-truncated"
      >顶层文件/目录较多，列表可能不完整；展开子目录时会按需加载。</p>
      <ResizableSplitPane
        v-if="treeNodes.length"
        class="mt-2"
        :class="{ 'opacity-70': filesLoading }"
        storage-key="task-detail-project-file-tree-split"
        data-testid="project-file-tree-body"
      >
        <template #left>
          <div class="max-h-72 overflow-auto rounded border border-gray-200 bg-white px-2 py-1">
            <ul class="space-y-0.5">
              <TaskDetailProjectFileTreeNode
                v-for="node in treeNodes"
                :key="node.path"
                :node="node"
                :selected-path="selectedPath"
                :expanded-paths="expandedPaths"
                @select-file="handleSelectFile"
                @select-dir="handleSelectDir"
                @toggle-dir="toggleDirExpand"
              />
            </ul>
          </div>
        </template>
        <template #right>
          <TaskDetailExecLayerChangePreview
            :preview-kind="previewKind"
            :selected-path="selectedPath"
            :loading="previewLoading"
            :error="previewError"
            :error-trace-id="previewErrorTraceId"
            :payload="previewPayload"
            :git-log-text="gitLogText"
            :is-repo-root="gitLogIsRepoRoot"
            :current-branch="gitLogCurrentBranch"
          />
        </template>
      </ResizableSplitPane>
      <p v-else-if="!filesLoading" class="text-xs text-gray-400 mt-2">暂无可展示文件</p>
    </div>
  </details>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { apiFetch } from '../../utils/apiUtils.js'
import { resolveTaskRouteIds } from '../../utils/resolveTaskRouteIds.js'
import { extractTraceId } from '../../utils/traceId.js'
import { appendCommentIdPath, buildTreeFromFiles, selectionPathStillExists } from '../../utils/taskDetailProjectFileTreeHelpers.js'
import { userFacingContainerComputeFailure, isContainerComputeWaitingMessage } from '../../utils/containerComputeWaiting.js'
import {
  fetchLayerFileContentWithPrefixFallback,
  fetchLayerRepoPathPrefixes,
} from '../../utils/taskDetailFetchLayerFileContent.js'
import { useFileTreeLazyChildren, entriesToPathStrings } from '../../composables/taskDetail/useFileTreeLazyChildren.js'
import ResizableSplitPane from '../ResizableSplitPane.vue'
import TaskDetailProjectFileTreeNode from './TaskDetailProjectFileTreeNode.vue'
import TaskDetailExecLayerChangePreview from './TaskDetailExecLayerChangePreview.vue'

const props = defineProps({
  layerId: { type: String, default: '' },
  expandByDefault: { type: Boolean, default: false },
  containerEndpointRegistered: { type: Boolean, default: false },
  containerReleased: { type: Boolean, default: false },
  fileTreeRefreshNonce: { type: Number, default: 0 },
  tenantId: { type: String, default: '' },
  workspaceId: { type: String, default: '' },
  taskId: { type: String, default: '' },
  commentId: { type: String, default: '' },
})

const route = useRoute()
const isDetailsOpen = ref(!!props.expandByDefault)
const filesLoading = ref(false)
const filesError = ref('')
const filesErrorTraceId = ref('')
const filesErrorPending = computed(() => isContainerComputeWaitingMessage(filesError.value))
const filesTruncated = ref(false)
const flatFiles = ref([])
const treeNodes = ref([])
const selectedPath = ref('')
/** @type {import('vue').Ref<'file'|'git'>} */
const previewKind = ref('file')
const previewLoading = ref(false)
const previewError = ref('')
const previewErrorTraceId = ref('')
const previewPayload = ref(null)
const gitLogText = ref('')
const gitLogIsRepoRoot = ref(false)
const gitLogCurrentBranch = ref('')
/** 并发拉取序号：仅最新一次响应可写入，避免慢请求覆盖新数据 */
let filesFetchSeq = 0
/** 当前层已展开目录 path；替换 Set 引用以触发视图更新 */
const expandedPaths = ref(new Set())
/** 按 layerId 缓存展开集合，跨层切换后可恢复 */
const expandedPathsByLayer = new Map()
/**
 * 按 layerId 缓存选中 path 与预览类型，切回层后重新拉取预览。
 * @type {Map<string, { path: string, kind: 'file'|'git' }>}
 */
const selectionByLayer = new Map()
/** 切换层后待恢复选中的 layerId；文件树拉取成功后消费一次 */
let pendingSelectionRestoreLayerId = ''
/** 按 layerId 缓存仓库 rel_prefix，供旧 children 无前缀路径预览回退 */
const repoPrefixesByLayer = new Map()

const requestContext = computed(() => ({
  ...resolveTaskRouteIds({ tenantId: props.tenantId, workspaceId: props.workspaceId, taskId: props.taskId, routeParams: route.params }),
  commentId: String(props.commentId || '').trim(),
}))

function persistExpandedForLayer(layerId) {
  const lid = String(layerId || '').trim()
  if (!lid) return
  expandedPathsByLayer.set(lid, new Set(expandedPaths.value))
}

function restoreExpandedForLayer(layerId) {
  const lid = String(layerId || '').trim()
  const cached = lid ? expandedPathsByLayer.get(lid) : null
  expandedPaths.value = cached ? new Set(cached) : new Set()
}

/** 按当前文件列表剪枝已失效的展开目录（须在有新列表后调用，勿对空列表剪枝） */
function pruneExpandedPathsForFiles(files) {
  const next = new Set()
  for (const p of expandedPaths.value) {
    if (selectionPathStillExists(p, 'git', files)) next.add(p)
  }
  expandedPaths.value = next
  const lid = String(props.layerId || '').trim()
  if (lid) expandedPathsByLayer.set(lid, new Set(next))
}

const { childrenCache, loadingDirs, fetchChildren, clearChildrenCache } =
  useFileTreeLazyChildren(
    () => props.layerId,
    requestContext,
    flatFiles,
    treeNodes,
    () => props.containerReleased,
  )

function persistSelectionForLayer(layerId) {
  const lid = String(layerId || '').trim()
  if (!lid) return
  const path = String(selectedPath.value || '').trim()
  if (!path) {
    selectionByLayer.delete(lid)
    return
  }
  selectionByLayer.set(lid, {
    path,
    kind: previewKind.value === 'git' ? 'git' : 'file',
  })
}

function clearPreviewUi() {
  selectedPath.value = ''
  previewKind.value = 'file'
  previewError.value = ''
  previewErrorTraceId.value = ''
  previewPayload.value = null
  gitLogText.value = ''
  gitLogIsRepoRoot.value = false
  gitLogCurrentBranch.value = ''
}

async function restoreSelectionForLayer(layerId) {
  const lid = String(layerId || '').trim()
  const cached = lid ? selectionByLayer.get(lid) : null
  if (!cached?.path) return
  // 切回层时若 path 已不在新列表中：静默清缓存，避免预览区报错
  if (!selectionPathStillExists(cached.path, cached.kind, flatFiles.value)) {
    selectionByLayer.delete(lid)
    clearPreviewUi()
    return
  }
  if (cached.kind === 'git') {
    await handleSelectDir(cached.path)
  } else {
    await handleSelectFile(cached.path)
  }
}

function toggleDirExpand(path) {
  const p = String(path || '').trim()
  if (!p) return
  const next = new Set(expandedPaths.value)
  if (next.has(p)) {
    next.delete(p)
  } else {
    next.add(p)
    // 首次展开时懒加载子目录
    if (!childrenCache.has(p)) {
      void fetchChildren(p)
    }
  }
  expandedPaths.value = next
  const lid = String(props.layerId || '').trim()
  if (lid) expandedPathsByLayer.set(lid, new Set(next))
}

/**
 * 拉取文件树顶层条目（按需懒加载子目录）。
 * 刷新时保留已有 treeNodes（stale-while-revalidate），
 * 避免 v-if 切换导致布局高度塌陷、页面抖动。
 */
async function fetchFiles() {
  const layerId = String(props.layerId || '').trim()
  if (!layerId) {
    filesError.value = '缺少 layer_id，无法拉取文件树'
    return
  }
  const { tenantId, workspaceId, taskId, commentId } = requestContext.value
  if (!tenantId || !workspaceId || !taskId) {
    filesError.value = '缺少路由上下文，无法拉取文件树'
    return
  }
  const seq = ++filesFetchSeq
  filesLoading.value = true
  filesError.value = ''
  filesErrorTraceId.value = ''
  // 刷新时清空懒加载缓存，避免与最新容器状态不一致
  clearChildrenCache()
  // 故意不在此清空 treeNodes / flatFiles：有旧数据时继续展示，等新结果到位再替换
  try {
    const apiPath = appendCommentIdPath(
      `/api/cloud/compute/container-layer-children/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}/` +
      `?layer_id=${encodeURIComponent(layerId)}`,
      commentId,
    )
    const resp = await apiFetch(apiPath, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (seq !== filesFetchSeq) return
    if (!resp.ok) {
      const mapped = userFacingContainerComputeFailure(data.detail, resp.status)
      const err = new Error(mapped.message || data.detail || `读取文件树失败（HTTP ${resp.status}）`)
      err.traceId = extractTraceId(resp) || extractTraceId(data)
      throw err
    }
    const entries = Array.isArray(data.entries) ? data.entries : []
    const paths = entriesToPathStrings(entries)
    flatFiles.value = paths
    filesTruncated.value = data.truncated === true
    treeNodes.value = buildTreeFromFiles(paths)
    pruneExpandedPathsForFiles(paths)
  } catch (err) {
    if (seq !== filesFetchSeq) return
    filesError.value = err?.message || '读取文件树失败'
    filesErrorTraceId.value = extractTraceId(err)
  } finally {
    if (seq === filesFetchSeq) {
      filesLoading.value = false
    }
  }
}

async function handleSelectDir(path) {
  const relPath = String(path || '').trim()
  if (!relPath) return
  selectedPath.value = relPath
  previewKind.value = 'git'
  previewError.value = ''
  previewErrorTraceId.value = ''
  previewPayload.value = null
  gitLogText.value = ''
  gitLogIsRepoRoot.value = false
  gitLogCurrentBranch.value = ''
  const layerId = String(props.layerId || '').trim()
  persistSelectionForLayer(layerId)
  const { tenantId, workspaceId, taskId, commentId } = requestContext.value
  if (!layerId || !tenantId || !workspaceId || !taskId) {
    previewError.value = '缺少必要参数，无法读取提交日志'
    return
  }
  previewLoading.value = true
  try {
    const apiPath = appendCommentIdPath(
      `/api/cloud/compute/container-layer-git-log/tenant_id/${encodeURIComponent(tenantId)}/workspace_id/${encodeURIComponent(workspaceId)}/task_id/${encodeURIComponent(taskId)}/` +
      `?layer_id=${encodeURIComponent(layerId)}&path=${encodeURIComponent(relPath)}&limit=30`,
      commentId,
    )
    const resp = await apiFetch(apiPath, {
      credentials: 'include',
      headers: { Accept: 'application/json' },
    })
    const data = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      const d = data.detail
      const msg =
        typeof d === 'string'
          ? d
          : d != null
            ? JSON.stringify(d)
            : `读取提交日志失败（HTTP ${resp.status}）`
      const err = new Error(msg)
      err.traceId = extractTraceId(resp) || extractTraceId(data)
      throw err
    }
    const t = typeof data.text === 'string' ? data.text : ''
    gitLogText.value = t
    if (!t.trim() && Array.isArray(data.commits) && data.commits.length === 0) {
      gitLogText.value = '（暂无提交记录）'
    }
    gitLogIsRepoRoot.value = data.is_repo_root === true
    const branch = typeof data.current_branch === 'string' ? data.current_branch.trim() : ''
    gitLogCurrentBranch.value = gitLogIsRepoRoot.value && branch ? branch : ''
  } catch (err) {
    previewError.value = err?.message || '读取提交日志失败'
    previewErrorTraceId.value = extractTraceId(err)
  } finally {
    previewLoading.value = false
  }
}

async function ensureRepoPrefixes(layerId) {
  const lid = String(layerId || '').trim()
  if (!lid) return []
  if (repoPrefixesByLayer.has(lid)) return repoPrefixesByLayer.get(lid)
  const prefixes = await fetchLayerRepoPathPrefixes({ ...requestContext.value, layerId: lid })
  repoPrefixesByLayer.set(lid, prefixes)
  return prefixes
}

async function handleSelectFile(path) {
  const relPath = String(path || '').trim()
  if (!relPath) return
  selectedPath.value = relPath
  previewKind.value = 'file'
  previewError.value = ''
  previewErrorTraceId.value = ''
  previewPayload.value = null
  gitLogText.value = ''
  gitLogIsRepoRoot.value = false
  gitLogCurrentBranch.value = ''
  const layerId = String(props.layerId || '').trim()
  persistSelectionForLayer(layerId)
  const ctx = requestContext.value
  if (!layerId || !ctx.tenantId || !ctx.workspaceId || !ctx.taskId) {
    previewError.value = '缺少必要参数，无法读取文件内容'
    return
  }
  previewLoading.value = true
  try {
    const prefixes = await ensureRepoPrefixes(layerId)
    const result = await fetchLayerFileContentWithPrefixFallback(
      { ...ctx, layerId },
      relPath,
      prefixes,
    )
    if (!result.ok) {
      previewError.value = result.error || '读取文件内容失败'
      previewErrorTraceId.value = result.traceId || ''
      return
    }
    if (result.path && result.path !== relPath) {
      selectedPath.value = result.path
      persistSelectionForLayer(layerId)
    }
    previewPayload.value = result.data
  } catch (err) {
    previewError.value = err?.message || '读取文件内容失败'
    previewErrorTraceId.value = extractTraceId(err)
  } finally {
    previewLoading.value = false
  }
}

function onToggle(ev) {
  isDetailsOpen.value = !!ev?.target?.open
}

function handleRefresh() {
  void fetchFiles()
}

watch(
  () => props.expandByDefault,
  (v) => {
    if (v) isDetailsOpen.value = true
  }
)

/** 展开且 layerId 就绪时拉取；避免仅靠 onMounted + loaded 与原生 <details> toggle 竞态导致从未请求或漏请求 */
watch(
  () => [!!isDetailsOpen.value, String(props.layerId || '').trim()],
  ([open, lid], prev) => {
    const prevLid = prev ? String(prev[1] || '').trim() : ''
    // 切换层时丢弃旧树及懒加载缓存，避免短暂展示错误层文件；同层刷新则保留（见 fetchFiles）
    if (lid !== prevLid) {
      if (prevLid) {
        persistExpandedForLayer(prevLid)
        persistSelectionForLayer(prevLid)
      }
      flatFiles.value = []
      treeNodes.value = []
      clearChildrenCache()
      loadingDirs.value = new Set()
      filesError.value = ''
      filesErrorTraceId.value = ''
      filesTruncated.value = false
      clearPreviewUi()
      restoreExpandedForLayer(lid)
      pendingSelectionRestoreLayerId = lid
    }
    if (!open || !lid) return
    void (async () => {
      await fetchFiles()
      const currentLid = String(props.layerId || '').trim()
      if (
        pendingSelectionRestoreLayerId
        && pendingSelectionRestoreLayerId === currentLid
        && currentLid === lid
      ) {
        pendingSelectionRestoreLayerId = ''
        await restoreSelectionForLayer(currentLid)
      }
    })()
  },
  { immediate: true }
)

/** 容器就绪后重新拉取文件列表（解决层节点先选中但容器未就绪导致列表为空的问题） */
watch(
  () => props.containerEndpointRegistered,
  (registered) => {
    if (!registered) return
    const lid = String(props.layerId || '').trim()
    if (!lid) return
    if (!isDetailsOpen.value) return
    void fetchFiles()
  }
)

/** 克隆完成时层图会更新，但本组件不感知；由父任务页 bump nonce 触发重新拉取 */
watch(
  () => Number(props.fileTreeRefreshNonce || 0),
  (n, prev) => {
    if (n <= 0) return
    if (prev != null && n === prev) return
    const lid = String(props.layerId || '').trim()
    if (!lid) return
    if (!isDetailsOpen.value) return
    void fetchFiles()
  }
)
</script>
