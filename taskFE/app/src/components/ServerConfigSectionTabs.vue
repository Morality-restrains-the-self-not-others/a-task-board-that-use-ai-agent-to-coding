<template>
  <div class="server-section-tabs flex items-center border-b border-gray-200">
    <div class="flex items-center min-w-0 overflow-x-auto">
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap"
        :class="tabClass('runtime')"
        @click="emit('select', 'runtime')"
      >
        服务器运行状态
      </button>
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap"
        :class="tabClass('content')"
        @click="emit('select-content')"
      >
        服务器内容
      </button>
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap"
        :class="tabClass('history')"
        @click="emit('select-history')"
      >
        历史服务器启动记录
      </button>
      <button
        v-if="isRelayToTraeEnabled"
        type="button"
        class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap"
        :class="tabClass('relayDirect')"
        data-testid="server-config-relay-direct-tab"
        @click="emit('select', 'relayDirect')"
      >
        直接启动
      </button>
    </div>
    <button
      type="button"
      data-testid="server-section-collapse-toggle"
      class="ml-auto shrink-0 inline-flex items-center gap-1 px-2 py-1 mr-1 border border-gray-300 rounded-md text-xs text-gray-600 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-primary"
      :aria-expanded="bodyExpanded ? 'true' : 'false'"
      :aria-label="bodyExpanded ? '收起服务器信息内容区' : '展开服务器信息内容区'"
      @click="emit('toggle-body')"
    >
      <span>{{ bodyExpanded ? '收起' : '展开' }}</span>
      <svg
        class="w-3.5 h-3.5 transition-transform duration-200"
        :class="{ 'rotate-180': bodyExpanded }"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
        aria-hidden="true"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </button>
  </div>
</template>

<script setup>
const props = defineProps({
  activeServerSection: { type: String, required: true },
  isRelayToTraeEnabled: { type: Boolean, default: false },
  bodyExpanded: { type: Boolean, default: false },
})

const emit = defineEmits([
  'select',
  'select-content',
  'select-history',
  'toggle-body',
])

function tabClass(section) {
  return props.activeServerSection === section
    ? 'border-primary text-primary'
    : 'border-transparent text-gray-500 hover:text-gray-700'
}
</script>
