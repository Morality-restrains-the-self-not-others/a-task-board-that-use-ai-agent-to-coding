<template>
  <div class="mt-12 p-6 bg-white rounded-xl shadow-md">
    <div class="flex justify-between items-center mb-4">
      <h3 class="text-xl font-bold text-text">交付物体系列表</h3>
      <button class="btn-primary" @click="$emit('createDeliverableSystem')">创建交付物体系</button>
    </div>
    
    <!-- 加载状态 -->
    <div v-if="loadingDeliverableSystems" class="flex justify-center items-center py-10">
      <div class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-primary"></div>
      <span class="ml-3 text-gray-600">加载交付物体系列表中...</span>
    </div>
    
    <!-- 交付物体系列表 -->
    <div v-else>
      <!-- 空状态 -->
      <div v-if="deliverableSystems.length === 0" class="flex flex-col items-center justify-center py-10">
        <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"></path>
        </svg>
        <h4 class="text-lg font-medium text-gray-900 mb-1">暂无交付物体系</h4>
        <p class="text-gray-500 mb-6">系统中还没有任何交付物体系，请点击"创建交付物体系"按钮创建第一个交付物体系。</p>
      </div>
      
      <!-- 交付物体系列表 -->
      <div v-else class="border border-gray-200 rounded-lg overflow-hidden">
        <!-- 表头 -->
        <div class="grid grid-cols-4 gap-2 p-2 bg-gray-50 border-b border-gray-200 text-xs font-medium text-gray-500">
          <div>交付物体系名称</div>
          <div class="truncate">层级结构</div>
          <div class="truncate">描述</div>
          <div class="text-right">类型</div>
        </div>
        <!-- 列表内容 -->
        <div>
          <div v-for="(deliverableSystem, index) in deliverableSystems" :key="deliverableSystem.id" class="grid grid-cols-4 gap-2 p-2 hover:bg-gray-50 transition-colors" :class="index > 0 ? 'border-t border-gray-200' : ''" style="max-height: 44px; align-items: center;">
            <div class="min-w-0">
              <h4 class="text-sm font-semibold text-gray-900 truncate">{{ deliverableSystem.name }}</h4>
            </div>
            <div class="min-w-0">
              <p class="text-xs text-gray-600 truncate">{{ deliverableSystem.level_structure || deliverableSystem.level_names?.join(' > ') || '-' }}</p>
            </div>
            <div class="min-w-0">
              <p class="text-xs text-gray-600 truncate">{{ deliverableSystem.description || '-' }}</p>
            </div>
            <div class="flex justify-end items-center space-x-2">
              <span class="px-2 py-1 text-xs rounded-full whitespace-nowrap" :class="deliverableSystem.is_system ? 'bg-blue-100 text-blue-800' : 'bg-green-100 text-green-800'">
                {{ deliverableSystem.is_system ? '系统' : '公司' }}
              </span>
              <!-- 仅为公司类型的交付物体系显示操作按钮 -->
              <div v-if="!deliverableSystem.is_system" class="flex space-x-1">
                <button class="px-2 py-1 text-xs text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded transition-colors" @click="$emit('editDeliverableSystem', deliverableSystem)">
                  编辑
                </button>
                <button class="px-2 py-1 text-xs text-red-600 hover:text-red-800 hover:bg-red-50 rounded transition-colors" @click="$emit('deleteDeliverableSystem', deliverableSystem)">
                  删除
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'

const props = defineProps({
  tenantId: {
    type: [String, Number],
    default: ''
  },
  deliverableSystems: {
    type: Array,
    default: () => []
  },
  loadingDeliverableSystems: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['createDeliverableSystem', 'editDeliverableSystem', 'deleteDeliverableSystem'])
</script>

<style scoped>
</style>