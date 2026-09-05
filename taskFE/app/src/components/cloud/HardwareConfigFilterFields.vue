<template>
  <div class="hardware-config-filter-fields">
    <div class="grid grid-cols-2 gap-3">
      <!-- CPU 核心数 -->
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">{{ cpuLabel }}</label>
        <input
          :value="cpuCores"
          type="number"
          class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          :placeholder="cpuPlaceholder"
          min="1"
          @input="$emit('update:cpuCores', $event.target.value)"
          @change="$emit('filter-change')"
        >
      </div>
      <!-- 内存 -->
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">{{ memoryLabel }}</label>
        <input
          :value="memory"
          type="number"
          class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          :placeholder="memoryPlaceholder"
          min="1"
          @input="$emit('update:memory', $event.target.value)"
          @change="$emit('filter-change')"
        >
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3 mt-3">
      <!-- 系统盘类型 -->
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">系统盘类型</label>
        <select
          :value="systemDiskCategory"
          class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          @change="$emit('update:systemDiskCategory', $event.target.value); $emit('filter-change')"
        >
          <option
            v-for="opt in diskOptions"
            :key="opt.value"
            :value="opt.value"
          >{{ opt.label }}</option>
        </select>
      </div>
      <!-- 数据盘类型（空值 = 不要数据盘；后端 data_disk_category 为空则不挂载） -->
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">数据盘类型</label>
        <select
          :value="dataDiskCategory"
          data-testid="data-disk-category-select"
          class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
          @change="$emit('update:dataDiskCategory', $event.target.value); $emit('filter-change')"
        >
          <option value="">不要数据盘</option>
          <option
            v-for="opt in diskOptions"
            :key="opt.value"
            :value="opt.value"
          >{{ opt.label }}</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  cpuCores: { type: [String, Number], default: '' },
  memory: { type: [String, Number], default: '' },
  systemDiskCategory: { type: String, default: 'cloud_essd' },
  dataDiskCategory: { type: String, default: 'cloud_essd' },
  diskOptions: {
    type: Array,
    default: () => [
      { value: 'cloud_efficiency', label: '高效云盘' },
      { value: 'cloud_ssd', label: 'SSD 云盘' },
      { value: 'cloud_essd', label: 'ESSD 云盘' },
      { value: 'cloud_auto', label: 'ESSD AutoPL 云盘' },
      { value: 'cloud_essd_entry', label: 'ESSD Entry 云盘' },
    ],
  },
  cpuLabel: { type: String, default: 'CPU 核心数' },
  memoryLabel: { type: String, default: '内存 (GB)' },
  cpuPlaceholder: { type: String, default: '如 2' },
  memoryPlaceholder: { type: String, default: '如 4' },
})

defineEmits([
  'update:cpuCores',
  'update:memory',
  'update:systemDiskCategory',
  'update:dataDiskCategory',
  'filter-change',
])
</script>
