<template>
  <div class="relative" :class="wrapperClass">
    <input
      :value="search"
      ref="inputRef"
      type="text"
      :data-alias="dataAlias"
      :aria-label="ariaLabel"
      :placeholder="placeholder"
      :class="inputClass"
      @input="onInput"
      @focus="$emit('focus')"
    >
    <!-- Teleport-OK: overflow-clip — 表头搜索下拉在 overflow-x-auto 表格内会被裁剪 -->
    <Teleport to="body">
      <div
        v-if="showDropdown && options.length > 0"
        :class="[dropdownClass, 'fixed z-50 mt-1 bg-white border rounded-lg shadow-lg max-h-60 overflow-auto']"
        :style="dropdownStyle"
      >
        <div
          v-for="option in options"
          :key="option.id"
          class="px-3 py-2 hover:bg-gray-100 cursor-pointer"
          @click="$emit('select', option)"
        >
          <div class="font-medium text-sm">{{ option.name }}</div>
          <div v-if="showId" class="text-xs text-gray-500">{{ option.id }}</div>
        </div>
      </div>
    </Teleport>
    <div
      v-if="selectedName"
      :class="selectedClass"
      :title="`已选择: ${selectedName}`"
    >
      已选择: {{ selectedName }}
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted } from 'vue'

// OPT-20260807-046/048 共用表头搜索过滤（输入 + 下拉 + 防抖 + Teleport fixed 定位）。
// 契约：
//   update:search  —— 输入即时同步（受控，:search + update:search）
//   search         —— 300ms 防抖后触发（消费方据此发请求，减少连续输入请求量）
//   focus          —— 输入聚焦（消费方据此打开下拉 / 预加载选项）
//   select         —— 点击选项，携带完整 option 对象

const props = defineProps({
  search: { type: String, default: '' },
  options: { type: Array, default: () => [] },
  showDropdown: { type: Boolean, default: false },
  selectedName: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  dataAlias: { type: String, default: '' },
  wrapperClass: { type: String, default: '' },
  dropdownClass: { type: String, default: '' },
  selectedClass: { type: String, default: 'mt-1 text-xs text-gray-500 truncate' },
  showId: { type: Boolean, default: false },
  inputClass: {
    type: String,
    default:
      'w-full px-2.5 py-1.5 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent',
  },
})

const emit = defineEmits(['update:search', 'search', 'focus', 'select'])

// 300ms 防抖（OPT-20260807-045）：值已受控（update:search 即时），
// 仅 search 请求触发防抖，降低连续输入期间的请求量。
const searchTimeout = ref(null)

const onInput = (event) => {
  emit('update:search', event.target.value)
  if (searchTimeout.value) clearTimeout(searchTimeout.value)
  searchTimeout.value = setTimeout(() => emit('search'), 300)
}

// 下拉 Teleport 到 body 后需按输入框位置 fixed 定位（打开时计算 + 滚动/缩放时跟随）
const inputRef = ref(null)
const dropdownStyle = ref({})

const positionDropdown = () => {
  if (!inputRef.value) return
  const rect = inputRef.value.getBoundingClientRect()
  dropdownStyle.value = {
    top: `${rect.bottom + 4}px`,
    left: `${rect.left}px`,
    width: `${rect.width}px`,
  }
}

watch(
  () => props.showDropdown,
  (show) => {
    if (show) positionDropdown()
  }
)

const reposition = () => positionDropdown()

onMounted(() => {
  window.addEventListener('scroll', reposition, true)
  window.addEventListener('resize', reposition)
})

onUnmounted(() => {
  window.removeEventListener('scroll', reposition, true)
  window.removeEventListener('resize', reposition)
})
</script>
