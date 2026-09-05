<template>
  <div>
    <SecureInteractiveRichTextFrame
      v-if="bodyMode === 'html' && richHtml"
      :html="richHtml"
      title="代理步骤富文本"
      data-testid="agent-step-rich-frame"
      @interactive="emit('rich-interact', $event)"
    />
    <SafeMarkdownBlock
      v-else-if="bodyMode === 'markdown' && markdownSource"
      :source="markdownSource"
      data-testid="agent-step-markdown"
    />
    <details v-if="toolResultText" class="mt-1.5">
      <summary class="cursor-pointer text-[11px] font-medium text-gray-600">
        tool_results.result（点击展开）
      </summary>
      <pre
        class="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-words bg-white border border-gray-200 rounded p-2 text-gray-800 text-[11px] leading-snug"
      >{{ toolResultText }}</pre>
    </details>
    <p v-if="timestamp" class="mt-1 text-[11px] text-gray-500">
      {{ timestamp }}
    </p>
    <p v-if="error" class="mt-1 text-[11px] text-red-700 whitespace-pre-wrap break-words">
      {{ error }}
    </p>
    <details v-if="jsonPretty" class="mt-1.5">
      <summary class="cursor-pointer text-gray-600 font-medium">本步完整数据（JSON）</summary>
      <pre
        class="mt-1 max-h-48 overflow-auto whitespace-pre-wrap break-words bg-white border border-gray-200 rounded p-2 text-gray-800 text-[11px] leading-snug"
      >{{ jsonPretty }}</pre>
    </details>
  </div>
</template>

<script setup>
import SecureInteractiveRichTextFrame from '../secure-rich-text/SecureInteractiveRichTextFrame.vue'
import SafeMarkdownBlock from '../secure-rich-text/SafeMarkdownBlock.vue'

defineProps({
  bodyMode: { type: String, default: 'text' },
  richHtml: { type: String, default: '' },
  markdownSource: { type: String, default: '' },
  toolResultText: { type: String, default: '' },
  timestamp: { type: String, default: '' },
  error: { type: String, default: '' },
  jsonPretty: { type: String, default: '' },
})

const emit = defineEmits(['rich-interact'])
</script>
