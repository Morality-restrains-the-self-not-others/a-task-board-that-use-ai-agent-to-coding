<template>
      <!-- 过滤选项 -->
      <div class="mt-4">
        <!-- 实例过滤选项 -->
        <div class="mb-2">
          <h4 class="text-sm font-medium text-gray-500 mb-2">实例</h4>
          <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-2">
            <!-- IoOptimized -->
            <div class="flex-shrink-0">
              <label class="block text-xs font-medium text-gray-500 mb-1">IoOptimized</label>
              <input type="checkbox" id="io-optimized" v-model="panel.filterOptions.io_optimized" class="mt-1" checked>
            </div>
            <!-- 按量付费实例的竞价策略 -->
            <div class="flex-shrink-0 min-w-[120px]">
              <label for="spot-strategy" class="block text-xs font-medium text-gray-500 mb-1">竞价策略</label>
              <select id="spot-strategy" v-model="panel.filterOptions.spot_strategy" class="w-full px-2 py-1 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary">
                <option value="">-- 请选择 --</option>
                <option value="NoSpot">正常按量付费</option>
                <option value="SpotWithPriceLimit">设置上限价格</option>
                <option value="SpotAsPriceGo">系统自动出价</option>
              </select>
            </div>
            <!-- CPU 架构（只读展示，与已选镜像一致） -->
            <div class="flex-shrink-0 min-w-[140px]">
              <label for="image-architecture-display" class="block text-xs font-medium text-gray-500 mb-1">CPU 架构</label>
              <div
                id="image-architecture-display"
                data-testid="image-architecture-display"
                class="w-full px-2 py-1 border border-gray-200 rounded-md text-xs bg-gray-50 text-gray-800 min-h-[26px] flex items-center"
                :title="panel.imageArchitectureDisplayHint"
              >
                {{ panel.imageArchitectureDisplayLabel }}
              </div>
            </div>
          </div>
          <!-- 硬件配置过滤（CPU/内存/磁盘类型） -->
          <div class="mt-2">
            <HardwareConfigFilterFields
              v-model:cpu-cores="panel.filterOptions.cores"
              v-model:memory="panel.filterOptions.memory"
              v-model:system-disk-category="panel.filterOptions.system_disk_category"
              v-model:data-disk-category="panel.filterOptions.data_disk_category"
              :disk-options="extendedDiskOptions"
              cpu-label="CPU核心数"
              memory-label="内存(GB)"
              cpu-placeholder="核心数"
              memory-placeholder="内存大小"
            />
          </div>
        </div>

      </div>
      <!-- 可用实例列表 -->
</template>

<script setup>
import { computed } from 'vue'
import HardwareConfigFilterFields from '../cloud/HardwareConfigFilterFields.vue'

defineProps({
  panel: { type: Object, required: true },
})

const extendedDiskOptions = [
  { value: 'cloud', label: '普通云盘' },
  { value: 'cloud_efficiency', label: '高效云盘' },
  { value: 'cloud_ssd', label: 'SSD 云盘' },
  { value: 'ephemeral_ssd', label: '本地 SSD 盘' },
  { value: 'cloud_essd', label: 'ESSD 云盘' },
  { value: 'cloud_auto', label: 'ESSD AutoPL 云盘' },
  { value: 'cloud_essd_entry', label: 'ESSD Entry 云盘' },
]
</script>
