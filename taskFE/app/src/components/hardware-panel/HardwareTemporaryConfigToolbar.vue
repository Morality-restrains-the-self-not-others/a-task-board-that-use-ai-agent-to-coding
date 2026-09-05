<template>
      <div
        v-if="!panel.runTemplateMode && panel.temporaryConfigExpanded && panel.hasConfiguredProjectRunTemplate()"
        class="mb-3 flex flex-wrap items-center justify-between gap-2 rounded-md border border-amber-100 bg-amber-50 px-3 py-2 text-xs text-amber-900"
        data-testid="temporary-config-toolbar"
      >
        <p class="m-0 text-[11px] leading-snug">
          以下为本次启动的临时硬件配置，不会保存到项目。
        </p>
        <button
          type="button"
          data-testid="restore-project-template-defaults-btn"
          class="shrink-0 px-3 py-1 border border-amber-300 text-amber-900 rounded-md text-xs font-medium hover:bg-amber-100 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-amber-400"
          @click="panel.restoreProjectTemplateDefaults"
        >
          恢复为项目模版默认值
        </button>
      </div>
      <!-- 网络配置选择：云平台、地域、VPC、安全组、可用区 -->
</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
</script>
