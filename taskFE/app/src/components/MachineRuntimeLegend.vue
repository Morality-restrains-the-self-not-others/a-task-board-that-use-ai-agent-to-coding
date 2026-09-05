<template>
  <span
    data-testid="machine-runtime-legend"
    class="inline-flex items-center gap-2 text-xs text-gray-500 align-middle"
    aria-label="图例：空心环表示机器节点运行中，实心点表示容器运行中"
  >
    <span class="inline-flex items-center gap-1" :title="RUNTIME_MACHINE_TIP">
      <span
        class="inline-block w-3 h-3 rounded-full border-2 border-emerald-500 shrink-0"
        aria-hidden="true"
      />
      <span>空心=已启动机器</span>
    </span>
    <span class="inline-flex items-center gap-1" :title="RUNTIME_CONTAINER_TIP">
      <span
        class="inline-block w-2 h-2 rounded-full bg-emerald-500 shrink-0"
        aria-hidden="true"
      />
      <span>实心=已启动容器</span>
    </span>
  </span>
</template>

<script setup>
import {
  RUNTIME_MACHINE_TIP,
  RUNTIME_CONTAINER_TIP,
} from '../utils/workPanelRuntimeIndicators.js'
</script>
