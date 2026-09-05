<template>
  <div
    class="flex flex-wrap gap-1 border-b border-gray-200 pt-1.5"
    role="tablist"
    aria-label="评论执行面板"
    data-testid="comment-execution-tablist"
  >
    <button
      v-if="layerZtreeTab"
      type="button"
      role="tab"
      class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
      :class="tabClass(activeTab === 'ztree')"
      :aria-selected="activeTab === 'ztree' ? 'true' : 'false'"
      data-testid="comment-execution-tab-ztree"
      @click="emit('pick', 'ztree')"
    >任务关联</button>
    <button
      type="button"
      role="tab"
      class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
      :class="tabClass(activeTab === 'details')"
      :aria-selected="activeTab === 'details' ? 'true' : 'false'"
      data-testid="comment-execution-tab-details"
      @click="emit('pick', 'details')"
    >执行细节</button>
    <button
      v-if="showPredecessorList"
      type="button"
      role="tab"
      class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
      :class="predecessorClass"
      :aria-selected="activeTab === 'predecessors' ? 'true' : 'false'"
      data-testid="comment-execution-tab-predecessors"
      @click="emit('pick', 'predecessors')"
    >前序评论<template v-if="predecessorTabCount"> · {{ predecessorTabCount }}</template></button>
    <button
      v-if="serverRuntimeStatusTab"
      type="button"
      role="tab"
      class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
      :class="tabClass(activeTab === 'serverRuntime')"
      :aria-selected="activeTab === 'serverRuntime' ? 'true' : 'false'"
      data-testid="comment-execution-tab-server-runtime"
      @click="emit('pick', 'serverRuntime')"
    >服务器运行状态</button>
    <button
      v-if="serverRuntimeStatusTab"
      type="button"
      role="tab"
      class="px-2.5 py-1 text-xs rounded-t-md border border-b-0 transition-colors"
      :class="tabClass(activeTab === 'serverContent')"
      :aria-selected="activeTab === 'serverContent' ? 'true' : 'false'"
      data-testid="comment-execution-tab-server-content"
      @click="emit('pick', 'serverContent')"
    >服务器内容</button>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  activeTab: { type: String, required: true },
  layerZtreeTab: { type: Boolean, default: false },
  serverRuntimeStatusTab: { type: Boolean, default: false },
  showPredecessorList: { type: Boolean, default: false },
  predecessorTabCount: { type: Number, default: 0 },
})

const emit = defineEmits(['pick'])

function tabClass(selected) {
  return selected
    ? 'bg-sky-50 border-sky-300 text-sky-900 font-medium'
    : 'bg-white border-gray-200 text-gray-600 hover:bg-gray-50'
}

const predecessorClass = computed(() => (
  props.activeTab === 'predecessors'
    ? 'bg-amber-50 border-amber-300 text-amber-900 font-medium'
    : 'bg-white border-gray-200 text-gray-600 hover:bg-gray-50'
))
</script>
