<template>
  <div v-if="show" class="app-modal-overlay bg-black/60 flex items-center justify-center z-[60]">
    <div class="bg-white rounded-xl p-6 max-w-md w-full mx-4">
      <h3 class="text-lg font-semibold mb-4">添加可访问该空间的成员</h3>
      <form @submit.prevent="$emit('submit')">
        <div class="mb-4">
          <label for="addMemberSelect" class="block text-sm font-medium text-gray-700 mb-1">选择成员</label>
          <div v-if="loading" class="py-2 text-sm text-gray-500">加载可选成员中...</div>
          <select
            v-else
            id="addMemberSelect"
            v-model="form.userId"
            class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-primary focus:border-primary"
            required
          >
            <option value="">请选择成员</option>
            <option
              v-for="member in members"
              :key="member.company_member_id"
              :value="member.company_member_id"
            >
              {{ labelFn(member) }}-{{ member.member_name || '未知用户' }}
            </option>
          </select>
          <p v-if="!loading && members.length === 0" class="mt-1 text-sm text-gray-500">所有成员已拥有访问权限</p>
        </div>
        <div class="mb-4">
          <label for="addMemberRole" class="block text-sm font-medium text-gray-700 mb-1">权限</label>
          <select
            id="addMemberRole"
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
          <button type="submit" class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50" :disabled="submitting || members.length === 0">
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
  members: { type: Array, default: () => [] },
  form: { type: Object, required: true },
  labelFn: { type: Function, default: (m) => m?.member_name || '' },
})
defineEmits(['close', 'submit'])
</script>
