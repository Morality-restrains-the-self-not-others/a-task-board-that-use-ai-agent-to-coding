<template>
  <div class="tag-member-input" ref="rootEl">
    <div class="flex flex-wrap gap-1 p-1.5 border border-gray-300 rounded-md bg-white min-h-[36px] focus-within:ring-1 focus-within:ring-primary focus-within:border-primary cursor-text" @click="focusInput">
      <!-- 已选 tag -->
      <span
        v-for="id in modelValue"
        :key="id"
        class="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 text-blue-800 rounded-full text-xs font-medium"
      >
        {{ optionName(id) }}
        <button
          type="button"
          class="inline-flex items-center justify-center w-4 h-4 rounded-full hover:bg-blue-200 focus:outline-none"
          @click.stop="remove(id)"
        >
          &times;
        </button>
      </span>

      <!-- 输入框 -->
      <input
        ref="inputEl"
        v-model="searchText"
        type="text"
        :placeholder="modelValue.length === 0 ? placeholder : ''"
        class="flex-1 min-w-[100px] border-none outline-none text-sm bg-transparent px-1 py-0.5"
        @focus="onFocus"
        @blur="onBlur"
        @keydown.enter.prevent="addHighlighted"
        @keydown.backspace="onBackspace"
        @keydown.arrow-down.prevent="highlightNext"
        @keydown.arrow-up.prevent="highlightPrev"
        @keydown.escape.prevent="closeDropdown"
      />
    </div>

    <!-- 下拉选项 -->
    <ul
      v-if="dropdownOpen && filteredOptions.length > 0"
      class="absolute z-50 mt-1 w-full max-h-40 overflow-y-auto bg-white border border-gray-200 rounded-md shadow-lg text-sm"
    >
      <li
        v-for="(opt, idx) in filteredOptions"
        :key="opt.id"
        :ref="(el) => { if (el) dropdownItems[idx] = el }"
        class="px-3 py-1.5 cursor-pointer hover:bg-blue-50"
        :class="{ 'bg-blue-100': idx === highlightIndex }"
        @mousedown.prevent="add(opt.id)"
      >
        {{ opt.name }}
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  options: { type: Array, default: () => [] },
  placeholder: { type: String, default: '搜索...' },
})

const emit = defineEmits(['update:modelValue'])

const rootEl = ref(null)
const inputEl = ref(null)
const searchText = ref('')
const dropdownOpen = ref(false)
const highlightIndex = ref(0)
const dropdownItems = ref({})

const selectedSet = computed(() => new Set((props.modelValue || []).map(String)))

const filteredOptions = computed(() => {
  const q = searchText.value.trim().toLowerCase()
  return (props.options || []).filter((o) => {
    if (selectedSet.value.has(String(o.id))) return false
    if (!q) return true
    return (o.name || '').toLowerCase().includes(q)
  })
})

const optionName = (id) => {
  const found = (props.options || []).find((o) => String(o.id) === String(id))
  return found ? found.name : `ID:${id}`
}

const focusInput = () => {
  inputEl.value?.focus()
}

const onFocus = () => {
  dropdownOpen.value = true
  highlightIndex.value = 0
}

const onBlur = () => {
  // 延迟关闭以允许 mousedown 先触发
  setTimeout(() => { dropdownOpen.value = false }, 150)
}

const add = (id) => {
  const next = [...(props.modelValue || []), id]
  emit('update:modelValue', next)
  searchText.value = ''
  highlightIndex.value = 0
  nextTick(() => inputEl.value?.focus())
}

const remove = (id) => {
  const next = (props.modelValue || []).filter((v) => String(v) !== String(id))
  emit('update:modelValue', next)
}

const addHighlighted = () => {
  const opts = filteredOptions.value
  if (opts.length === 0) return
  const idx = Math.min(Math.max(0, highlightIndex.value), opts.length - 1)
  add(opts[idx].id)
}

const onBackspace = () => {
  if (searchText.value === '' && (props.modelValue || []).length > 0) {
    const next = props.modelValue.slice(0, -1)
    emit('update:modelValue', next)
  }
}

const highlightNext = () => {
  const max = filteredOptions.value.length
  if (max > 0) highlightIndex.value = (highlightIndex.value + 1) % max
}

const highlightPrev = () => {
  const max = filteredOptions.value.length
  if (max > 0) highlightIndex.value = (highlightIndex.value - 1 + max) % max
}

const closeDropdown = () => {
  dropdownOpen.value = false
}

watch(filteredOptions, () => {
  if (highlightIndex.value >= filteredOptions.value.length) {
    highlightIndex.value = Math.max(0, filteredOptions.value.length - 1)
  }
})
</script>

<style scoped>
.tag-member-input {
  position: relative;
}
</style>
