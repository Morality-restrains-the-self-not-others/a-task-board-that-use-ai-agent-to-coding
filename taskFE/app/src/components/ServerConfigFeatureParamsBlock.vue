<template>
  <div class="mt-4 space-y-2" data-testid="feature-params-block">
    <label
      :class="variant === 'form' ? 'block text-sm font-medium text-gray-700' : 'block text-xs text-gray-600'"
      for="feature-params-source-selector"
    >智能体资源配置</label>
    <select
      id="feature-params-source-selector"
      :value="featureParamsSource"
      :class="selectClass"
      :disabled="!sourcesAvailable"
      data-testid="feature-params-source-selector"
      @change="onSourceSelect($event)"
    >
      <option value="" disabled>-- 请选择智能体资源配置 --</option>
      <option value="company">公司默认</option>
      <option value="workspace">工作空间默认</option>
      <option value="personal">个人配置</option>
    </select>
    <p
      v-if="!sourcesAvailable"
      class="text-[11px] text-gray-400"
      data-testid="feature-params-source-unavailable-hint"
      role="status"
    >
      暂无可用智能体资源配置，请先在「智能体资源配置」设置页配置
      <a
        v-if="settingsHrefs.company"
        data-testid="feature-params-company-settings-link"
        class="text-primary underline hover:text-blue-700"
        :href="settingsHrefs.company"
      >公司</a><template v-else>公司</template>、
      <a
        v-if="settingsHrefs.workspace"
        data-testid="feature-params-workspace-settings-link"
        class="text-primary underline hover:text-blue-700"
        :href="settingsHrefs.workspace"
      >工作空间</a><template v-else>工作空间</template>
      或
      <a
        data-testid="feature-params-personal-settings-link"
        class="text-primary underline hover:text-blue-700"
        :href="settingsHrefs.personal"
      >个人环境变量</a>
    </p>
    <p
      v-else-if="showRequiredHint && !featureParamsSource"
      class="text-[11px] text-amber-700"
      data-testid="feature-params-source-required-hint"
      role="status"
    >
      {{ sourceRequiredHint }}
    </p>

    <div v-if="featureParamsSource === 'personal'" class="space-y-2">
      <label
        :class="variant === 'form' ? 'block text-sm font-medium text-gray-700' : 'block text-xs text-gray-600'"
        for="feature-params-personal-config-selector"
      >个人配置</label>
      <select
        id="feature-params-personal-config-selector"
        :value="selectedPersonalConfigId"
        :class="selectClass"
        data-testid="feature-params-personal-config-selector"
        @change="$emit('update:selectedPersonalConfigId', ($event.target).value)"
      >
        <option value="" disabled>-- 请选择个人配置 --</option>
        <option
          v-for="cfg in personalConfigs"
          :key="cfg.id"
          :value="cfg.id"
        >{{ cfg.name }}</option>
      </select>
      <p
        v-if="showRequiredHint && !selectedPersonalConfigId"
        class="text-[11px] text-amber-700"
        data-testid="feature-params-personal-config-required-hint"
        role="status"
      >
        {{ sourceRequiredHint }}
      </p>
      <p v-if="personalConfigs.length === 0" class="text-[11px] text-gray-400">
        暂无个人配置，请先在「智能体资源配置」设置页创建
      </p>
      <div v-if="showPreview" class="flex items-center gap-2">
        <button
          type="button"
          class="px-3 py-1 text-[11px] border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-60"
          data-testid="feature-params-env-preview-btn"
          :disabled="!selectedPersonalConfigId || isEnvPreviewLoading"
          @click="$emit('previewEnv')"
        >
          {{ isEnvPreviewLoading ? '加载中...' : '预览环境变量' }}
        </button>
      </div>

      <div
        v-if="showPreview && envPreviewExpanded"
        class="p-3 bg-gray-50 border border-gray-200 rounded-md max-h-64 overflow-y-auto"
        data-testid="feature-params-env-preview-panel"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-gray-600">解析后的环境变量</span>
          <button
            type="button"
            class="text-[11px] text-gray-400 hover:text-gray-600"
            @click="$emit('update:envPreviewExpanded', false)"
          >✕ 收起</button>
        </div>
        <div
          v-for="(value, key) in resolvedEnvPreview"
          :key="'preview-' + key"
          class="grid grid-cols-[minmax(160px,1fr)_minmax(200px,2fr)] gap-1 py-0.5 text-[11px]"
        >
          <span class="font-mono text-gray-700 truncate">{{ key }}</span>
          <span class="font-mono text-gray-500 break-all">{{ isSensitivePreviewKey(key) ? '●●●●' : value }}</span>
        </div>
      </div>
    </div>

    <p
      v-if="persistError"
      class="text-[11px] text-red-600"
      data-testid="feature-params-persist-error"
      role="alert"
    >
      {{ persistError }}
    </p>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { featureParamsSettingsHrefs } from '../utils/envParamsSourceSelection.js'

const SENSITIVE_KEY_PATTERN = /token|key|secret|password|api_key|access/i

function isSensitivePreviewKey(key) {
  return SENSITIVE_KEY_PATTERN.test(key || '')
}

const props = defineProps({
  featureParamsSource: { type: String, default: '' },
  personalConfigs: { type: Array, default: () => [] },
  selectedPersonalConfigId: { type: String, default: '' },
  resolvedEnvPreview: { type: Object, default: () => ({}) },
  isEnvPreviewLoading: { type: Boolean, default: false },
  envPreviewExpanded: { type: Boolean, default: false },
  /** compact：镜像卡；form：创建/编辑任务表单 */
  variant: { type: String, default: 'compact' },
  showRequiredHint: { type: Boolean, default: true },
  sourceRequiredHint: { type: String, default: '请先选择智能体资源配置后再启动' },
  showPreview: { type: Boolean, default: true },
  /** PATCH 持久化失败时的用户可见错误（如非 Owner 403） */
  persistError: { type: String, default: '' },
  /**
   * 公司/工作空间/个人三类环境变量是否至少存在一个可选项；
   * false 时选择器禁用（无可用来源时无意义可点）。
   */
  sourcesAvailable: { type: Boolean, default: true },
  /** 当前租户，用于空状态「公司 / 工作空间」设置页链接 */
  tenantId: { type: [String, Number], default: '' },
})

const settingsHrefs = computed(() => featureParamsSettingsHrefs(props.tenantId))

const selectClass = computed(() => (
  props.variant === 'form'
    ? 'mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary focus:border-primary disabled:opacity-60 disabled:cursor-not-allowed'
    : 'w-full max-w-xs px-3 py-1.5 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-primary focus:border-primary disabled:opacity-60 disabled:cursor-not-allowed'
))

const emit = defineEmits([
  'sourceChange',
  'previewEnv',
  'update:featureParamsSource',
  'update:selectedPersonalConfigId',
  'update:envPreviewExpanded',
])

function onSourceSelect(event) {
  emit('update:featureParamsSource', event.target.value)
  emit('sourceChange')
}
</script>
