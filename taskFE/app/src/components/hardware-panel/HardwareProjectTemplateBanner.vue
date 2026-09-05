<template>
      <div
        v-if="panel.showProjectTemplateSummaryOnly"
        class="mb-3 rounded-md border border-blue-100 bg-blue-50 px-3 py-2 text-xs text-blue-900 space-y-1"
        data-testid="hardware-project-template-banner"
      >
        <p class="m-0 font-medium">{{ panel.projectTemplateBannerText }}</p>
        <p v-if="panel.projectTemplateHardwareSummary" class="m-0 text-[11px] text-blue-800" data-testid="hardware-project-template-spec-summary">
          {{ panel.projectTemplateHardwareSummary }}
        </p>
        <p v-if="panel.projectTemplatePriceHint" class="m-0 text-[11px] text-blue-800" data-testid="hardware-project-template-price-hint">
          {{ panel.projectTemplatePriceHint }}
        </p>
        <p class="m-0 text-[11px] text-blue-700">
          将使用上述项目模版启动服务器。如需本次单独调整，请点击「临时配置」。
        </p>
      </div>

</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
</script>
