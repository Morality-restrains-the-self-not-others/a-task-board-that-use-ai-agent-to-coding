<template>
  <!-- Teleport-OK: overflow-clip — 投递错误气泡在表格滚动容器内会被裁剪 -->
  <Teleport to="body">
    <div
      v-if="show"
      class="delivery-error-popover"
      :style="{ top: y + 'px', left: x + 'px' }"
    >{{ content }}</div>
  </Teleport>
</template>

<script setup>
defineProps({
  show: { type: Boolean, default: false },
  content: { type: String, default: '' },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
})
</script>

<style>
.delivery-error-popover {
  position: absolute;
  z-index: 9999;
  max-width: 360px;
  padding: 8px 12px;
  background: rgba(30, 30, 30, 0.92);
  color: #f1f5f9;
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace;
  line-height: 1.5;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
  pointer-events: none;
  white-space: pre-line;
  word-break: break-word;
}
</style>
