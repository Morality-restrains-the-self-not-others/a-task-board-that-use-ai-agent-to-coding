<template>
  <!-- 默认折叠：运行中展开（配合「跟步」），结束后（代理步骤已结束）自动收起 -->
  <details
    v-if="cards.length"
    class="group/agent-steps mt-1 space-y-1"
    :open="isAgentStepsRunning"
    data-testid="layer-agent-steps-cards"
  >
    <div class="flex items-center justify-between">
      <TaskDetailAgentStepsHeader :running="isAgentStepsRunning" />
      <label v-if="isAgentStepsRunning" class="flex items-center gap-1 text-xs text-gray-400 cursor-pointer select-none">
        <input type="checkbox" v-model="followRunning" class="w-3 h-3" />
        跟步
      </label>
    </div>
    <div class="space-y-1" role="list" data-testid="layer-agent-steps-accordion">
      <TaskDetailAgentStepCard
        v-for="card in cards"
        :key="card.key"
        :card="card"
        :copy-feedback-key="copyFeedbackKey"
        :open="openStepKey === card.key"
        @copy-json="onCopyJson"
        @rich-interact="onRichInteract"
        @toggle="onStepToggle"
      />
    </div>
  </details>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import TaskDetailAgentStepCard from './TaskDetailAgentStepCard.vue'
import TaskDetailAgentStepsHeader from './TaskDetailAgentStepsHeader.vue'

const props = defineProps({
  copyFeedbackKey: { type: String, default: '' },
  jobStatus: { type: String, default: '' },
  cards: {
    type: Array,
    default: () => [],
  },
})

/** 空串 = 全部折叠；仅用户点击 summary 时打开（手风琴仍最多一开） */
const openStepKey = ref('')

// OPT-20260719-030: optional "follow running step" toggle, persisted to localStorage.
const LS_KEY = 'agent-steps-follow-running'
const followRunning = ref(_readFollowRunning())

function _readFollowRunning() {
  try { return localStorage.getItem(LS_KEY) === '1' } catch { return false }
}
function _persistFollowRunning(v) {
  try { localStorage.setItem(LS_KEY, v ? '1' : '0') } catch { /* ignore */ }
}

function isStepActive(card) {
  const st = card?.rawStep?.state
  return st != null && st !== '' && st !== 'completed' && st !== 'error'
}

watch(followRunning, (v) => _persistFollowRunning(v))

watch(
  () => props.cards,
  (cards) => {
    const list = Array.isArray(cards) ? cards : []
    if (!list.length) {
      openStepKey.value = ''
      return
    }
    // 列表刷新时：若当前打开的 key 已不存在则收起
    const stillExists = list.some((c) => c.key === openStepKey.value)
    if (!stillExists) {
      openStepKey.value = ''
    }
    // Auto-follow: open the first active (running) step when toggle is on
    if (followRunning.value && !openStepKey.value) {
      const active = list.find((c) => isStepActive(c))
      if (active) openStepKey.value = active.key
    }
  },
  { immediate: true, deep: true },
)

const isAgentStepsRunning = computed(() => {
  const s = String(props.jobStatus || '')
    .trim()
    .toLowerCase()
  if (s === 'pending' || s === 'running') return true
  for (const c of props.cards) {
    if (isStepActive(c)) return true
  }
  return false
})

const emit = defineEmits(['copy-agent-step-json', 'rich-interact'])

function onCopyJson(step, stepIdx) {
  emit('copy-agent-step-json', step, stepIdx)
}

function onRichInteract(payload) {
  emit('rich-interact', payload)
}

function onStepToggle({ key, open }) {
  if (open) {
    openStepKey.value = key
    return
  }
  if (openStepKey.value === key) {
    openStepKey.value = ''
  }
}
</script>
