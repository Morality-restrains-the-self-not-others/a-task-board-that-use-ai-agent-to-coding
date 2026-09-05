<template>
      <!-- 价格信息 -->
      <div v-if="panel.selectedInstance" class="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
        <div class="flex justify-between items-start mb-1.5">
          <h4 class="text-xs font-medium text-blue-800">选中实例价格信息</h4>
          <span class="text-[10px] text-gray-500">价格仅供参考，实际以云厂商为准</span>
        </div>
        <div v-if="panel.priceLoading[panel.selectedInstance.instance_type]" class="text-gray-400 text-xs">
          实例类型: {{ panel.selectedInstance.instance_type }}
          <div class="mt-1 flex items-center">
            <span class="inline-block animate-spin h-3 w-3 border-2 border-gray-300 border-t-primary rounded-full mr-1.5"></span>
            价格: 获取中...
          </div>
        </div>
        <div v-else-if="panel.selectedInstancePriceView" class="text-green-600 font-medium text-xs">
          实例类型: {{ panel.selectedInstance.instance_type }}
          <div class="mt-1">价格: {{ panel.formatPriceAmount(panel.selectedInstancePriceView.currency, panel.selectedInstancePriceView.tradePrice) }}</div>
          <div
            v-if="panel.selectedInstancePriceView.originalPrice != null && panel.selectedInstancePriceView.originalPrice !== panel.selectedInstancePriceView.tradePrice"
            class="mt-0.5 text-[10px] text-gray-500 line-through"
          >
            原价: {{ panel.formatPriceAmount(panel.selectedInstancePriceView.currency, panel.selectedInstancePriceView.originalPrice) }}
          </div>
          <div
            v-if="panel.selectedInstancePriceView.discountPrice != null && panel.selectedInstancePriceView.discountPrice > 0"
            class="mt-0.5 text-[10px] text-orange-600"
          >
            折扣: -{{ panel.formatPriceAmount(panel.selectedInstancePriceView.currency, panel.selectedInstancePriceView.discountPrice) }}
          </div>
          <div v-if="panel.selectedInstancePriceView.details.length" class="mt-1.5 text-[10px] text-gray-600 space-y-0.5">
            <div v-for="(detail, index) in panel.selectedInstancePriceView.details" :key="index" class="flex justify-between gap-2">
              <span>{{ panel.getResourceName(detail.resource) }}:</span>
              <span>{{ panel.formatPriceAmount(panel.selectedInstancePriceView.currency, detail.amount) }}</span>
            </div>
          </div>
        </div>
        <div v-else class="text-gray-400 text-xs">
          价格: 获取中...
        </div>
      </div>
</template>

<script setup>
defineProps({
  panel: { type: Object, required: true },
})
</script>
