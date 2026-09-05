<template>
  <div data-alias="view-system-admin-license-agreement" id="license-agreement" class="p-6">
    <div class="mb-6">
      <div class="flex justify-between items-center">
        <h2 class="text-xl font-semibold text-gray-900">软件许可及服务协议管理</h2>
        <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="openCreateModal">
          创建协议
        </button>
      </div>
      <p class="text-sm text-gray-500 mt-2">
        若新版本勾选「含实质性变更」，老用户登录后需再次确认同意；同意记录会写入数据库供合规审计（亦可在下方按用户 ID 查询）。
      </p>
      <div
        v-if="!paymentTermsPublished && !paymentTermsChecking"
        class="mt-4 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800"
        data-testid="payment-terms-missing-warning"
      >
        <span class="font-medium">支付服务条款未发布：</span>用户支付时将跳过条款签署（fail-open）。请在「支付服务条款」页签创建并勾选「立即生效」。
      </div>
      <div class="mt-4 flex gap-2 border-b border-gray-200">
        <button
          v-for="tab in DOCUMENT_KIND_TABS"
          :key="tab.value"
          type="button"
          class="px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors"
          :class="documentKindFilter === tab.value
            ? 'border-primary text-primary'
            : 'border-transparent text-gray-500 hover:text-gray-800'"
          @click="setDocumentKindFilter(tab.value)"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <div class="mb-8 rounded-lg border border-gray-200 bg-gray-50/80 p-4">
      <h3 class="text-sm font-medium text-gray-800 mb-3">用户同意记录查询</h3>
      <div class="flex flex-wrap items-end gap-3">
        <div>
          <label for="consent-user-id" class="block text-xs text-gray-500 mb-1">用户 ID</label>
          <input
            id="consent-user-id"
            v-model="consentUserId"
            type="text"
            class="w-56 px-3 py-2 rounded-lg border border-gray-300 text-sm"
            placeholder="数字 ID"
            @keydown.enter="fetchUserConsents"
          >
        </div>
        <button
          type="button"
          class="px-4 py-2 bg-gray-800 text-white text-sm rounded-lg hover:bg-gray-700 disabled:opacity-50"
          :disabled="consentLoading"
          @click="fetchUserConsents"
        >
          {{ consentLoading ? '查询中...' : '查询' }}
        </button>
      </div>
      <p v-if="consentError" class="text-sm text-amber-800 mt-2">{{ consentError }}</p>
      <div v-if="consentRows.length" class="mt-4 overflow-x-auto">
        <table class="min-w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 border-b">
              <th class="pr-3 py-2">同意时间</th>
              <th class="pr-3 py-2">场景</th>
              <th class="pr-3 py-2">协议版本</th>
              <th class="pr-3 py-2">实质性变更</th>
              <th class="py-2">IP</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in consentRows" :key="row.id" class="border-b border-gray-100">
              <td class="pr-3 py-2 whitespace-nowrap">{{ formatDate(row.consented_at) }}</td>
              <td class="pr-3 py-2">{{ consentContextLabel(row.context) }}</td>
              <td class="pr-3 py-2">{{ row.license_version }} / {{ row.license_title }}</td>
              <td class="pr-3 py-2">{{ row.is_material_change ? '是' : '否' }}</td>
              <td class="py-2 text-gray-500">{{ row.client_ip || '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else-if="consentSearched && !consentError" class="mt-2 text-sm text-gray-500">无记录</p>
    </div>

    <div v-if="createModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">创建协议</h3>
          <button @click="createModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>

        <form @submit.prevent="handleCreate">
          <div class="space-y-4">
            <div>
              <label for="create-kind" class="block text-sm font-medium text-gray-700 mb-2">文档类型</label>
              <select
                id="create-kind"
                v-model="createForm.document_kind"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option v-for="tab in DOCUMENT_KIND_TABS" :key="tab.value" :value="tab.value">{{ tab.label }}</option>
              </select>
            </div>
            <div>
              <label for="create-title" class="block text-sm font-medium text-gray-700 mb-2">协议标题</label>
              <input type="text" id="create-title" v-model="createForm.title"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="输入协议标题" required>
            </div>
            <div>
              <label for="create-version" class="block text-sm font-medium text-gray-700 mb-2">版本号</label>
              <input type="text" id="create-version" v-model="createForm.version"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="例如：1.0.0" required>
            </div>
            <div>
              <label for="create-content" class="block text-sm font-medium text-gray-700 mb-2">协议内容</label>
              <textarea id="create-content" v-model="createForm.content"
                        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                        placeholder="输入协议内容" rows="10"></textarea>
            </div>
            <div class="flex items-center">
              <input type="checkbox" id="create-is-active" v-model="createForm.is_active"
                     class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <label for="create-is-active" class="ml-2 block text-sm text-gray-700">立即生效</label>
            </div>
            <div v-if="createForm.document_kind === 'service'" class="flex items-start gap-2">
              <input
                id="create-material"
                v-model="createForm.is_material_change"
                type="checkbox"
                class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded mt-0.5"
              >
              <label for="create-material" class="text-sm text-gray-700">
                含实质性变更（老用户须登录后再次确认）
              </label>
            </div>
          </div>
          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="createModalVisible = false"
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="creating"
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ creating ? '创建中...' : '创建协议' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="editModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">编辑协议</h3>
          <button @click="editModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        <form @submit.prevent="handleEdit">
          <div class="space-y-4">
            <div>
              <label for="edit-kind" class="block text-sm font-medium text-gray-700 mb-2">文档类型</label>
              <select
                id="edit-kind"
                v-model="editForm.document_kind"
                class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option v-for="tab in DOCUMENT_KIND_TABS" :key="tab.value" :value="tab.value">{{ tab.label }}</option>
              </select>
            </div>
            <div>
              <label for="edit-title" class="block text-sm font-medium text-gray-700 mb-2">协议标题</label>
              <input type="text" id="edit-title" v-model="editForm.title"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="输入协议标题" required>
            </div>
            <div>
              <label for="edit-version" class="block text-sm font-medium text-gray-700 mb-2">版本号</label>
              <input type="text" id="edit-version" v-model="editForm.version"
                     class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                     placeholder="例如：1.0.0" required>
            </div>
            <div>
              <label for="edit-content" class="block text-sm font-medium text-gray-700 mb-2">协议内容</label>
              <textarea id="edit-content" v-model="editForm.content"
                        class="w-full px-4 py-3 rounded-lg border border-gray-300 focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent transition-all duration-300"
                        placeholder="输入协议内容" rows="10"></textarea>
            </div>
            <div class="flex items-center">
              <input type="checkbox" id="edit-is-active" v-model="editForm.is_active"
                     class="w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <label for="edit-is-active" class="ml-2 block text-sm text-gray-700">生效</label>
            </div>
            <div v-if="editForm.document_kind === 'service'" class="flex items-start gap-2">
              <input type="checkbox" id="edit-is-material" v-model="editForm.is_material_change"
                     class="mt-0.5 w-4 h-4 text-primary focus:ring-primary border-gray-300 rounded">
              <div>
                <label for="edit-is-material" class="block text-sm text-gray-700">相对上一版为实质性变更</label>
                <p class="text-xs text-gray-500 mt-0.5">面向已注册用户的再次确认需求，详见创建协议说明。</p>
              </div>
            </div>
          </div>
          <div class="flex justify-end space-x-3 mt-6">
            <button type="button" @click="editModalVisible = false"
                    class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
              取消
            </button>
            <button type="submit" :disabled="editing"
                    class="px-4 py-3 bg-primary text-white rounded-lg hover:bg-primary/90 transition-all duration-300">
              {{ editing ? '保存中...' : '保存修改' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="deleteModalVisible" class="app-modal-overlay bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-lg font-semibold text-gray-900">删除协议</h3>
          <button @click="deleteModalVisible = false" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
        <p class="text-gray-600 mb-6">确定要删除协议"{{ deleteTarget.title }}"吗？此操作不可恢复。</p>
        <div class="flex justify-end space-x-3">
          <button type="button" @click="deleteModalVisible = false"
                  class="px-4 py-3 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-all duration-300">
            取消
          </button>
          <button type="button" :disabled="deleting" @click="handleDelete"
                  class="px-4 py-3 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-all duration-300">
            {{ deleting ? '删除中...' : '确认删除' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center items-center py-20">
      <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-primary"></div>
      <span class="ml-3 text-gray-600">加载中...</span>
    </div>

    <div v-else>
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">标题</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">类型</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">版本</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">实质性变更</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-if="agreements.length === 0">
              <td colspan="8" class="px-6 py-12 text-center">
                <div class="flex flex-col items-center justify-center">
                  <svg class="w-16 h-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                  </svg>
                  <h3 class="text-lg font-medium text-gray-900 mb-1">暂无{{ documentKindLabel(documentKindFilter) }}</h3>
                  <p class="text-gray-500 mb-6">请点击"创建协议"按钮创建第一条文档。</p>
                  <button class="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors" @click="openCreateModal">
                    创建协议
                  </button>
                </div>
              </td>
            </tr>
            <tr v-for="agreement in agreements" :key="agreement.id">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ agreement.id }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ agreement.title }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ documentKindLabel(agreement.document_kind) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">v{{ agreement.version }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <span :class="agreement.is_active ? 'text-green-600' : 'text-gray-400'">
                  {{ agreement.is_active ? '生效中' : '未生效' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <span :class="agreement.is_material_change ? 'text-amber-700' : 'text-gray-500'">
                  {{ agreement.is_material_change ? '是' : '否' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ formatDate(agreement.created_at) }}</td>
              <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
                <button class="text-primary hover:text-primary/90 mr-3" @click="openEditModal(agreement)">
                  编辑
                </button>
                <button class="text-red-600 hover:text-red-800" @click="openDeleteModal(agreement)">
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useSystemAdminLicenseAgreement } from '../composables/useSystemAdminLicenseAgreement.js'

const {
  DOCUMENT_KIND_TABS,
  documentKindFilter,
  setDocumentKindFilter,
  documentKindLabel,
  createModalVisible,
  editModalVisible,
  deleteModalVisible,
  loading,
  creating,
  editing,
  deleting,
  createForm,
  editForm,
  consentUserId,
  consentRows,
  consentLoading,
  consentError,
  consentSearched,
  deleteTarget,
  agreements,
  paymentTermsPublished,
  paymentTermsChecking,
  formatDate,
  consentContextLabel,
  openCreateModal,
  fetchUserConsents,
  handleCreate,
  openEditModal,
  handleEdit,
  openDeleteModal,
  handleDelete
} = useSystemAdminLicenseAgreement()
</script>
<style scoped>
</style>
