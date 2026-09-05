<template>
  <li class="text-xs">
    <div v-if="node.type === 'dir'" class="ml-1">
      <div class="flex items-start gap-0.5 rounded px-0.5 py-0.5 hover:bg-gray-50">
        <button
          type="button"
          class="mt-0.5 shrink-0 w-4 text-center text-gray-500 hover:text-gray-800"
          :aria-expanded="expanded"
          aria-label="展开或折叠子目录"
          data-testid="project-file-tree-toggle-dir"
          @click="emit('toggle-dir', node.path)"
        >
          {{ expanded ? '▼' : '▶' }}
        </button>
        <button
          type="button"
          class="flex-1 min-w-0 text-left rounded px-1 py-0.5 text-gray-700 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-primary/40"
          :class="selectedPath === node.path ? 'bg-blue-50 ring-1 ring-blue-200' : ''"
          @click="emit('select-dir', node.path)"
        >
          <span class="inline-flex items-start gap-1 min-w-0">
            <TaskDetailFileTypeIcon class="mt-0.5" type="folder" :path="node.path || ''" />
            <span class="font-mono break-all">{{ node.name }}</span>
          </span>
        </button>
      </div>
      <ul v-if="expanded && node.children?.length" class="mt-1 ml-3 space-y-0.5 border-l border-gray-200 pl-2">
        <TaskDetailProjectFileTreeNode
          v-for="child in node.children"
          :key="child.path"
          :node="child"
          :selected-path="selectedPath"
          :expanded-paths="expandedPaths"
          @select-file="onSelectFile"
          @select-dir="onSelectDir"
          @toggle-dir="onToggleDir"
        />
      </ul>
    </div>
    <button
      v-else
      type="button"
      class="w-full text-left rounded px-1 py-0.5 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-primary/40"
      :class="selectedPath === node.path ? 'bg-blue-50 ring-1 ring-blue-200' : ''"
      @click="emit('select-file', node.path)"
    >
      <span class="inline-flex items-start gap-1 min-w-0">
        <TaskDetailFileTypeIcon class="mt-0.5" :path="node.path || node.name || ''" />
        <span class="font-mono text-gray-800 break-all">{{ node.name }}</span>
      </span>
    </button>
  </li>
</template>

<script setup>
import { computed } from 'vue'
import TaskDetailFileTypeIcon from './TaskDetailFileTypeIcon.vue'

const props = defineProps({
  node: {
    type: Object,
    required: true,
  },
  selectedPath: { type: String, default: '' },
  /** 父组件托管的已展开目录 path 集合（跨层缓存后可恢复） */
  expandedPaths: { type: Object, required: true },
})

const emit = defineEmits(['select-file', 'select-dir', 'toggle-dir'])

const expanded = computed(() => {
  const paths = props.expandedPaths
  if (!(paths instanceof Set)) return false
  return paths.has(props.node.path)
})

function onSelectFile(path) {
  emit('select-file', path)
}

function onSelectDir(path) {
  emit('select-dir', path)
}

function onToggleDir(path) {
  emit('toggle-dir', path)
}
</script>
