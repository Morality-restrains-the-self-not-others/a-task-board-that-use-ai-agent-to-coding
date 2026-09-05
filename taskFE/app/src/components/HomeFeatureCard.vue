<template>
  <div class="overflow-hidden group bg-white shadow-lg rounded-lg transition-all duration-300 p-8">
    <div
      class="bg-primary/10 rounded-2xl w-16 h-16 flex items-center justify-center mb-6 transition-all duration-300 group-hover:bg-primary/20"
    >
      <svg class="w-8 h-8 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="icon" />
      </svg>
    </div>
    <h3 class="text-2xl font-bold text-gray-800 mb-3">{{ title }}</h3>
    <p class="text-gray-600 mb-6">{{ description }}</p>
    <a
      :href="href"
      class="inline-flex items-center text-primary font-medium hover:text-purple-700 transition-colors duration-300 group-hover:translate-x-1"
    >
      了解更多
      <svg
        class="w-5 h-5 ml-2 transition-transform duration-300 group-hover:translate-x-1"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" />
      </svg>
    </a>
  </div>
</template>

<script setup>
/**
 * 首页「核心功能」三张特性卡的可复用子组件（OPT-20260811-071）。
 * 图标/标题/文案/CTA 结构完全一致，抽离避免三份拷贝样式漂移。
 */
defineProps({
  icon: { type: String, required: true },
  title: { type: String, required: true },
  description: { type: String, required: true },
  href: { type: String, default: '#' },
})
</script>
