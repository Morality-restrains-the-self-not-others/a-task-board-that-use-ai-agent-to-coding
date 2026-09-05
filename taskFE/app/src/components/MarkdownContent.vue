<template>
  <div v-if="renderedHtml" data-alias="cmp-markdown-content" class="markdown-content max-w-none" v-html="renderedHtml" />
  <p v-else class="text-gray-700">{{ emptyText }}</p>
</template>

<script setup>
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const md = new MarkdownIt({
  breaks: true,
  linkify: true,
})

const props = defineProps({
  content: { type: String, default: '' },
  emptyText: { type: String, default: '暂无描述' },
})

const renderedHtml = computed(() => {
  if (!props.content || !props.content.trim()) return ''
  const rawHtml = md.render(props.content)
  return DOMPurify.sanitize(rawHtml)
})
</script>

<style scoped>
.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3) {
  font-weight: 600;
  margin-top: 1em;
  margin-bottom: 0.5em;
}
.markdown-content :deep(h1) { font-size: 1.5em; }
.markdown-content :deep(h2) { font-size: 1.25em; }
.markdown-content :deep(h3) { font-size: 1.125em; }
.markdown-content :deep(p) { margin-bottom: 0.75em; }
.markdown-content :deep(ul),
.markdown-content :deep(ol) { padding-left: 1.5em; margin-bottom: 0.75em; }
.markdown-content :deep(li) { margin-bottom: 0.25em; }
.markdown-content :deep(code) {
  background: #f1f5f9;
  padding: 0.125em 0.375em;
  border-radius: 0.25em;
  font-size: 0.875em;
  font-family: ui-monospace, monospace;
}
.markdown-content :deep(pre) {
  background: #1e293b;
  color: #e2e8f0;
  padding: 1em;
  border-radius: 0.5em;
  overflow-x: auto;
  margin-bottom: 0.75em;
}
.markdown-content :deep(pre code) {
  background: none;
  padding: 0;
  color: inherit;
}
.markdown-content :deep(blockquote) {
  border-left: 3px solid #d1d5db;
  padding-left: 1em;
  color: #6b7280;
  margin-bottom: 0.75em;
}
.markdown-content :deep(a) {
  color: #3b82f6;
  text-decoration: underline;
}
.markdown-content :deep(table) {
  border-collapse: collapse;
  margin-bottom: 0.75em;
  width: 100%;
}
.markdown-content :deep(th),
.markdown-content :deep(td) {
  border: 1px solid #d1d5db;
  padding: 0.5em;
  text-align: left;
}
.markdown-content :deep(th) { background: #f9fafb; font-weight: 600; }
.markdown-content :deep(hr) { margin: 1em 0; border-color: #e5e7eb; }
.markdown-content :deep(img) { max-width: 100%; height: auto; }
</style>
