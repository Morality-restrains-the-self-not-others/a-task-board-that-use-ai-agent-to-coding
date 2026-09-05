<template>
  <div data-alias="view-pricing-page" class="min-h-screen bg-gray-50 font-sans pt-24 pb-16 px-4 sm:px-6 lg:px-8">
    <div class="max-w-3xl mx-auto">
      <!-- 页面标题 -->
      <h1 class="text-3xl font-bold text-gray-900 mb-2 -mt-10">资源收费标准</h1>
      <p class="text-gray-600 mb-4">以下为平台各项资源的计费标准。用户需先购买对应资源，后续使用时直接扣减资源配额。</p>

      <!-- 加载/错误状态 -->
      <div v-if="loadError" class="text-sm text-red-600 mb-4" :data-traceId="loadTraceId">{{ loadError }}</div>

      <!-- ===== 商品列表（外层已保证 !loading，内层 loading 分支为死代码已删，OPT-20260815-011）===== -->
      <section v-if="!loading && pricingData">
        <!-- 资源列表 -->
        <ul class="space-y-4">
          <!-- 任务帖 -->
          <li
            v-if="pricingData.task_post"
            class="bg-white rounded-lg shadow-sm border border-gray-200 p-5"
          >
            <div class="flex items-start justify-between mb-3">
              <div>
                <h3 class="text-lg font-semibold text-gray-800">任务</h3>
                <p class="text-xs text-gray-500 mt-0.5">{{ pricingData.task_post.description }}</p>
              </div>
              <span class="text-xl font-bold text-primary tabular-nums whitespace-nowrap">
                {{ pricingData.task_post.price_yuan || '—' }}
                <span class="text-sm font-normal text-gray-500">元/帖/12个月</span>
              </span>
            </div>

            <!-- 购买门槛标签 -->
            <div class="flex flex-wrap gap-2 mb-3">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"
                :class="tierBadgeClass(pricingData.task_post.required_tier)"
              >
                {{ pricingData.task_post.required_tier_label || '普通会员' }}可购买
              </span>
              <span
                v-if="!pricingData.task_post.min_consumption_cents"
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-50 text-green-700 border border-green-200"
              >
                无最低消费门槛
              </span>
            </div>

            <!-- 购买信息 -->
            <div class="text-sm text-gray-600 space-y-1 bg-gray-50 rounded-lg p-3">
              <p>• 最低购买数量：<span class="font-medium">1 帖</span></p>
              <p>• 购买后发放到账户配额（创建帖次数）；创建帖消耗 1 次并获 <span class="font-medium">12 个月</span> 存续期</p>
              <p>• 帖子到期后下架；续存消耗 1 次配额、不另扣费，再延长 12 个月</p>
            </div>
          </li>

          <!-- GitLab 磁盘 -->
          <li
            v-if="pricingData.gitlab_disk"
            class="bg-white rounded-lg shadow-sm border border-gray-200 p-5"
          >
            <div class="flex items-start justify-between mb-3">
              <div>
                <h3 class="text-lg font-semibold text-gray-800">GitLab 磁盘</h3>
                <p class="text-xs text-gray-500 mt-0.5">{{ pricingData.gitlab_disk.description }}</p>
              </div>
              <span class="text-xl font-bold text-primary tabular-nums whitespace-nowrap">
                {{ pricingData.gitlab_disk.price_yuan || '—' }}
                <span class="text-sm font-normal text-gray-500">元/GB/月</span>
              </span>
            </div>

            <!-- 购买门槛标签 -->
            <div class="flex flex-wrap gap-2 mb-3">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"
                :class="tierBadgeClass(pricingData.gitlab_disk.required_tier)"
              >
                {{ pricingData.gitlab_disk.required_tier_label || 'VIP1' }}可购买
              </span>
              <span
                v-if="pricingData.tiers?.vip1?.upgrade_threshold_yuan"
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-50 text-amber-700 border border-amber-200"
              >
                需累计消费满 {{ pricingData.tiers.vip1.upgrade_threshold_yuan }} 元
              </span>
            </div>

            <!-- 购买信息 -->
            <div class="text-sm text-gray-600 space-y-1 bg-gray-50 rounded-lg p-3">
              <p>• 最低购买数量：<span class="font-medium">{{ pricingData.gitlab_disk.min_quantity || 10 }} GB</span></p>
              <p v-if="pricingData.gitlab_disk.max_quantity">• 累计限购：<span class="font-medium">{{ pricingData.gitlab_disk.max_quantity }} GB/租户</span></p>
              <p>• 按月计费，到期后可续费</p>
            </div>
          </li>

          <!-- GitLab 流量费 -->
          <li
            v-if="pricingData.gitlab_traffic"
            class="bg-white rounded-lg shadow-sm border border-gray-200 p-5"
          >
            <div class="flex items-start justify-between mb-3">
              <div>
                <h3 class="text-lg font-semibold text-gray-800">GitLab 流量费</h3>
                <p class="text-xs text-gray-500 mt-0.5">{{ pricingData.gitlab_traffic.description }}</p>
              </div>
              <span class="text-xl font-bold text-primary tabular-nums whitespace-nowrap">
                {{ pricingData.gitlab_traffic.price_yuan || '—' }}
                <span class="text-sm font-normal text-gray-500">元/GB</span>
              </span>
            </div>

            <!-- 购买门槛标签 -->
            <div class="flex flex-wrap gap-2 mb-3">
              <span
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium"
                :class="tierBadgeClass(pricingData.gitlab_traffic.required_tier)"
              >
                {{ pricingData.gitlab_traffic.required_tier_label || 'VIP1' }}可购买
              </span>
              <span
                v-if="pricingData.tiers?.vip1?.upgrade_threshold_yuan"
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-50 text-amber-700 border border-amber-200"
              >
                需累计消费满 {{ pricingData.tiers.vip1.upgrade_threshold_yuan }} 元
              </span>
              <!-- 前置依赖 -->
              <span
                v-if="pricingData.gitlab_traffic.requires_label"
                class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-50 text-red-700 border border-red-200"
              >
                ⚠ 需先购买 {{ pricingData.gitlab_traffic.requires_label }}
              </span>
            </div>

            <!-- 购买信息 -->
            <div class="text-sm text-gray-600 space-y-1 bg-gray-50 rounded-lg p-3">
              <p>• 最低购买数量：<span class="font-medium">1 GB</span></p>
              <p v-if="pricingData.gitlab_traffic.requires_label">
                • <span class="text-red-600 font-medium">前置条件：</span>须先购买 {{ pricingData.gitlab_traffic.requires_label }}，才可购买流量
              </p>
              <p>• 同区域内网流量不计费</p>
            </div>
          </li>
        </ul>
      </section>

      <!-- ===== 购买与扣费说明 ===== -->
      <section v-if="!loading" class="mt-8 bg-white rounded-lg shadow-sm border border-gray-200 p-5">
        <h2 class="text-lg font-semibold text-gray-800 mb-3">购买与扣费说明</h2>
        <div class="text-sm text-gray-600 space-y-2">
          <p>• 选择所需资源数量下单并支付，资源将直接发放到账户配额中</p>
          <p>• 平台<strong>不提供退款退费服务</strong>，请谨慎购买</p>
          <p>• 普通会员累计消费满 <span class="font-semibold text-amber-700">100 元</span> 后，系统将自动升级为 <span class="font-semibold text-amber-700">VIP1</span>，解锁 GitLab 磁盘和流量购买权限</p>
          <p>• 如有疑问，请联系平台客服</p>
        </div>
      </section>

      <!-- 价格保留声明：置于页面底部 -->
      <div
        class="mt-8 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950"
        data-testid="pricing-change-notice"
        role="note"
      >
        产品价格可能发生变动，平台保留修改价格的权利。请您依据实际用量按需购买，定期查看此页面以获知最新价格信息。
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { apiFetch } from '../utils/apiUtils'
import { safeJson } from '@/utils/safeResponseJson.js'

/* @alias:view-pricing-page */

const loading = ref(true)
const loadError = ref('')
const loadTraceId = ref('')
const pricingData = ref(null)

const tierBadgeClass = (tier) => {
  if (tier === 'vip1') {
    return 'bg-amber-100 text-amber-800 border border-amber-300'
  }
  return 'bg-gray-100 text-gray-700 border border-gray-300'
}

onMounted(async () => {
  loadError.value = ''
  loadTraceId.value = ''
  let requestTraceId = ''
  try {
    // 使用新端点 /api/public/resource-pricing/，返回完整的定价、会员等级、购买门槛数据
    const r = await apiFetch('/api/public/resource-pricing/', { method: 'GET' })
    requestTraceId = r.traceId || ''
    if (!r.ok) {
      loadTraceId.value = requestTraceId
      loadError.value = '暂时无法拉取价格信息，请稍后再试。'
      return
    }
    const data = await safeJson(r, {})
    if (data.status === 'success' && data.pricing) {
      pricingData.value = data.pricing
    } else {
      loadError.value = '价格数据格式异常，请稍后再试。'
    }
  } catch (e) {
    loadTraceId.value = e.traceId || requestTraceId
    loadError.value = '暂时无法拉取价格信息，请稍后再试。'
  } finally {
    loading.value = false
  }
})
</script>
