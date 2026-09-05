<template>
  <div class="p-6" data-testid="tenant-page-access-empty">
    <div class="bg-white p-6 rounded-xl shadow text-center py-16">
      <h2 class="text-lg font-semibold text-text mb-2">暂无访问权限</h2>
      <p class="text-sm text-text-light">
        您没有访问该页面的权限（page:{{ pageKey }}）。如需访问请联系公司管理员开通。
      </p>
    </div>
  </div>
</template>

<script setup>
defineProps({
  pageKey: {
    type: String,
    default: '',
  },
})
</script>
