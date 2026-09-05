<template>
  <div class="p-3 bg-white border border-gray-200 rounded-lg">
    <h3 class="text-sm font-medium text-gray-500 mb-2">分支策略</h3>
    <div v-if="!isEditing" class="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
      <div>
        <p class="text-gray-500 mb-0.5">工作分支</p>
        <p class="font-mono text-gray-900 break-all">{{ taskBranchDisplay.work || '未配置' }}</p>
      </div>
      <div>
        <p class="text-gray-500 mb-0.5">合并目标分支</p>
        <p class="font-mono text-gray-900 break-all">{{ taskBranchDisplay.merge || '未配置' }}</p>
      </div>
    </div>
    <div v-else-if="editingTask" class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <div class="space-y-1.5">
        <label for="edit-work-branch" class="text-xs text-gray-500 block">工作分支</label>
        <input
          id="edit-work-branch"
          v-model="editingTask.workBranchName"
          type="text"
          list="edit-work-branch-preset-options"
          class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-primary focus:border-primary font-mono"
          placeholder="选择模版或手动输入分支名"
          @input="onWorkBranchInput"
          @change="onWorkBranchChange"
        />
        <datalist id="edit-work-branch-preset-options">
          <option
            v-for="opt in workBranchDatalistOptions"
            :key="`edit-work-${opt.preset}`"
            :value="opt.value"
            :label="opt.label"
          />
        </datalist>
        <div class="flex flex-wrap gap-2 text-[10px]">
          <span class="text-gray-500">可从下拉选模版，或直接改分支名</span>
          <span v-if="isTaskTitleTranslating" class="text-gray-500">正在翻译...</span>
          <span
            v-if="taskTitleTranslationError"
            class="text-red-600"
            v-bind="taskTitleTranslationErrorTraceId ? { 'data-traceId': taskTitleTranslationErrorTraceId } : {}"
          >{{ taskTitleTranslationError }}</span>
        </div>
      </div>
      <div class="space-y-1.5">
        <div class="flex items-center justify-between gap-2">
          <label for="edit-merge-target-branch" class="text-xs text-gray-500 block">目标分支</label>
          <button
            type="button"
            class="text-[11px] px-1.5 py-0.5 text-primary hover:bg-primary/5 rounded"
            :disabled="commonMergeTargetBranchesLoading"
            @click="emit('refresh-common-merge-target-branches')"
          >
            {{ commonMergeTargetBranchesLoading ? '加载中…' : '刷新共有分支' }}
          </button>
        </div>
        <input
          id="edit-merge-target-branch"
          v-model="editingTask.mergeTargetName"
          type="text"
          list="edit-merge-target-common-options"
          class="w-full px-2 py-1.5 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-primary focus:border-primary font-mono"
          placeholder="选择各仓库共有分支或手动输入"
          @input="onMergeTargetInput"
          @change="onMergeTargetChange"
        />
        <datalist id="edit-merge-target-common-options">
          <option
            v-for="branch in commonMergeTargetBranches"
            :key="`edit-merge-${branch}`"
            :value="branch"
          />
        </datalist>
        <div class="flex flex-wrap gap-2 text-[10px]">
          <span v-if="commonMergeTargetBranchesLoading" class="text-gray-500">正在拉取各仓库分支…</span>
          <span v-else-if="commonMergeTargetBranchesError" class="text-red-600" v-bind="commonMergeTargetBranchesErrorTraceId ? { 'data-traceId': commonMergeTargetBranchesErrorTraceId } : {}">{{ commonMergeTargetBranchesError }}</span>
          <span
            v-else-if="commonMergeTargetBranches.length === 0"
            class="text-amber-700"
          >暂无共有分支，可手动输入</span>
          <span v-else class="text-gray-500">共有分支 {{ commonMergeTargetBranches.length }} 个</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick } from 'vue'
import { WORK_BRANCH_PRESET_OPTIONS } from '../../utils/workBranchPresetOptions.js'
import {
  buildWorkBranchName,
  resolveBranchNamePlaceholders,
  sanitizeBranchSegment,
} from '../../utils/taskDetailBranchAndRepoUtils.js'

const props = defineProps({
  task: { type: Object, required: true },
  taskId: { type: String, default: '' },
  isEditing: { type: Boolean, default: false },
  editingTask: { type: Object, default: null },
  companyUserName: { type: String, default: '' },
  isTaskTitleTranslating: { type: Boolean, default: false },
  taskTitleTranslationError: { type: String, default: '' },
  taskTitleTranslationErrorTraceId: { type: String, default: '' },
  commonMergeTargetBranches: { type: Array, default: () => [] },
  commonMergeTargetBranchesLoading: { type: Boolean, default: false },
  commonMergeTargetBranchesError: { type: String, default: '' },
  commonMergeTargetBranchesErrorTraceId: { type: String, default: '' },
})

const emit = defineEmits(['refresh-common-merge-target-branches'])

const titleSegmentForPresets = computed(() =>
  sanitizeBranchSegment(props.editingTask?.title || props.task?.title || '', 'task'),
)

const workBranchBuildCtx = computed(() => ({
  companyUserName: props.companyUserName,
  localTaskTitle: props.editingTask?.title || props.task?.title || '',
  customWorkBranchFallback: String(props.editingTask?.workBranchName || ''),
  taskCreatedAt: props.task?.created_at,
}))

const workBranchDatalistOptions = computed(() => {
  if (!props.editingTask) return []
  const tid = String(props.taskId || props.editingTask.id || '').trim()
  const titleSegment = titleSegmentForPresets.value
  const ctx = workBranchBuildCtx.value
  return WORK_BRANCH_PRESET_OPTIONS.filter((preset) => preset.value !== 'custom').map((preset) => ({
    preset: preset.value,
    value: buildWorkBranchName(preset.value, titleSegment, tid, ctx),
    label: preset.label,
  }))
})

const blurIfPicked = async (inputEl, candidates, value) => {
  const trimmed = String(value || '').trim()
  if (!trimmed || !Array.isArray(candidates) || candidates.length === 0) return
  if (!candidates.includes(trimmed)) return
  await nextTick()
  inputEl?.blur?.()
}

const syncWorkBranchPresetFromName = () => {
  if (!props.editingTask) return
  const branchName = String(props.editingTask.workBranchName || '')
  const tid = String(props.taskId || props.editingTask.id || '').trim()
  const titleSegment = titleSegmentForPresets.value
  const ctx = workBranchBuildCtx.value
  let matched = 'custom'
  for (const preset of WORK_BRANCH_PRESET_OPTIONS) {
    if (preset.value === 'custom') continue
    if (buildWorkBranchName(preset.value, titleSegment, tid, ctx) === branchName) {
      matched = preset.value
      break
    }
  }
  props.editingTask.workBranchPreset = matched
}

const onWorkBranchInput = () => {
  syncWorkBranchPresetFromName()
}

const onWorkBranchChange = async (event) => {
  syncWorkBranchPresetFromName()
  const candidates = workBranchDatalistOptions.value.map((opt) => opt.value)
  await blurIfPicked(event?.target, candidates, event?.target?.value)
}

const onMergeTargetInput = () => {
  if (!props.editingTask) return
  props.editingTask.mergeTargetPreset = 'custom'
}

const onMergeTargetChange = async (event) => {
  if (!props.editingTask) return
  props.editingTask.mergeTargetPreset = 'custom'
  const value = String(event?.target?.value ?? '').trim()
  await blurIfPicked(event?.target, props.commonMergeTargetBranches, value)
}

const taskBranchDisplay = computed(() => {
  const tid = String(props.taskId || '').trim()
  const titleSegment = sanitizeBranchSegment(props.task?.title || '', 'task')
  const sub = (s) =>
    resolveBranchNamePlaceholders(s, { taskId: tid, taskTitleSegment: titleSegment })
  const bs = props.task?.branch_strategy
  if (!bs || typeof bs !== 'object') {
    return { work: '', merge: '' }
  }
  const work =
    (typeof bs.work_branch_name === 'string' && bs.work_branch_name.trim()) ||
    (typeof bs.target_branch_name === 'string' && bs.target_branch_name.trim()) ||
    ''
  const merge =
    (typeof bs.merge_target_branch_name === 'string' && bs.merge_target_branch_name.trim()) || 'develop'
  return { work: sub(work), merge: sub(merge) }
})
</script>
