<template>
      <!-- 可用实例列表 -->
      <div class="mt-4">
        <div class="flex justify-between items-center mb-3">
          <div class="flex items-center space-x-4">
            <h4 class="text-sm font-medium text-gray-500">可用实例列表 <span class="text-xs text-gray-400">(共 {{ panel.availableInstances.length }} 行)</span></h4>
            <div v-if="panel.sdkMethods.length > 0" class="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
              SDK方法: 
              <span v-for="(method, index) in panel.sdkMethods" :key="index" class="text-primary">
                {{ method.method }}
                <span v-if="index < panel.sdkMethods.length - 1">, </span>
              </span>
            </div>
            <div class="text-sm text-gray-500">
              第 {{ panel.currentPage }} / {{ panel.totalPages }} 页
            </div>
          </div>
          <div class="flex space-x-2">
            <button 
              @click="panel.prevPage" 
              class="px-3 py-1 bg-gray-200 text-gray-700 rounded text-sm hover:bg-gray-300"
              :disabled="panel.currentPage === 1"
            >
              上一页
            </button>
            <button 
              v-if="panel.nextToken || panel.currentPage < panel.totalPages" 
              @click="panel.loadNextPage" 
              class="px-3 py-1 bg-primary text-white rounded text-sm hover:bg-blue-600"
              :disabled="panel.isLoadingNextPage || (panel.currentPage === panel.totalPages && !panel.nextToken)"
            >
              <span v-if="panel.isLoadingNextPage" class="flex items-center">
                <span class="inline-block animate-spin h-4 w-4 border-2 border-white border-t-transparent rounded-full mr-2"></span>
                加载中...
              </span>
              <span v-else>下一页</span>
            </button>
          </div>
        </div>
        <div v-if="panel.isLoadingInstances" class="flex justify-center items-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
        <div v-else-if="panel.isLoadingNextPage" class="flex justify-center items-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        </div>
        <div v-else-if="panel.availableInstances.length === 0" class="text-center py-8 text-gray-500">
          暂无可用实例
        </div>
        <div v-else class="space-y-1">
          <div 
            v-for="instance in panel.availableInstances" 
            :key="instance.instance_type" 
            class="flex flex-col gap-1 px-2 py-1.5 border border-gray-200 rounded hover:bg-gray-50"
            v-show="!panel.selectedInstance || panel.selectedInstance.instance_type === instance.instance_type"
          >
            <div class="flex items-center justify-between gap-2 min-w-0">
              <span class="font-medium text-xs text-gray-900 truncate min-w-0" :title="instance.instance_type">{{ instance.instance_type }}</span>
              <button 
                type="button"
                @click="panel.selectInstance(instance)" 
                :class="['shrink-0 px-1.5 py-0.5 rounded text-[11px] leading-none', panel.selectedInstance && panel.selectedInstance.instance_type === instance.instance_type ? 'bg-gray-200 text-gray-600' : 'bg-primary text-white hover:bg-blue-600']"
              >
                {{ panel.selectedInstance && panel.selectedInstance.instance_type === instance.instance_type ? '已选' : '选择' }}
              </button>
            </div>
            <!-- 核心规格一行 -->
            <div class="text-[11px] text-gray-800 tabular-nums leading-tight">
              <span>{{ instance.cpu_cores }}核</span>
              <span class="text-gray-400 mx-1">·</span>
              <span>{{ instance.memory_gb }}GB</span>
              <template v-if="instance.gpu_cores">
                <span class="text-gray-400 mx-1">·</span>
                <span>GPU {{ instance.gpu_cores }}核</span>
              </template>
              <template v-if="instance.gpu_type">
                <span class="text-gray-400 mx-1">·</span>
                <span>{{ instance.gpu_type }}</span>
              </template>
            </div>
            <!-- 其余规格：紧凑换行 -->
            <div class="flex flex-wrap gap-x-2 gap-y-0.5 text-[11px] text-gray-600 leading-snug">
              <span v-if="instance.instance_type_family">家族 {{ instance.instance_type_family }}</span>
              <span v-if="instance.status">状态 {{ instance.status === 'available' ? '可用' : '不可用' }}</span>
              <span v-if="instance.instance_category">分类 {{ instance.instance_category }}</span>
              <span v-if="instance.local_disk_size">本地盘 {{ instance.local_disk_size }}GB</span>
              <span v-if="instance.network_performance">网络 {{ instance.network_performance }}</span>
              <span v-if="instance.max_bandwidth_out">出 {{ instance.max_bandwidth_out }}Mbps</span>
              <span v-if="instance.max_bandwidth_in">入 {{ instance.max_bandwidth_in }}Mbps</span>
              <span v-if="instance.max_pps">PPS {{ instance.max_pps }}</span>
              <span v-if="instance.architecture">架构 {{ instance.architecture }}</span>
              <span v-if="instance.cpu_type">{{ instance.cpu_type }}</span>
              <span v-if="instance.gpu_memory">显存 {{ instance.gpu_memory }}GB</span>
              <span v-if="instance.gpu_count">GPU×{{ instance.gpu_count }}</span>
            </div>
            
            <!-- 存储信息 -->
            <div class="mt-1 pt-1 border-t border-gray-100">
              <div v-if="instance.storage_amount || instance.storage_category" class="text-[11px] text-gray-600 flex flex-wrap gap-x-3 gap-y-0 leading-tight">
                <span v-if="instance.storage_amount">盘数 {{ instance.storage_amount }}</span>
                <span v-if="instance.storage_category">{{ instance.storage_category }}</span>
              </div>
              
              <!-- 存储配置 -->
              <div class="mt-1 space-y-1">
                <div v-if="instance.storage_types && instance.storage_types.length > 0" class="flex flex-wrap items-center gap-x-2 gap-y-1">
                  <span class="text-[10px] font-medium text-gray-500 shrink-0">存储类型</span>
                  <div class="flex flex-wrap gap-1">
                    <button 
                      v-for="type in instance.storage_types" 
                      :key="type"
                      type="button"
                      @click="panel.handleStorageTypeChange(instance, type)"
                      :class="['px-1.5 py-px text-[10px] border rounded hover:bg-gray-100 leading-tight', instance.storage_type === type ? 'bg-primary text-white border-primary' : 'border-gray-300']"
                    >
                      {{ type }}
                    </button>
                  </div>
                </div>
                <div class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                  <label class="text-[10px] font-medium text-gray-500 shrink-0" :for="'storage-size-' + instance.instance_type">大小(GB)</label>
                  <input type="number" 
                         :id="'storage-size-' + instance.instance_type" 
                         v-model="instance.storage_gb" 
                         @change="panel.handleStorageSizeChange(instance)"
                         class="w-[4.5rem] px-1 py-px border border-gray-300 rounded text-[11px] focus:outline-none focus:ring-primary focus:border-primary"
                         :min="panel.getStorageMin(instance, instance.storage_type)"
                         :max="panel.getStorageMax(instance, instance.storage_type)"
                         step="10"
                         placeholder="GB">
                  <span class="text-[10px] text-gray-500 tabular-nums">
                    {{ panel.getStorageMin(instance, instance.storage_type) }}–{{ panel.getStorageMax(instance, instance.storage_type) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
        
        <!-- 带宽设置 -->
        <div v-if="panel.selectedInstance" class="mt-4 p-3 bg-gray-50 border border-gray-200 rounded-lg">
          <h4 class="text-xs font-medium text-gray-500 mb-2">带宽设置</h4>
          <div class="flex flex-wrap gap-3">
            <div>
              <label class="block text-[10px] font-medium text-gray-500 mb-1">计费模式</label>
              <select 
                id="bandwidth-charging-mode-select"
                v-model="panel.selectedBandwidthChargingMode"
                class="w-28 px-1.5 py-0.5 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
              >
                <option value="PayByBandwidth">按固定带宽</option>
                <option value="PayByTraffic">按使用流量</option>
              </select>
            </div>
            <div>
              <label class="block text-[10px] font-medium text-gray-500 mb-1">带宽</label>
              <div class="flex items-center space-x-1.5">
                <input
                  id="bandwidth-input"
                  type="number"
                  :value="panel.selectedBandwidth[panel.selectedInstance.instance_type]"
                  class="w-28 px-1.5 py-0.5 border border-gray-300 rounded-md text-xs focus:outline-none focus:ring-primary focus:border-primary"
                  :min="panel.bandwidthLimitations[panel.selectedInstance.instance_type]?.min_bandwidth ?? 0"
                  :max="panel.bandwidthLimitations[panel.selectedInstance.instance_type]?.max_bandwidth ?? 100"
                  :placeholder="panel.bandwidthLoading[panel.selectedInstance.instance_type] ? '加载中...' : 'Mbps'"
                  :disabled="!panel.bandwidthLimitations[panel.selectedInstance.instance_type] || panel.bandwidthLoading[panel.selectedInstance.instance_type]"
                  @input="panel.onBandwidthInput(panel.selectedInstance.instance_type, $event)"
                  @blur="panel.onBandwidthBlur(panel.selectedInstance.instance_type)"
                />
                <span class="text-[10px] text-gray-500">Mbps</span>
                <button 
                  v-if="!panel.bandwidthLimitations[panel.selectedInstance.instance_type] && !panel.bandwidthLoading[panel.selectedInstance.instance_type]"
                  @click="panel.fetchBandwidthLimitation(panel.selectedInstance)"
                  class="px-1.5 py-0.5 bg-gray-200 text-gray-700 rounded text-[10px] hover:bg-gray-300"
                >
                  获取
                </button>
                <span v-else-if="panel.bandwidthLimitations[panel.selectedInstance.instance_type]" class="text-[10px] text-gray-500">
                  ({{ panel.bandwidthLimitations[panel.selectedInstance.instance_type].min_bandwidth }}-{{ panel.bandwidthLimitations[panel.selectedInstance.instance_type].max_bandwidth }}Mbps)
                </span>
                <span v-else class="text-[10px] text-gray-500">
                  (默认: {{ panel.DEFAULT_BANDWIDTH }}Mbps)
                </span>
              </div>
            </div>
          </div>
        </div>
        
        <!-- 分页控件 -->
        <div class="mt-3 flex justify-between items-center">
          <button 
            @click="panel.prevPage" 
            class="px-2 py-0.5 bg-gray-200 text-gray-700 rounded text-xs hover:bg-gray-300"
            :disabled="panel.currentPage === 1"
          >
            上一页
          </button>
          <div class="text-xs text-gray-500">
            第 {{ panel.currentPage }} / {{ panel.totalPages }} 页
          </div>
          <button 
            @click="panel.nextPage" 
            class="px-2 py-0.5 bg-primary text-white rounded text-xs hover:bg-blue-600"
            :disabled="(panel.currentPage === panel.totalPages && !panel.nextToken) || panel.isLoadingNextPage"
          >
            <span v-if="panel.isLoadingNextPage" class="flex items-center">
              <span class="inline-block animate-spin h-3 w-3 border-2 border-white border-t-transparent rounded-full mr-1"></span>
              加载中...
            </span>
            <span v-else>下一页</span>
          </button>
        </div>
      </div>

</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
</script>
