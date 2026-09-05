<template>
  <div v-if="visible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900">编辑用户</h3>
        <button type="button" class="text-gray-500 hover:text-gray-700" @click="$emit('close')">
          <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <form @submit.prevent="$emit('submit')">
        <div v-if="loading" class="flex justify-center items-center py-10">
          <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
        </div>
        <div v-else class="space-y-4">
          <div>
            <label for="edit-user-id" class="block text-sm font-medium text-gray-700 mb-2">用户ID</label>
            <input id="edit-user-id" type="text"
                   :value="userId == null ? '' : String(userId)"
                   readonly
                   class="w-full px-4 py-3 rounded-lg border border-gray-200 bg-gray-50 font-mono text-sm text-gray-900 select-all">
          </div>
          <div>
            <label for="edit-username" class="block text-sm font-medium text-gray-700 mb-2">用户名</label>
            <input id="edit-username" v-model="form.username" type="text"
                   class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                   placeholder="输入用户名" required>
          </div>
          <div>
            <label for="edit-email" class="block text-sm font-medium text-gray-700 mb-2">邮箱</label>
            <input id="edit-email" v-model="form.email" type="email"
                   class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                   placeholder="输入邮箱" required>
            <p v-if="errorField === 'email'" data-testid="edit-email-error" class="mt-1 text-sm text-red-600" :data-traceId="errorTraceId || undefined">{{ error }}</p>
          </div>
          <div>
            <div class="flex items-center justify-between mb-2">
              <label for="edit-phone" class="block text-sm font-medium text-gray-700">手机号</label>
              <!-- Anti-Replay-OK: local form clear; persist on 保存 -->
              <button v-if="String(form.phone || '').trim()" type="button" data-testid="unbind-phone-btn"
                      class="text-sm text-red-600 hover:text-red-700" @click="form.phone = ''">解绑</button>
            </div>
            <input id="edit-phone" v-model="form.phone" type="tel"
                   class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                   placeholder="留空并保存即可解绑">
            <p v-if="error && errorField !== 'email'" data-testid="edit-phone-error" class="mt-1 text-sm text-red-600" :data-traceId="errorTraceId || undefined">{{ error }}</p>
          </div>
          <div>
            <label for="edit-password" class="block text-sm font-medium text-gray-700 mb-2">密码</label>
            <input id="edit-password" v-model="form.password" type="password"
                   class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                   placeholder="输入新密码（可选）">
          </div>
          <div class="flex space-x-4">
            <div class="flex items-center">
              <input id="edit-is-superuser" v-model="form.is_superuser" type="checkbox"
                     :disabled="!canSetSuperuser"
                     :title="canSetSuperuser ? '' : '仅系统管理员可设置'"
                     class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded disabled:opacity-50">
              <label for="edit-is-superuser" class="ml-2 block text-sm text-gray-700">超级用户</label>
            </div>
            <div class="flex items-center">
              <input id="edit-is-staff" v-model="form.is_staff" type="checkbox"
                     class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <label for="edit-is-staff" class="ml-2 block text-sm text-gray-700">员工</label>
            </div>
          </div>
          <div class="flex space-x-4">
            <SystemAdminUserRoleFlags
              :form="form"
              prefix="edit"
              @tester-change="$emit('tester-change')"
            />
          </div>
        </div>
        <div class="flex justify-between items-center mt-6">
          <div class="flex flex-col items-start gap-2">
            <button
              v-if="canImpersonate"
              type="button"
              data-testid="impersonate-user-btn"
              class="px-4 py-3 bg-amber-700 text-white rounded-lg hover:bg-amber-800 transition-all duration-300 disabled:opacity-60"
              :disabled="impersonating"
              :aria-busy="impersonating ? 'true' : 'false'"
              @click="$emit('impersonate')"
            >
              {{ impersonating ? '正在登录...' : '以该用户身份登录' }}
            </button>
            <p
              v-if="impersonateError"
              class="text-sm text-red-600"
              :data-trace-id="impersonateErrorTraceId || undefined"
            >{{ impersonateError }}</p>
          </div>
          <div class="flex space-x-3">
            <button type="button"
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300"
                    @click="$emit('close')">
              取消
            </button>
            <button type="submit" :disabled="saving"
                    :aria-busy="saving ? 'true' : 'false'"
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ saving ? '保存中...' : '保存更改' }}
            </button>
          </div>
        </div>
      </form>
    </div>
  </div>
</template>

<script>
import SystemAdminUserRoleFlags from './SystemAdminUserRoleFlags.vue'

export default {
  name: 'SystemAdminEditUserModal',
  components: { SystemAdminUserRoleFlags },
  props: {
    visible: { type: Boolean, required: true },
    form: { type: Object, required: true },
    userId: { type: [String, Number], default: '' },
    loading: { type: Boolean, default: false },
    saving: { type: Boolean, default: false },
    canSetSuperuser: { type: Boolean, default: false },
    error: { type: String, default: '' },
    errorTraceId: { type: String, default: '' },
    errorField: { type: String, default: '' },
    impersonating: { type: Boolean, default: false },
    impersonateError: { type: String, default: '' },
    impersonateErrorTraceId: { type: String, default: '' },
    canImpersonate: { type: Boolean, default: false },
  },
  emits: ['close', 'submit', 'impersonate', 'tester-change'],
}
</script>
