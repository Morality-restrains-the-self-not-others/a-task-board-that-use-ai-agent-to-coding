<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <h4 class="text-base font-semibold text-text">模型供应商列表（可多份）</h4>
      <button class="px-3 py-1.5 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50" type="button" @click="addProvider">
        新增 provider
      </button>
    </div>

    <div v-for="(provider, index) in providers" :key="index" class="border border-gray-200 rounded-lg p-4">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-text-light mb-1">供应商</label>
          <input
            v-model.trim="provider.provider"
            type="text"
            placeholder="例如：openai"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @change="onProviderNameChange(index)"
          >
        </div>
        <div>
          <label class="block text-sm font-medium text-text-light mb-1">api_key</label>
          <div class="flex items-center gap-2">
            <input
              v-model.trim="provider.api_key"
              type="password"
              placeholder="请输入 API Key"
              class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            >
            <template v-if="subTokenUiEnabled">
              <label class="flex items-center gap-1.5 cursor-pointer">
                <input
                  v-model="provider.use_sub_token"
                  type="checkbox"
                  class="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                  @change="onUseSubTokenChange(index)"
                >
                <span class="text-sm font-medium text-text-light">启用派生子Key</span>
              </label>
              <label
                class="flex items-center gap-1.5 cursor-pointer"
                :class="provider.use_sub_token ? '' : 'opacity-50'"
                :title="provider.use_sub_token ? '' : '须先启用派生子 Key'"
              >
                <input
                  v-model="provider.budget_enabled"
                  type="checkbox"
                  class="w-4 h-4 text-primary border-gray-300 rounded focus:ring-primary"
                  :disabled="!provider.use_sub_token"
                >
                <span class="text-sm font-medium text-text-light">启用 LLM 预算（本端点）</span>
              </label>
            </template>
          </div>
          <p v-if="subTokenUiEnabled && isSubTokenProvider(provider.provider)" class="mt-1 text-xs text-primary">该供应商支持派生Token，已默认勾选</p>
        </div>
        <div class="md:col-span-2">
          <label class="block text-sm font-medium text-text-light mb-1">API 端点(base_url)</label>
          <input
            v-model.trim="provider.base_url"
            type="text"
            placeholder="例如：https://api.openai.com/v1"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @blur="onBaseUrlBlur(index)"
          >
        </div>
        <div class="md:col-span-2">
          <label class="block text-sm font-medium text-text-light mb-1">支持模型列表（每行一个，可用中英文逗号/分号分隔）</label>
          <textarea
            v-model="provider.supported_models_text"
            rows="3"
            placeholder="例如：gpt-4.1，gpt-4.1-mini"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
            @blur="normalizeSupportedModelsTextAt(index)"
          />
        </div>
      </div>
      <div class="mt-3 flex justify-end">
        <button
          class="px-3 py-1.5 text-red-600 border border-red-200 rounded-md hover:bg-red-50"
          type="button"
          @click="removeProvider(index)"
          :disabled="providers.length === 1"
        >
          删除
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { watch } from 'vue'
import { normalizeSupportedModelsText } from '../utils/featureParamsModelValidation.js'
import {
  FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED,
  applySubTokenUiPolicy,
} from '../utils/featureParamsSubTokenUi.js'
import { repairConcatenatedAbsoluteUrl } from '../utils/featureParamsBaseUrl.js'

const props = defineProps({
  providers: { type: Array, required: true },
  subTokenProviders: { type: Array, default: () => [] },
})

const subTokenUiEnabled = FEATURE_PARAMS_SUB_TOKEN_UI_ENABLED

watch(
  () => props.providers,
  (list) => {
    for (const provider of list || []) {
      applySubTokenUiPolicy(provider)
    }
  },
  { deep: true, immediate: true },
)

const normalizeSupportedModelsTextAt = (index) => {
  const provider = props.providers[index]
  if (!provider) return
  provider.supported_models_text = normalizeSupportedModelsText(
    provider.supported_models_text || provider.supported_models,
  )
}

const isSubTokenProvider = (providerName) => {
  const name = String(providerName || '').trim().toLowerCase()
  return props.subTokenProviders.some((p) => p.provider_name.toLowerCase() === name)
}

const onUseSubTokenChange = (index) => {
  const provider = props.providers[index]
  if (!provider.use_sub_token) {
    provider.budget_enabled = false
  }
}

const updateDefaultUseSubToken = (index) => {
  if (!subTokenUiEnabled) return
  const provider = props.providers[index]
  if (isSubTokenProvider(provider.provider)) {
    provider.use_sub_token = true
  }
}

const onBaseUrlBlur = (index) => {
  const provider = props.providers[index]
  if (!provider) return
  provider.base_url = repairConcatenatedAbsoluteUrl(provider.base_url)
}

const onProviderNameChange = (index) => {
  updateDefaultUseSubToken(index)
}

const addProvider = () => {
  props.providers.push({
    provider: '',
    api_key: '',
    base_url: '',
    supported_models_text: '',
    use_sub_token: false,
    budget_enabled: false,
  })
}

const removeProvider = (index) => {
  if (props.providers.length === 1) return
  props.providers.splice(index, 1)
}
</script>
