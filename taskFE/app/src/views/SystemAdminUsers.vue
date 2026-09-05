<template>
  <div data-alias="view-system-admin-users" id="users" class="p-6">
    <!-- 顶部固定区域 -->
    <div class="mb-6">
      <div class="flex justify-between items-center mb-4">
        <div>
          <h2 class="text-xl font-semibold text-gray-900">用户列表</h2>
          <p v-if="isUserListTab && totalUsers !== null" class="text-sm text-gray-500 mt-1">共 {{ totalUsers }} 个用户</p>
        </div>
        <div class="flex space-x-3">
          <button class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors" @click="openEmailInviteModal">
            📧 发送邮箱邀请
          </button>
          <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="openAddUserModal">
            添加用户
          </button>
        </div>
      </div>
      <!-- 搜索栏：用户 tab 专用 -->
      <div v-if="isUserListTab" class="flex items-center space-x-3 mb-4">
        <div class="relative flex-1 max-w-md">
          <input
            type="text"
            v-model="searchQuery"
            placeholder="搜索邮箱、ID、手机号..."
            class="w-full pl-10 pr-4 py-2 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
            @keyup.enter="handleSearch"
          />
          <svg class="absolute left-3 top-2.5 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
          </svg>
        </div>
        <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="handleSearch">
          搜索
        </button>
        <button v-if="searchQuery" class="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors" @click="clearSearch">
          清除
        </button>
      </div>
      <!-- 标签页切换 -->
      <SystemAdminUsersTabs :active-tab="activeTab" @switch="switchTab" />
      <!-- OPT-20260824-091: 活跃 Tab 搜索结果含归档用户（后端检索不再套用 is_archived=false）时提示 -->
      <div
        v-if="activeTabHasArchivedResults"
        data-testid="archived-in-active-results-hint"
        class="mt-3 px-4 py-3 rounded-lg bg-amber-50 border border-amber-200 text-sm text-amber-800"
      >
        ⚠️ 搜索命中了已归档用户：其手机号/邮箱标识已被回收，行内已标「已归档」。
        如需查看全部归档用户，请切换到「已归档」Tab。
      </div>
    </div>

      <!-- 添加用户模态框 -->
      <div v-if="addUserModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
          <div class="flex justify-between items-center mb-4">
            <h3 class="text-lg font-semibold text-gray-900">添加用户</h3>
            <button @click="addUserModalVisible = false" class="text-gray-500 hover:text-gray-700">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>
          
          <form @submit.prevent="handleAddUser">
            <div class="space-y-4">
              <div>
                <label for="add-username" class="block text-sm font-medium text-gray-700 mb-2">用户名</label>
                <input type="text" id="add-username" v-model="addUserForm.username" 
                       class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                       placeholder="输入用户名" required>
              </div>
              
              <div>
                <label for="add-email" class="block text-sm font-medium text-gray-700 mb-2">邮箱</label>
                <input type="email" id="add-email" v-model="addUserForm.email" 
                       class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                       placeholder="输入邮箱" required>
                <p v-if="addUserErrorField === 'email'" data-testid="add-email-error" class="mt-1 text-sm text-red-600" :data-traceId="addUserErrorTraceId || undefined">{{ addUserError }}</p>
              </div>
              
              <div>
                <label for="add-phone" class="block text-sm font-medium text-gray-700 mb-2">手机号</label>
                <input type="tel" id="add-phone" v-model="addUserForm.phone" 
                       class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                       placeholder="输入手机号">
                <p v-if="addUserError && addUserErrorField !== 'email'" data-testid="add-phone-error" class="mt-1 text-sm text-red-600" :data-traceId="addUserErrorTraceId || undefined">{{ addUserError }}</p>
              </div>
              
              <div>
                <label for="add-password" class="block text-sm font-medium text-gray-700 mb-2">密码</label>
                <input type="password" id="add-password" v-model="addUserForm.password" 
                       class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300" 
                       placeholder="输入密码" required>
              </div>
              
              <div class="flex space-x-4">
                <div v-if="canSetSuperuser" class="flex items-center">
                  <input type="checkbox" id="add-is-superuser" v-model="addUserForm.is_superuser"
                         class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
                  <label for="add-is-superuser" class="ml-2 block text-sm text-gray-700">超级用户</label>
                </div>

                <div class="flex items-center">
                  <input type="checkbox" id="add-is-staff" v-model="addUserForm.is_staff"
                         class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
                  <label for="add-is-staff" class="ml-2 block text-sm text-gray-700">员工</label>
                </div>
              </div>

              <div class="flex space-x-4">
                <SystemAdminUserRoleFlags
                  :form="addUserForm"
                  prefix="add"
                  @tester-change="onTesterFlagChange(addUserForm)"
                />
              </div>
            </div>
            
            <div class="flex justify-end space-x-3 mt-6">
              <button type="button" @click="addUserModalVisible = false" 
                      class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
                取消
              </button>
              <button type="submit" :disabled="addingUser" :aria-busy="addingUser ? 'true' : 'false'" 
                      class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
                {{ addingUser ? '添加中...' : '添加用户' }}
              </button>
            </div>
          </form>
        </div>
      </div>

      <SystemAdminEditUserModal
        :visible="editUserModalVisible"
        :form="editUserForm"
        :user-id="editUserId"
        :loading="loadingUser"
        :saving="editingUser"
        :can-set-superuser="canSetSuperuser"
        :error="editUserError"
        :error-trace-id="editUserErrorTraceId"
        :error-field="editUserErrorField"
        :impersonating="impersonating"
        :impersonate-error="impersonateError"
        :impersonate-error-trace-id="impersonateErrorTraceId"
        :can-impersonate="hasPlatformPerm('user:impersonate')"
        @close="editUserModalVisible = false"
        @submit="handleEditUser"
        @impersonate="handleImpersonateUser"
        @tester-change="onTesterFlagChange(editUserForm)"
      />

      <!-- 删除用户模态框 -->
      <div v-if="deleteUserModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
          <div class="flex justify-between items-center mb-4">
            <h3 class="text-lg font-semibold text-gray-900">删除用户</h3>
            <button @click="deleteUserModalVisible = false" class="text-gray-500 hover:text-gray-700">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>

          <p class="text-gray-700 mb-6">您确定要禁用用户 "{{ deleteUserName }}" 吗？此操作不会删除数据，仅将账号设为不可登录。</p>
          
          <div class="flex justify-end space-x-3">
            <button type="button" @click="deleteUserModalVisible = false" 
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="button" :disabled="deletingUser" @click="handleDeleteUser" 
                    class="px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-all duration-300">
              {{ deletingUser ? '删除中...' : '确认删除' }}
            </button>
          </div>
        </div>
      </div>

    <SystemAdminTenantsPanel v-if="activeTab === 'tenants'" />
    <SystemAdminReferralApplicationsPanel v-else-if="activeTab === 'referral-apps'" />

    <!-- 用户列表主体 -->
    <div v-else>
      <div v-if="loadingUsers" class="flex justify-center items-center py-20">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
        <span class="ml-3 text-gray-600">加载用户列表中...</span>
      </div>

      <div v-else-if="loadError" class="flex flex-col items-center justify-center py-16" :data-traceId="loadErrorTraceId || undefined">
        <svg class="w-16 h-16 text-red-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
        </svg>
        <h3 class="text-lg font-medium text-gray-900 mb-2">加载失败</h3>
        <p class="text-gray-500 mb-6 text-center max-w-md">{{ loadError }}</p>
        <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="refreshUsers">
          重新加载
        </button>
      </div>

      <div v-else class="overflow-x-auto">
        <p
          v-if="impersonateError && !editUserModalVisible"
          class="text-sm text-red-600 mb-3"
          data-testid="user-row-impersonate-error"
          :data-trace-id="impersonateErrorTraceId || undefined"
        >{{ impersonateError }}</p>
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">邮箱</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">所属租户公司</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">手机号</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">登录方式</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">推荐人</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">注册时间</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">最后活跃时间</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">角色</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">是否获得分账资格</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
            </tr>
            <tr data-testid="user-list-column-filters" data-alias="SystemAdminUsersFiltersRow" class="bg-gray-50/60">
              <SystemAdminUsersFilters
                :filters="columnFilters"
                @update:filter="onColumnFilterUpdate"
                @reset="resetColumnFilters"
              />
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-if="users.length === 0">
              <td colspan="12" class="px-6 py-12 text-center">
                <div class="flex flex-col items-center justify-center">
                  <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
                  </svg>
                  <h3 class="text-lg font-medium text-gray-900 mb-1">暂无用户</h3>
                  <p class="text-gray-500 mb-6">系统中还没有任何用户，请点击"添加用户"按钮创建第一个用户。</p>
                  <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="openAddUserModal">
                    添加用户
                  </button>
                </div>
              </td>
            </tr>
            <UserListRow
              v-for="user in users"
              :key="user.id"
              :user="user"
              :can-impersonate="hasPlatformPerm('user:impersonate')"
              @edit="openEditUserModal"
              @delete="openDeleteUserModal"
              @recharge="openRechargeDrawer"
              @kyc="openKycDrawer"
              @referral="openReferralPerformanceDrawer"
              @archive="handleArchiveUser"
              @unarchive="handleUnarchiveUser"
              @impersonate="handleImpersonateRow"
            />
          </tbody>
        </table>
      </div>

      <SystemAdminUserRechargeDrawer
        :visible="rechargeDrawerVisible"
        :user-id="rechargeDrawerUserId"
        @close="rechargeDrawerVisible = false"
      />

      <SystemAdminUserKycDrawer
        :visible="kycDrawerVisible"
        :user-id="kycDrawerUserId"
        @close="kycDrawerVisible = false"
      />

      <SystemAdminReferralPerformanceDrawer
        :visible="referralPerformanceVisible"
        :user-id="referralPerformanceUserId"
        @close="referralPerformanceVisible = false"
      />

      <div v-if="totalUsers !== null && totalUsers > 0" class="flex justify-between items-center mt-4 pt-4 border-t border-gray-200">
        <span class="text-sm text-gray-500">
          第 {{ Math.floor(currentOffset / pageLimit) + 1 }} 页，共 {{ Math.ceil(totalUsers / pageLimit) }} 页
        </span>
        <div class="flex space-x-3">
          <button
            :disabled="currentOffset === 0"
            class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="currentOffset === 0 ? 'bg-gray-100 text-gray-400 cursor-not-allowed' : 'bg-gray-200 text-gray-700 hover:bg-gray-300'"
            @click="goToPrevPage"
          >
            上一页
          </button>
          <button
            :disabled="currentOffset + pageLimit >= totalUsers"
            class="px-4 py-2 rounded-lg text-sm font-medium transition-colors"
            :class="currentOffset + pageLimit >= totalUsers ? 'bg-gray-100 text-gray-400 cursor-not-allowed' : 'bg-primary text-white hover:bg-primary/90'"
            @click="goToNextPage"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <SystemAdminEmailInvitationsSection
      ref="emailInviteSection"
      :show-list="isUserListTab"
    />

    <ImpersonateReasonModal
      v-if="reasonModalVisible"
      :reason="impersonateReason"
      :pending="impersonating"
      :error="impersonateError"
      :error-trace-id="impersonateErrorTraceId"
      @update:reason="impersonateReason = $event"
      @confirm="confirmImpersonateReason"
      @cancel="reasonModalVisible = false"
    />
  </div>
</template>

<script setup>
/* @alias:view-system-admin-users */
import UserListRow from '../components/UserListRow.vue'
import SystemAdminUsersFilters from '../components/SystemAdminUsersFilters.vue'
import SystemAdminUserRoleFlags from '../components/SystemAdminUserRoleFlags.vue'
import SystemAdminEditUserModal from '../components/SystemAdminEditUserModal.vue'
import SystemAdminUserRechargeDrawer from '../components/SystemAdminUserRechargeDrawer.vue'
import SystemAdminUserKycDrawer from '../components/SystemAdminUserKycDrawer.vue'
import SystemAdminReferralPerformanceDrawer from '../components/SystemAdminReferralPerformanceDrawer.vue'
import SystemAdminUsersTabs from '../components/SystemAdminUsersTabs.vue'
import SystemAdminReferralApplicationsPanel from '../components/SystemAdminReferralApplicationsPanel.vue'
import SystemAdminTenantsPanel from '../components/SystemAdminTenantsPanel.vue'
import SystemAdminEmailInvitationsSection from '../components/SystemAdminEmailInvitationsSection.vue'
import { useSystemAdminUsers } from '../composables/useSystemAdminUsers.js'
import { usePermissions } from '../composables/usePermissions.js'
import { useImpersonateUser } from '../composables/useImpersonateUser.js'
import { canSetSuperuserFlag } from '../composables/adminUserSuperuserGate.js'
import ImpersonateReasonModal from '../components/ImpersonateReasonModal.vue'
import { computed, ref } from 'vue'

const {
  addUserModalVisible, editUserModalVisible, deleteUserModalVisible,
  rechargeDrawerVisible, rechargeDrawerUserId, openRechargeDrawer,
  kycDrawerVisible, kycDrawerUserId, openKycDrawer,
  loadingUsers, loadingUser, addingUser, editingUser, deletingUser, archivingUser,
  addUserForm, editUserForm, deleteUserName, editUserError, editUserErrorTraceId, editUserErrorField,
  addUserError, addUserErrorTraceId, addUserErrorField,
  users, totalUsers, loadError, loadErrorTraceId, currentOffset, pageLimit,
  searchQuery, activeTab, isUserListTab, activeTabHasArchivedResults, editUserId, columnFilters,
  refreshUsers, goToPrevPage, goToNextPage,
  switchTab, handleSearch, clearSearch,
  onColumnFilterUpdate, resetColumnFilters,
  handleArchiveUser, handleUnarchiveUser,
  openAddUserModal, handleAddUser, openEditUserModal, handleEditUser,
  openDeleteUserModal, handleDeleteUser, onTesterFlagChange
} = useSystemAdminUsers()

const { hasPlatformPerm, isPlatformRole } = usePermissions()
const canSetSuperuser = computed(() => canSetSuperuserFlag(isPlatformRole, hasPlatformPerm))
const {
  impersonating,
  impersonateError,
  impersonateErrorTraceId,
  impersonateUser,
} = useImpersonateUser()
function handleImpersonateUser() {
  openImpersonateReason(editUserId.value)
}
function handleImpersonateRow(user) {
  openImpersonateReason(user?.id != null ? String(user.id) : '')
}
const reasonModalVisible = ref(false)
const impersonateReason = ref('')
const reasonTargetUserId = ref('')
function openImpersonateReason(uid) {
  reasonTargetUserId.value = String(uid || '').trim()
  impersonateReason.value = ''
  reasonModalVisible.value = true
}
function confirmImpersonateReason() {
  return impersonateUser(reasonTargetUserId.value, impersonateReason.value)
}
const referralPerformanceVisible = ref(false)
const referralPerformanceUserId = ref('')
const emailInviteSection = ref(null)
function openReferralPerformanceDrawer(user) {
  referralPerformanceUserId.value = user?.id != null ? String(user.id) : ''
  referralPerformanceVisible.value = true
}
function openEmailInviteModal() {
  emailInviteSection.value?.openEmailInviteModal?.()
}
</script>
