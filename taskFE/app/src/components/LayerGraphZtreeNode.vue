<template>
  <li class="relative list-none pl-0">
    <div
      class="flex items-start gap-1 py-0.5 pr-1 rounded transition-colors"
      :style="{ paddingLeft: depth * 14 + 'px' }"
      :class="[rowClass, isSelected ? 'bg-primary/10 ring-1 ring-primary/30' : '']"
    >
      <button
        v-if="hasChildren"
        type="button"
        class="shrink-0 w-4 h-4 text-[10px] leading-4 text-gray-600 border border-gray-300 rounded-sm bg-gray-50 hover:bg-gray-100"
        :aria-expanded="expanded"
        @click.stop="expanded = !expanded"
      >
        {{ expanded ? '−' : '+' }}
      </button>
      <span v-else class="inline-block w-4 shrink-0" aria-hidden="true" />
      <button
        type="button"
        class="break-words flex-1 min-w-0 text-left cursor-pointer hover:opacity-90"
        :title="node.title || node.name"
        @click="onSelect"
      >
        {{ node.name }}
      </button>
      <div
        v-if="node.jobId || node.canSubmit || node.canPush || node.canMerge || node.pushAheadLabel || node.canOpenPr || node.canSubmitAndPush || node.pushErrorLabel"
        class="shrink-0 flex items-center gap-1"
      >
        <button
          v-if="node.canInterrupt"
          type="button"
          class="text-[10px] px-1.5 py-0 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.interruptDisabled || isBusy('interrupt')"
          :title="node.interruptTitle || '中断'"
          @click.stop="onInterrupt"
        >
          {{ isBusy('interrupt') ? '…' : '中断' }}
        </button>
        <button
          v-if="node.canContinue"
          type="button"
          class="text-[10px] px-1.5 py-0 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.continueDisabled || isBusy('continue')"
          :title="node.continueTitle || '继续'"
          @click.stop="onContinue"
        >
          {{ isBusy('continue') ? '…' : '继续' }}
        </button>
        <button
          v-if="node.canRedo"
          type="button"
          data-testid="layer-ztree-redo-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.redoDisabled || isBusy('redo')"
          :title="node.redoTitle || '重新执行'"
          @click.stop="onRedo"
        >
          {{ isBusy('redo') ? '…' : '重新执行' }}
        </button>
        <button
          v-if="node.canEditRun"
          type="button"
          class="text-[10px] px-1.5 py-0 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.editRunDisabled || isBusy('edit')"
          :title="node.editRunTitle || '修改指令后执行'"
          @click.stop="onEditRun"
        >
          {{ isBusy('edit') ? '…' : '改后执行' }}
        </button>
        <button
          v-if="node.canDelete"
          type="button"
          class="text-[10px] px-1.5 py-0 rounded border border-red-300 bg-white text-red-700 hover:bg-red-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.deleteDisabled || isBusy('delete')"
          :title="node.deleteTitle || '删除'"
          @click.stop="onDelete"
        >
          {{ isBusy('delete') ? '…' : '删除' }}
        </button>
        <!-- Combined submit + merge button -->
        <!-- Anti-Replay-OK: 只读失败提示 + clipboard write only; no HTTP -->
        <span
          v-if="node.pushErrorLabel"
          class="inline-flex items-center gap-0.5 shrink-0"
        >
          <span
            data-testid="layer-ztree-push-error-label"
            class="text-[10px] text-red-700 whitespace-nowrap shrink-0"
            :title="node.pushErrorTitle || node.pushErrorLabel"
            :data-traceId="node.pushErrorTraceId || undefined"
          >
            {{ node.pushErrorLabel }}
          </span>
          <!-- Anti-Replay-OK: 真实 a[href] 绑定入口（grant_kind=pending）；禁止 @click 冒充 -->
          <a
            v-if="pushErrorBindHref"
            :href="pushErrorBindHref"
            target="_blank"
            rel="noopener noreferrer"
            data-testid="layer-ztree-push-error-bind"
            class="text-[10px] px-1.5 py-0 rounded border border-amber-300 bg-white text-amber-900 hover:bg-amber-50 no-underline"
            :title="pushErrorBindTitle"
            @click.stop
          >去绑定</a>
          <button
            type="button"
            data-testid="layer-ztree-push-error-copy"
            class="text-[10px] px-1.5 py-0 rounded border border-red-300 bg-white text-red-700 hover:bg-red-50"
            :title="copyPushErrorDone ? '已复制' : '复制失败信息'"
            @click.stop="onCopyPushError"
          >
            {{ copyPushErrorDone ? '已复制' : '复制' }}
          </button>
        </span>
        <button
          v-if="canShowSubmitAndMerge"
          type="button"
          data-testid="layer-ztree-submit-merge-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-emerald-300 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.submitAndMergeDisabled || isBusy('submit_merge')"
          :title="node.submitAndMergeTitle || '提交并合并'"
          @click.stop="onSubmitAndMerge"
        >
          {{ isBusy('submit_merge') ? '…' : '提交并合并' }}
        </button>
        <!-- Combined submit + push + PR button -->
        <button
          v-if="canShowSubmitAndPush"
          type="button"
          data-testid="layer-ztree-submit-push-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-indigo-300 bg-indigo-50 text-indigo-700 hover:bg-indigo-100 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.submitAndPushDisabled || isBusy('submit_push')"
          :title="node.submitAndPushTitle || '提交并创建PR'"
          @click.stop="onSubmitAndPush"
        >
          {{ isBusy('submit_push') ? '…' : '提交并创建PR' }}
        </button>
        <!-- Standalone push-ahead label (only when combined button not shown) -->
        <span
          v-if="!canShowSubmitAndPush && node.pushAheadLabel"
          data-testid="layer-ztree-push-ahead-label"
          class="text-[10px] text-violet-900/85 whitespace-nowrap shrink-0"
          :title="node.pushTitle || ''"
        >
          {{ node.pushAheadLabel }}
        </span>
        <span
          v-if="canShowSubmitAndPush && node.submitFileChangesLabel"
          class="text-[10px] text-indigo-900/85 whitespace-nowrap shrink-0"
          :title="node.submitFileChangesTitle || ''"
        >
          {{ node.submitFileChangesLabel }}
        </span>
        <button
          v-if="!canShowSubmitAndPush && node.canSubmit"
          type="button"
          data-testid="layer-ztree-submit-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-blue-300 bg-white text-blue-700 hover:bg-blue-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.submitDisabled || isBusy('submit')"
          :title="node.submitTitle || '提交'"
          @click.stop="onSubmit"
        >
          {{ isBusy('submit') ? '…' : '提交' }}
        </button>
        <button
          v-if="!canShowSubmitAndPush && node.canPush"
          type="button"
          data-testid="layer-ztree-push-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-violet-300 bg-white text-violet-700 hover:bg-violet-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.pushDisabled || isBusy('push')"
          :title="node.pushTitle || '推送并创建PR'"
          @click.stop="onPush"
        >
          {{ isBusy('push') ? '…' : '推送并创建PR' }}
        </button>
        <a
          v-if="node.canOpenPr && prHref"
          data-testid="layer-ztree-pr-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-sky-300 bg-white text-sky-800 hover:bg-sky-50 no-underline"
          :href="prHref"
          target="_blank"
          rel="noopener noreferrer"
          :title="node.prTitle || '打开 PR 审查页'"
          @click.stop
        >
          PR
        </a>
        <button
          v-if="node.canMerge"
          type="button"
          data-testid="layer-ztree-merge-btn"
          class="text-[10px] px-1.5 py-0 rounded border border-emerald-300 bg-white text-emerald-800 hover:bg-emerald-50 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="node.mergeDisabled || isBusy('merge')"
          :title="node.mergeTitle || '合并到目标分支'"
          @click.stop="onMerge"
        >
          {{ isBusy('merge') ? '…' : '合并到目标分支' }}
        </button>
      </div>
    </div>
    <ul v-if="hasChildren && expanded" class="m-0 list-none p-0">
      <LayerGraphZtreeNode
        v-for="ch in node.children"
        :key="String(ch.id)"
        :node="ch"
        :depth="depth + 1"
        :selected-id="selectedId"
        :action-busy-key="actionBusyKey"
        @node-select="(n) => emit('node-select', n)"
        @job-redo="(id) => emit('job-redo', id)"
        @job-interrupt="(id) => emit('job-interrupt', id)"
        @job-continue="(id) => emit('job-continue', id)"
        @job-edit-run="(n) => emit('job-edit-run', n)"
        @job-delete="(id) => emit('job-delete', id)"
        @layer-delete="(id) => emit('layer-delete', id)"
        @layer-submit="(n) => emit('layer-submit', n)"
        @layer-push="(n) => emit('layer-push', n)"
        @layer-merge="(n) => emit('layer-merge', n)"
        @layer-submit-and-push="(n) => emit('layer-submit-and-push', n)"
        @layer-submit-and-merge="(n) => emit('layer-submit-and-merge', n)"
      />
    </ul>
  </li>
</template>

<script setup>
import { ref, computed, onBeforeUnmount } from 'vue'
import { writeClipboardText } from '../utils/writeClipboardText.js'
import { formatPushErrorClipboardText } from '../utils/layerZtreePushError.js'

const props = defineProps({
  node: { type: Object, required: true },
  depth: { type: Number, default: 0 },
  selectedId: { type: [String, Number], default: null },
  actionBusyKey: { type: String, default: '' },
})

const emit = defineEmits([
  'node-select',
  'job-redo',
  'job-interrupt',
  'job-continue',
  'job-edit-run',
  'job-delete',
  'layer-delete',
  'layer-submit',
  'layer-push',
  'layer-merge',
  'layer-submit-and-push',
  'layer-submit-and-merge',
])

const expanded = ref(props.node.open !== false)
const copyPushErrorDone = ref(false)
let copyPushErrorTimer = null

onBeforeUnmount(() => {
  if (copyPushErrorTimer) clearTimeout(copyPushErrorTimer)
})

async function onCopyPushError() {
  const text = formatPushErrorClipboardText(props.node)
  if (!text) return
  try {
    await writeClipboardText(text)
    copyPushErrorDone.value = true
    if (copyPushErrorTimer) clearTimeout(copyPushErrorTimer)
    copyPushErrorTimer = setTimeout(() => { copyPushErrorDone.value = false }, 1500)
  } catch {
    /* clipboard write may be denied by the browser */
  }
}

const hasChildren = computed(
  () => Array.isArray(props.node.children) && props.node.children.length > 0
)

const canShowSubmitAndPush = computed(
  () =>
    !!props.node.canSubmitAndPush &&
    (!props.node.submitAndPushDisabled || props.node.containerReleased === true),
)

const canShowSubmitAndMerge = computed(
  () => !!(props.node.canSubmit && props.node.canMerge && !props.node.mergeDisabled) && !props.node.submitAndMergeDisabled,
)

const prHref = computed(() => {
  const url = typeof props.node.prHtmlUrl === 'string' ? props.node.prHtmlUrl.trim() : ''
  return /^https?:\/\//i.test(url) ? url : ''
})

/** BINDING_MISSING 真实授权入口：仅接受站内路径或 http(s)，禁止拼任意字符串当 href */
const pushErrorBindHref = computed(() => {
  const url =
    typeof props.node?.pushErrorBindHref === 'string' ? props.node.pushErrorBindHref.trim() : ''
  if (!url) return ''
  return url.startsWith('/') || /^https?:\/\//i.test(url) ? url : ''
})

const pushErrorBindTitle = computed(() => {
  const t =
    typeof props.node?.pushErrorBindTitle === 'string' ? props.node.pushErrorBindTitle.trim() : ''
  return t || '去 Git 站点授权绑定该仓库'
})

const isSelected = computed(() => {
  if (props.selectedId == null || props.selectedId === '') {
    return false
  }
  return String(props.node.id) === String(props.selectedId)
})

function isBusy(action) {
  if (!props.actionBusyKey) return false
  const jobId = props.node.jobId != null ? String(props.node.jobId) : ''
  const layerId = props.node.layerId != null ? String(props.node.layerId) : ''
  if (jobId && props.actionBusyKey === `${action}:job:${jobId}`) return true
  if (layerId && props.actionBusyKey === `${action}:layer:${layerId}`) return true
  return false
}

const rowClass = computed(() => {
  const k = props.node.nodeKind
  const z = props.node.ztStyle
  if (k === 'virtual') return 'font-semibold text-gray-800'
  if (z === 'active') return 'text-amber-700'
  if (z === 'no_git') return 'text-slate-400'
  if (z === 'clean') return 'text-emerald-700'
  if (k === 'cycle') return 'text-red-500'
  return 'text-slate-600'
})

function onSelect() {
  emit('node-select', props.node)
}

function onRedo() {
  if (!props.node.jobId || props.node.redoDisabled) {
    return
  }
  emit('job-redo', String(props.node.jobId))
}

function onInterrupt() {
  if (!props.node.jobId || props.node.interruptDisabled) return
  emit('job-interrupt', String(props.node.jobId))
}

function onContinue() {
  if (!props.node.jobId || props.node.continueDisabled) return
  emit('job-continue', String(props.node.jobId))
}

function onEditRun() {
  if (!props.node.jobId || props.node.editRunDisabled) return
  emit('job-edit-run', props.node)
}

function onDelete() {
  if (props.node.deleteDisabled) return
  if (props.node.layerId) {
    emit('layer-delete', String(props.node.layerId))
    return
  }
  if (!props.node.jobId) return
  emit('job-delete', String(props.node.jobId))
}

function onSubmit() {
  if (!props.node.layerId || props.node.submitDisabled) return
  emit('layer-submit', props.node)
}

function onPush() {
  if (!props.node.layerId || props.node.pushDisabled) return
  emit('layer-push', props.node)
}

function onSubmitAndPush() {
  if (!props.node.layerId || props.node.submitAndPushDisabled) return
  emit('layer-submit-and-push', props.node)
}

function onSubmitAndMerge() {
  if (!props.node.layerId || props.node.submitAndMergeDisabled) return
  emit('layer-submit-and-merge', props.node)
}

function onMerge() {
  if (!props.node.layerId || props.node.mergeDisabled) return
  emit('layer-merge', props.node)
}
</script>
