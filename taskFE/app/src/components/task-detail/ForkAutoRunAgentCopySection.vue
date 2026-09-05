<template>
  <div class="space-y-3" data-testid="fork-auto-run-agent-copy-section">
    <div class="space-y-2">
      <label class="text-sm font-medium text-gray-800" for="fork-agent-resource-source">智能体资源</label>
      <select
        id="fork-agent-resource-source"
        data-testid="fork-agent-resource-source"
        class="w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm text-gray-900 disabled:opacity-50"
        :disabled="disabled"
        :value="featureParamsSource"
        @change="$emit('update:featureParamsSource', $event.target.value)"
      >
        <option value="" disabled>-- 请选择智能体资源配置 --</option>
        <option value="company">公司默认</option>
        <option value="workspace">工作空间默认</option>
        <option value="personal">个人配置</option>
      </select>
      <div v-if="featureParamsSource === 'personal'" class="space-y-1">
        <label class="text-xs text-gray-600" for="fork-agent-personal-config">个人配置</label>
        <select
          id="fork-agent-personal-config"
          data-testid="fork-agent-personal-config"
          class="w-full px-2 py-1.5 border border-gray-300 rounded-md text-sm text-gray-900 disabled:opacity-50"
          :disabled="disabled"
          :value="personalConfigId"
          @change="$emit('update:personalConfigId', $event.target.value)"
        >
          <option value="" disabled>-- 请选择个人配置 --</option>
          <option
            v-for="cfg in personalConfigs"
            :key="cfg.id"
            :value="cfg.id"
          >{{ cfg.name }}</option>
        </select>
      </div>
    </div>
    <div class="space-y-2">
      <p class="text-sm font-medium text-gray-800">智能体</p>
      <p
        v-if="loading"
        class="text-xs text-gray-500"
        data-testid="fork-agent-models-loading"
      >正在加载模型清单…</p>
      <div v-else-if="loadError" class="space-y-1">
        <p
          class="text-xs text-red-600"
          role="alert"
          data-testid="fork-agent-models-error"
          :data-traceId="loadErrorTraceId || undefined"
        >{{ loadError }}</p>
        <button
          type="button"
          class="text-xs px-2 py-1 border border-gray-300 rounded text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          data-testid="fork-agent-models-retry"
          :disabled="disabled"
          @click="$emit('retry-models')"
        >
          <!-- Anti-Replay-OK: read-only GET 重拉模型清单，不写资源 -->
          重试
        </button>
      </div>
      <div
        v-else-if="modelOptions.length === 0"
        class="text-xs text-amber-700"
        data-testid="fork-agent-models-empty"
      >
        未配置可用模型，请先到智能体资源配置页面设置智能体模型提供商与支持模型列表。
      </div>
      <label
        v-for="model in modelOptions"
        :key="model"
        class="flex items-center gap-2 text-sm text-gray-800"
        data-testid="fork-agent-model-option"
      >
        <input
          type="checkbox"
          :value="model"
          :checked="selectedModels.includes(model)"
          :disabled="disabled"
          :data-testid="`fork-agent-model-${model}`"
          @change="$emit('toggle-model', model)"
        />
        <span>{{ model }}</span>
      </label>
      <p
        v-if="selectedModels.length > 0"
        class="text-xs text-amber-700"
        data-testid="fork-agent-copy-hint"
      >
        将创建 {{ selectedModels.length }} 个副本，每个使用不同模型并尝试启动云资源。
      </p>
    </div>
  </div>
</template>

<script setup>
defineProps({
  disabled: { type: Boolean, default: false },
  featureParamsSource: { type: String, default: '' },
  personalConfigId: { type: String, default: '' },
  personalConfigs: { type: Array, default: () => [] },
  modelOptions: { type: Array, default: () => [] },
  selectedModels: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  loadError: { type: String, default: '' },
  loadErrorTraceId: { type: String, default: '' },
})

defineEmits([
  'update:featureParamsSource',
  'update:personalConfigId',
  'toggle-model',
  'retry-models',
])
</script>
