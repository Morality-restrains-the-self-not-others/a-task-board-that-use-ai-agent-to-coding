<template>
  <div class="bg-white rounded-xl shadow overflow-hidden">
    <!-- Collapse toggle header -->
    <button
      type="button"
      class="w-full px-4 py-3 flex items-center gap-3 hover:bg-gray-50/80 transition-colors text-left group"
      @click="collapsed = !collapsed"
    >
      <!-- Icon -->
      <div class="flex-shrink-0 w-8 h-8 rounded-lg bg-gradient-to-br from-violet-100 to-purple-100 flex items-center justify-center">
        <svg class="w-4 h-4 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9.75 3.104v5.714a2.25 2.25 0 01-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 014.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0112 15a9.065 9.065 0 00-6.23.693L5 14.5m14.8.8l1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0112 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L4.2 15.3" />
        </svg>
      </div>
      <!-- Title + summary -->
      <div class="flex-1 min-w-0">
        <h3 class="text-lg font-bold text-text">LLM 供应商 & 模型配置</h3>
        <p v-if="collapsed && summaryText" class="text-xs text-text-light mt-0.5 truncate">{{ summaryText }}</p>
      </div>
      <!-- Chevron -->
      <svg
        class="flex-shrink-0 w-5 h-5 text-gray-400 transition-transform duration-300"
        :class="{ 'rotate-180': !collapsed }"
        fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    <!-- Expandable content with transition -->
    <div
      class="overflow-hidden transition-all duration-300 ease-in-out"
      :class="collapsed ? 'max-h-0' : 'max-h-[4000px]'"
    >
      <div class="px-4 pb-4 space-y-4 border-t border-gray-100">
        <!-- Readonly banner -->
        <div v-if="readonly" class="flex items-center gap-2 bg-amber-50 border border-amber-200 text-amber-700 text-sm rounded-lg px-3 py-2">
          <span>🔒</span>
          <span>以下配置继承自<strong>公司默认配置</strong>，仅可查看不可编辑。如需修改，请先选择「自定义工作空间配置」。</span>
        </div>

        <!-- Recommended providers -->
        <div class="bg-gradient-to-br from-blue-50/60 to-indigo-50/40 rounded-xl border border-blue-100 overflow-hidden">
          <button
            type="button"
            class="w-full px-4 py-2.5 flex items-center justify-between hover:bg-white/40 transition-colors"
            @click="showRecommended = !showRecommended"
          >
            <div class="flex items-center gap-2.5">
              <span class="text-lg">📚</span>
              <div class="text-left">
                <h4 class="text-sm font-semibold text-blue-800">推荐供应商</h4>
                <p class="text-xs text-blue-600/70">查看主流 LLM 供应商的文档与价格参考</p>
              </div>
            </div>
            <span class="text-xs font-medium text-blue-600 hover:text-blue-700 flex items-center gap-1">
              {{ showRecommended ? '收起' : '展开' }}
              <svg class="w-3.5 h-3.5 transition-transform duration-200" :class="{ 'rotate-180': showRecommended }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"/></svg>
            </span>
          </button>
          <div v-if="showRecommended" class="px-4 pb-3 space-y-3">
            <!-- Provider cards grid -->
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
              <a
                v-for="item in visibleRecommendedProviders"
                :key="item.id"
                :href="item.docs_url"
                target="_blank"
                rel="noopener noreferrer"
                class="group/card flex items-start gap-2.5 bg-white rounded-lg p-2.5 border border-blue-100 hover:border-blue-300 hover:shadow-sm transition-all"
              >
                <span class="flex-shrink-0 w-7 h-7 rounded bg-blue-100 flex items-center justify-center text-xs font-bold text-blue-600 group-hover/card:bg-blue-200 transition-colors">
                  {{ item.name.charAt(0).toUpperCase() }}
                </span>
                <div class="flex-1 min-w-0">
                  <p class="font-medium text-sm text-text group-hover/card:text-blue-700 transition-colors">{{ item.name }}</p>
                  <p class="text-xs text-text-light mt-0.5 line-clamp-2">{{ item.description || '模型与价格文档' }}</p>
                </div>
              </a>
            </div>
            <div v-if="recommendedProviders.length === 0" class="text-sm text-gray-400 text-center py-2">暂无推荐供应商</div>
            <button
              v-if="recommendedProviders.length > 3"
              type="button"
              class="text-blue-600 hover:text-blue-700 text-sm font-medium hover:underline w-full text-center"
              @click="showAllProviders = !showAllProviders"
            >
              {{ showAllProviders ? '收起' : `查看更多推荐供应商（共 ${recommendedProviders.length} 个）` }}
            </button>
            <!-- Disclaimer -->
            <div class="flex items-start gap-2 bg-amber-50 border border-amber-200 text-amber-700 text-xs rounded-lg p-2.5">
              <span class="flex-shrink-0 mt-0.5">⚠️</span>
              <span>免责说明：推荐列表仅供参考，不构成任何采购、服务质量或安全合规承诺。请结合自身场景独立评估供应商可靠性。</span>
            </div>
          </div>
        </div>

        <!-- Providers editor section -->
        <div>
          <div class="flex items-center gap-2 mb-3">
            <span class="w-1.5 h-4 rounded-full bg-violet-500"></span>
            <h4 class="text-sm font-semibold text-text">供应商列表</h4>
            <span class="text-xs text-text-light">({{ providerCountLabel }})</span>
          </div>
          <div class="bg-gray-50/60 rounded-lg p-4 border border-gray-100">
            <FeatureParamsProvidersEditor
              :providers="providers"
              :sub-token-providers="subTokenProviders"
            />
          </div>
        </div>

        <!-- Model configuration cards -->
        <div>
          <div class="flex items-center gap-2 mb-3">
            <span class="w-1.5 h-4 rounded-full bg-purple-500"></span>
            <h4 class="text-sm font-semibold text-text">模型配置</h4>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <!-- Agent model card -->
            <div class="bg-gray-50/60 rounded-lg p-4 border border-gray-100" :class="{ 'opacity-70': readonly }">
              <div class="flex items-center gap-1.5 mb-3">
                <span class="text-base">🤖</span>
                <h5 class="font-semibold text-text text-sm">智能体模型</h5>
              </div>
              <div class="space-y-2.5">
                <div>
                  <label class="block text-xs font-medium text-text-light mb-1">供应商</label>
                  <select
                    :value="agentModelProvider"
                    @change="emit('update:agentModelProvider', $event.target.value)"
                    :disabled="readonly"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-400 focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500 disabled:cursor-not-allowed"
                  >
                    <option value="">请选择 provider</option>
                    <option v-for="provider in providerOptions" :key="`agent-${provider}`" :value="provider">
                      {{ provider }}
                    </option>
                  </select>
                </div>
                <div>
                  <label class="block text-xs font-medium text-text-light mb-1">模型名称</label>
                  <input
                    :value="agentModel"
                    @input="emit('update:agentModel', $event.target.value)"
                    list="agent-model-options-panel"
                    type="text"
                    placeholder="例如：gpt-4.1"
                    :disabled="readonly"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-400 focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500 disabled:cursor-not-allowed"
                  >
                  <datalist id="agent-model-options-panel">
                    <option v-for="model in agentModelOptions" :key="`agent-model-${model}`" :value="model" />
                  </datalist>
                  <p v-if="agentModelOutOfListHint" class="mt-1 text-xs text-red-600">
                    {{ agentModelOutOfListHint }}
                  </p>
                  <p v-else-if="agentModelProvider && agentModelOptions.length > 0" class="mt-1 text-xs text-purple-600">
                    已按供应商 {{ agentModelProvider }} 提供 {{ agentModelOptions.length }} 个候选模型
                  </p>
                </div>
                <div>
                  <label class="block text-xs font-medium text-text-light mb-1">自动迭代次数</label>
                  <input
                    :value="agentMaxSteps"
                    @input="emit('update:agentMaxSteps', $event.target.value)"
                    type="number"
                    min="1"
                    step="1"
                    placeholder="例如：200"
                    :disabled="readonly"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-400 focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500 disabled:cursor-not-allowed"
                  >
                </div>
              </div>
            </div>

            <!-- Summary model card -->
            <div class="bg-gray-50/60 rounded-lg p-4 border border-gray-100" :class="{ 'opacity-70': readonly }">
              <div class="flex items-center gap-1.5 mb-3">
                <span class="text-base">📋</span>
                <h5 class="font-semibold text-text text-sm">摘要模型</h5>
              </div>
              <div class="space-y-2.5">
                <div>
                  <label class="block text-xs font-medium text-text-light mb-1">供应商</label>
                  <select
                    :value="summaryModelProvider"
                    @change="emit('update:summaryModelProvider', $event.target.value)"
                    :disabled="readonly"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-400 focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500 disabled:cursor-not-allowed"
                  >
                    <option value="">请选择 provider</option>
                    <option v-for="provider in providerOptions" :key="`summary-${provider}`" :value="provider">
                      {{ provider }}
                    </option>
                  </select>
                </div>
                <div>
                  <label class="block text-xs font-medium text-text-light mb-1">模型名称</label>
                  <input
                    :value="summaryModel"
                    @input="emit('update:summaryModel', $event.target.value)"
                    list="summary-model-options-panel"
                    type="text"
                    placeholder="例如：gpt-4.1-mini"
                    :disabled="readonly"
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-purple-400 focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500 disabled:cursor-not-allowed"
                  >
                  <datalist id="summary-model-options-panel">
                    <option v-for="model in summaryModelOptions" :key="`summary-model-${model}`" :value="model" />
                  </datalist>
                  <p v-if="summaryModelOutOfListHint" class="mt-1 text-xs text-red-600">
                    {{ summaryModelOutOfListHint }}
                  </p>
                  <p v-else-if="summaryModelProvider && summaryModelOptions.length > 0" class="mt-1 text-xs text-purple-600">
                    已按供应商 {{ summaryModelProvider }} 提供 {{ summaryModelOptions.length }} 个候选模型
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Budget panel -->
        <div v-if="hasAnyBudgetEnabled">
          <div class="flex items-center gap-2 mb-3">
            <span class="w-1.5 h-4 rounded-full bg-emerald-500"></span>
            <h4 class="text-sm font-semibold text-text">LLM 预算</h4>
          </div>
          <div class="bg-gray-50/60 rounded-lg p-4 border border-gray-100">
            <FeatureParamsWorkspaceBudgetPanel
              :tenant-id="tenantId"
              :enabled="hasAnyBudgetEnabled"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import FeatureParamsProvidersEditor from './FeatureParamsProvidersEditor.vue'
import FeatureParamsWorkspaceBudgetPanel from './FeatureParamsWorkspaceBudgetPanel.vue'
import {
  parseSupportedModels,
  validateModelsAgainstProviders,
} from '../utils/featureParamsModelValidation'

const props = defineProps({
  providers: { type: Array, required: true },
  subTokenProviders: { type: Array, default: () => [] },
  recommendedProviders: { type: Array, default: () => [] },
  agentModel: { type: String, default: '' },
  agentModelProvider: { type: String, default: '' },
  agentMaxSteps: { type: [String, Number], default: '200' },
  summaryModel: { type: String, default: '' },
  summaryModelProvider: { type: String, default: '' },
  tenantId: { type: String, default: '' },
  hasAnyBudgetEnabled: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:agentModel',
  'update:agentModelProvider',
  'update:agentMaxSteps',
  'update:summaryModel',
  'update:summaryModelProvider',
])

const collapsed = ref(true)
const showRecommended = ref(false)
const showAllProviders = ref(false)

const providerCountLabel = computed(() => {
  const count = props.providers.filter(p => (p.provider || '').trim()).length
  return `${count} 个已配置`
})

const summaryText = computed(() => {
  const parts = []
  const count = props.providers.filter(p => (p.provider || '').trim()).length
  if (count > 0) parts.push(`${count} 个供应商`)
  if (props.agentModel) parts.push(`智能体: ${props.agentModel}`)
  else if (props.agentModelProvider) parts.push(`智能体供应商: ${props.agentModelProvider}`)
  if (props.summaryModel) parts.push(`摘要: ${props.summaryModel}`)
  else if (props.summaryModelProvider) parts.push(`摘要供应商: ${props.summaryModelProvider}`)
  return parts.length > 0 ? parts.join(' · ') : '点击展开配置 LLM 供应商与模型'
})

const providerOptions = computed(() => {
  const names = props.providers
    .map((item) => (item.provider || '').trim())
    .filter(Boolean)
  return Array.from(new Set(names))
})

const providerSupportedModelsMap = computed(() => {
  const result = {}
  props.providers.forEach((item) => {
    const provider = String(item.provider || '').trim()
    if (!provider) return
    result[provider] = Array.from(new Set(parseSupportedModels(item.supported_models_text || item.supported_models)))
  })
  return result
})

const agentModelOptions = computed(() => {
  const provider = String(props.agentModelProvider || '').trim()
  if (!provider) return []
  return providerSupportedModelsMap.value[provider] || []
})

const summaryModelOptions = computed(() => {
  const provider = String(props.summaryModelProvider || '').trim()
  if (!provider) return []
  return providerSupportedModelsMap.value[provider] || []
})

const agentModelOutOfListHint = computed(() => {
  const model = String(props.agentModel || '').trim()
  if (!model) return ''
  return validateModelsAgainstProviders(props.providers, {
    agentModel: props.agentModel,
    agentModelProvider: props.agentModelProvider,
    summaryModel: '',
    summaryModelProvider: '',
  }) || ''
})

const summaryModelOutOfListHint = computed(() => {
  const model = String(props.summaryModel || '').trim()
  if (!model) return ''
  return validateModelsAgainstProviders(props.providers, {
    agentModel: '',
    agentModelProvider: '',
    summaryModel: props.summaryModel,
    summaryModelProvider: props.summaryModelProvider,
  }) || ''
})

const visibleRecommendedProviders = computed(() => {
  if (showAllProviders.value) return props.recommendedProviders
  return props.recommendedProviders.slice(0, 3)
})
</script>
