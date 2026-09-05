<template>
  <div class="mt-3" data-testid="layer-files-tabs">
    <div
      class="flex flex-wrap gap-1 border-b border-gray-200"
      role="tablist"
      aria-label="层文件视图"
      data-testid="layer-files-tablist"
    >
      <button
        type="button"
        role="tab"
        class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
        :class="tabClass(activeTab === 'tree')"
        :aria-selected="activeTab === 'tree' ? 'true' : 'false'"
        data-testid="layer-files-tab-tree"
        @click="activeTab = 'tree'"
      >项目文件树</button>
      <button
        type="button"
        role="tab"
        class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
        :class="tabClass(activeTab === 'changes')"
        :aria-selected="activeTab === 'changes' ? 'true' : 'false'"
        data-testid="layer-files-tab-changes"
        @click="activeTab = 'changes'"
      >文件变动<template v-if="changesCount > 0"> · {{ changesCount }}</template></button>
    </div>
    <!-- v-show：切 Tab 不卸载，树展开/预览与变动选中可恢复。Anti-Replay-OK: 只读 Tab，无 HTTP -->
    <div
      v-show="activeTab === 'tree'"
      class="pt-1"
      role="tabpanel"
      data-testid="layer-files-panel-tree"
    >
      <slot name="tree" />
    </div>
    <div
      v-show="activeTab === 'changes'"
      class="pt-1"
      role="tabpanel"
      data-testid="layer-files-panel-changes"
    >
      <slot name="changes" />
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  layerId: { type: String, default: '' },
  changesCount: { type: Number, default: 0 },
})

const activeTab = ref('changes')

function tabClass(selected) {
  return selected
    ? 'bg-sky-50 border-sky-300 text-sky-900 font-medium'
    : 'bg-white border-gray-200 text-gray-600 hover:bg-gray-50'
}

watch(
  () => String(props.layerId || '').trim(),
  (lid, prev) => {
    if (prev != null && lid !== prev) activeTab.value = 'changes'
  },
)
</script>
