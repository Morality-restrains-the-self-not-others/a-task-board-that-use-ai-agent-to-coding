<template>
  <div
    v-if="visible"
    class="p-4 rounded-lg border border-amber-200 bg-amber-50 text-amber-950"
    role="status"
    data-testid="container-http-unreachable-banner"
  >
    <p class="text-sm font-medium">容器当前无法连接</p>
    <p class="mt-1 text-xs text-amber-900/90">
      平台已记录业务端点，但转发容器的 HTTP 请求失败。发往容器的指令、执行日志拉取与部分操作将不可用；容器恢复后我们会自动探测并恢复为可连接状态。
    </p>
  </div>
</template>

<script setup>
defineProps({
  visible: { type: Boolean, required: true },
})
</script>
