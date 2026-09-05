<template>
  <div class="space-y-3" data-testid="resource-grant-matrix">
    <div
      v-for="page in pages"
      :key="page.id"
      class="border border-border/60 rounded-lg p-2"
    >
      <label class="flex items-center gap-2 text-sm font-medium cursor-pointer">
        <input
          type="checkbox"
          class="rounded border-border"
          :checked="isPageFullySelected(page)"
          :disabled="disabled"
          @change="togglePage(page, $event.target.checked)"
        />
        <span>{{ page.display_name }}</span>
        <span class="text-xs text-text-light font-normal">（整页）</span>
      </label>
      <div class="mt-2 ml-6 grid grid-cols-1 gap-2">
        <div
          v-for="region in page.children || []"
          :key="region.id"
          class="flex flex-wrap items-center gap-x-3 gap-y-1 px-2 py-1.5 rounded border border-border/60 text-sm"
        >
          <span class="truncate min-w-0 flex-1 font-medium">{{ region.display_name }}</span>
          <label class="flex items-center gap-1.5 cursor-pointer text-xs shrink-0">
            <input
              type="checkbox"
              class="rounded border-border"
              :checked="Boolean(modelValue[region.group_key])"
              :disabled="disabled"
              @change="onViewToggle(region.group_key, $event.target.checked)"
            />
            <span>可访问</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer text-xs shrink-0">
            <input
              type="checkbox"
              class="rounded border-border"
              :checked="modelValue[region.group_key] === 'operate'"
              :disabled="disabled"
              @change="onOperateToggle(region.group_key, $event.target.checked)"
            />
            <span>可编辑执行</span>
          </label>
        </div>
      </div>
    </div>
    <p v-if="!pages.length" class="text-sm text-text-light text-center py-4">暂无资源组目录</p>
  </div>
</template>

<script setup>
import {
  EFFECT_OPERATE,
  EFFECT_VIEW,
  isPageFullyGranted,
  setGrantEffect,
  togglePageGrants,
} from '../domain/auth/resourceGrantEffects.js'

const props = defineProps({
  pages: { type: Array, default: () => [] },
  modelValue: { type: Object, default: () => ({}) },
  disabled: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

function isPageFullySelected(page) {
  return isPageFullyGranted(props.modelValue, page)
}

function togglePage(page, checked) {
  emit('update:modelValue', togglePageGrants(props.modelValue, page, checked))
}

function onViewToggle(groupKey, checked) {
  emit('update:modelValue', setGrantEffect(props.modelValue, groupKey, checked ? EFFECT_VIEW : null))
}

function onOperateToggle(groupKey, checked) {
  if (checked) {
    emit('update:modelValue', setGrantEffect(props.modelValue, groupKey, EFFECT_OPERATE))
    return
  }
  if (props.modelValue[groupKey]) {
    emit('update:modelValue', setGrantEffect(props.modelValue, groupKey, EFFECT_VIEW))
  }
}
</script>
