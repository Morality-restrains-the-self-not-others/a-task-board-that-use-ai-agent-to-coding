<template>
  <div class="layer-graph-ztree rounded-md border border-gray-200 bg-white p-2 text-sm">
    <p v-if="metaLine" class="text-xs text-gray-500 mb-2 px-1">{{ metaLine }}</p>
    <ul class="ztree-mimic m-0 list-none p-0 pl-0">
      <LayerGraphZtreeNode
        v-for="node in roots"
        :key="String(node.id)"
        :node="node"
        :depth="0"
        :selected-id="selectedId"
        :action-busy-key="actionBusyKey"
        @node-select="(n) => emit('node-select', n)"
        @job-redo="(id) => emit('job-redo', id)"
        @job-interrupt="(id) => emit('job-interrupt', id)"
        @job-continue="(id) => emit('job-continue', id)"
        @job-edit-run="(n) => emit('job-edit-run', n)"
        @job-delete="(id) => emit('job-delete', id)"
        @layer-delete="(id) => emit('layer-delete', id)"
        @layer-submit="(n) => emit('layer-submit', n)"
        @layer-push="(n) => emit('layer-push', n)"
        @layer-merge="(n) => emit('layer-merge', n)"
        @layer-submit-and-push="(n) => emit('layer-submit-and-push', n)"
        @layer-submit-and-merge="(n) => emit('layer-submit-and-merge', n)"
      />
    </ul>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { simpleDataToTreeRoots } from '../utils/layerZtreeNodes.js'
import LayerGraphZtreeNode from './LayerGraphZtreeNode.vue'

const props = defineProps({
  flatNodes: {
    type: Array,
    default: () => [],
  },
  metaLine: {
    type: String,
    default: '',
  },
  selectedId: {
    type: [String, Number],
    default: null,
  },
  actionBusyKey: {
    type: String,
    default: '',
  },
})

const emit = defineEmits([
  'node-select',
  'job-redo',
  'job-interrupt',
  'job-continue',
  'job-edit-run',
  'job-delete',
  'layer-delete',
  'layer-submit',
  'layer-push',
  'layer-merge',
  'layer-submit-and-push',
  'layer-submit-and-merge',
])

const roots = computed(() => {
  const list = simpleDataToTreeRoots(props.flatNodes || [])
  if (list.length === 1 && list[0]?.nodeKind === 'virtual') {
    return list[0].children || []
  }
  return list
})
</script>

<style scoped>
.ztree-mimic {
  font-size: 12px;
  line-height: 1.5;
}
</style>
