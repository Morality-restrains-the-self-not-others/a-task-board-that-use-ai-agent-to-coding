<template>
  <div data-testid="admin-order-lookup-box">
    <div class="flex gap-1 mb-3" role="tablist" aria-label="订单查询方式" data-testid="admin-order-lookup-tabs">
      <button
        type="button"
        role="tab"
        class="px-3 py-1.5 text-sm rounded-md border"
        :class="activeTab === 'trade'
          ? 'bg-gray-800 text-white border-gray-800'
          : 'bg-white text-gray-700 border-gray-300 hover:border-gray-400'"
        :aria-selected="activeTab === 'trade' ? 'true' : 'false'"
        data-testid="admin-order-lookup-tab-trade"
        @click="activeTab = 'trade'"
      >
        交易单号
      </button>
      <button
        type="button"
        role="tab"
        class="px-3 py-1.5 text-sm rounded-md border"
        :class="activeTab === 'wechat'
          ? 'bg-gray-800 text-white border-gray-800'
          : 'bg-white text-gray-700 border-gray-300 hover:border-gray-400'"
        :aria-selected="activeTab === 'wechat' ? 'true' : 'false'"
        data-testid="admin-order-lookup-tab-wechat"
        @click="activeTab = 'wechat'"
      >
        微信关联账号
      </button>
    </div>
    <OrderNumberPasteJump
      v-if="activeTab === 'trade'"
      :searching="searching"
      @search="emit('search-trade', $event)"
    />
    <WechatLinkedAccountQuery
      v-else
      :searching="searching"
      @search="emit('search-wechat', $event)"
    />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import OrderNumberPasteJump from './OrderNumberPasteJump.vue'
import WechatLinkedAccountQuery from './WechatLinkedAccountQuery.vue'

const props = defineProps({
  searching: { type: Boolean, default: false },
  initialTab: { type: String, default: 'trade' },
})

const emit = defineEmits(['search-trade', 'search-wechat'])

const activeTab = ref(props.initialTab === 'wechat' ? 'wechat' : 'trade')

watch(
  () => props.initialTab,
  (tab) => {
    activeTab.value = tab === 'wechat' ? 'wechat' : 'trade'
  },
)
</script>
