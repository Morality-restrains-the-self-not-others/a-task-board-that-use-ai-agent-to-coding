<template>
  <div v-if="show" class="app-modal-overlay bg-black/60 flex items-center justify-center z-[60]">
    <div class="bg-white rounded-xl p-6 max-w-md w-full mx-4">
      <h3 class="text-lg font-semibold mb-4">添加可访问该空间的小组</h3>
      <form @submit.prevent="$emit('submit')">
        <div class="mb-4">
          <label for="addGroupSelect" class="block text-sm font-medium text-gray-700 mb-1">选择小组</label>
          <div v-if="loading" class="py-2 text-sm text-gray-500">加载可选小组中...</div>
          <select
            v-else
            id="addGroupSelect"
            v-model="form.groupId"
            class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
            required
          >
            <option value="">请选择小组</option>
            <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
          </select>
          <p v-if="!loading && groups.length === 0" class="mt-1 text-sm text-gray-500">所有小组已拥有访问权限</p>
        </div>
        <div class="mb-4">
          <label for="addGroupRole" class="block text-sm font-medium text-gray-700 mb-1">权限</label>
          <select
            id="addGroupRole"
            v-model="form.role"
            class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
            required
          >
            <option value="view">仅查看</option>
            <option value="edit">可编辑</option>
            <option value="admin">管理员</option>
          </select>
        </div>
        <div class="flex justify-end space-x-3">
          <button type="button" class="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50" @click="$emit('close')">取消</button>
          <button type="submit" class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50" :disabled="submitting || groups.length === 0">
            {{ submitting ? '添加中...' : '添加' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
defineProps({
  show: Boolean,
  loading: Boolean,
  submitting: Boolean,
  groups: { type: Array, default: () => [] },
  form: { type: Object, required: true },
})
defineEmits(['close', 'submit'])
</script>
