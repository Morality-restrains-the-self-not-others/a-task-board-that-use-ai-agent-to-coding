<template>
  <div class="p-8">
    <div class="flex flex-col space-y-6">
      <div class="flex justify-between items-center">
        <div>
          <h2 class="text-2xl font-bold text-text">工作空间设置</h2>
          <p class="text-text-light mt-1">管理工作空间的交付物体系设置</p>
        </div>
      </div>

      <TaskSystemSettings />
    </div>
  </div>
</template>

<script setup>
import DeliverableSystemSettings from '../components/DeliverableSystemSettings.vue'
</script>

<style scoped>
</style>
