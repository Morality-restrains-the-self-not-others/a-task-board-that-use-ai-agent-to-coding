<template>
  <summary
    class="cursor-pointer select-none list-none [&::-webkit-details-marker]:hidden flex flex-col gap-1"
    data-testid="layer-agent-step-card-summary"
  >
    <div class="flex items-start justify-between gap-2 pb-1">
      <div class="flex items-start gap-1.5 min-w-0">
        <span
          class="inline-block shrink-0 text-gray-500 transition-transform duration-200 group-open/agent-step:rotate-90 mt-0.5"
          aria-hidden="true"
        >▶</span>
        <p class="min-w-0 font-medium text-gray-900 whitespace-pre-wrap break-words">{{ title }}</p>
      </div>
      <div class="shrink-0 flex items-center gap-1">
        <span
          v-if="modelBadge"
          class="inline-flex items-center rounded-full bg-indigo-50 px-2 py-0.5 text-[11px] text-indigo-700 border border-indigo-100"
        >
          {{ modelBadge }}
        </span>
        <span
          v-if="usageBadge"
          class="inline-flex items-center rounded-full bg-cyan-50 px-2 py-0.5 text-[11px] text-cyan-700 border border-cyan-100"
        >
          {{ usageBadge }}
        </span>
        <!-- Anti-Replay-OK: clipboard copy of already-rendered step JSON; no HTTP write -->
        <button
          type="button"
          class="text-[11px] px-2 py-0.5 rounded border border-gray-300 bg-white text-gray-700 hover:bg-gray-50"
          @click.stop.prevent="emit('copy-json')"
        >
          {{ isCopied ? '已复制JSON' : '复制JSON' }}
        </button>
      </div>
    </div>
    <pre
      v-if="plainSubtitle"
      class="mt-1 max-h-48 overflow-auto whitespace-pre-wrap break-words rounded border border-indigo-100 bg-indigo-50/40 p-2 text-[11px] leading-snug text-gray-700"
      data-testid="layer-agent-step-command-preview"
      @click.stop
    >{{ plainSubtitle }}</pre>
  </summary>
</template>

<script setup>
defineProps({
  title: { type: String, default: '' },
  modelBadge: { type: String, default: '' },
  usageBadge: { type: String, default: '' },
  isCopied: { type: Boolean, default: false },
  plainSubtitle: { type: String, default: '' },
})

const emit = defineEmits(['copy-json'])
</script>
