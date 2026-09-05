<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold mb-6">UserData模板管理</h1>

    <div class="flex justify-end items-center mb-6">
      <button @click="addTemplateModalVisible = true"
              class="bg-primary hover:bg-primary-dark text-white px-4 py-2 rounded-lg transition-colors">
        新增模板
      </button>
    </div>

    <!-- 模板列表 -->
    <div class="bg-white rounded-lg shadow-md p-6">
      <h2 class="text-lg font-semibold mb-4">模板列表</h2>
      <div v-if="loadingTemplates" class="text-center py-10">
        <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <p class="mt-2 text-gray-600">加载中...</p>
      </div>
      <div v-else-if="userDataTemplates.length === 0" class="text-center py-10">
        <p class="text-gray-600">暂无模板</p>
      </div>
      <div v-else>
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-gray-100">
              <th class="px-4 py-3 text-left text-sm font-semibold text-gray-700 border-b">模板名称</th>
              <th class="px-4 py-3 text-left text-sm font-semibold text-gray-700 border-b">版本</th>
              <th class="px-4 py-3 text-left text-sm font-semibold text-gray-700 border-b">操作系统版本</th>
              <th class="px-4 py-3 text-left text-sm font-semibold text-gray-700 border-b">状态</th>
              <th class="px-4 py-3 text-left text-sm font-semibold text-gray-700 border-b">创建时间</th>
              <th class="px-4 py-3 text-left text-sm font-semibold text-gray-700 border-b">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="template in userDataTemplates" :key="template.id" class="hover:bg-gray-50">
              <td class="px-4 py-3 border-b">{{ template.name }}</td>
              <td class="px-4 py-3 border-b">{{ template.version }}</td>
              <td class="px-4 py-3 border-b">{{ getOsTypeLabel(template.os_type) }}</td>
              <td class="px-4 py-3 border-b">
                <span v-if="template.is_active" class="px-2 py-1 bg-green-100 text-green-800 text-xs rounded-full">启用</span>
                <span v-else class="px-2 py-1 bg-red-100 text-red-800 text-xs rounded-full">禁用</span>
              </td>
              <td class="px-4 py-3 border-b">{{ formatDate(template.created_at) }}</td>
              <td class="px-4 py-3 border-b">
                <button @click="handleEditTemplate(template)"
                        class="text-blue-600 hover:text-blue-800 mr-3">
                  编辑
                </button>
                <button @click="handleDeleteTemplate(template.id)"
                        class="text-red-600 hover:text-red-800">
                  删除
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 新增模板模态框 -->
    <UserDataTemplateFormModal
      :visible="addTemplateModalVisible"
      title="新增UserData模板"
      form-id="add-template-form"
      content-field-id="add-template-content"
      :form="addForm"
      :loading="addingTemplate"
      :os-groups="USERDATA_OS_GROUPS"
      @close="addTemplateModalVisible = false"
      @submit="handleAddTemplate"
      @generate-script="onGenerateAddScript"
      @add-var="(type) => addVariable(addForm, type)"
      @remove-var="(type, idx) => removeVariable(addForm, type, idx)"
    />

    <!-- 编辑模板模态框 -->
    <UserDataTemplateFormModal
      :visible="editTemplateModalVisible"
      title="编辑UserData模板"
      :form="editForm"
      :loading="editingTemplate"
      :os-groups="USERDATA_OS_GROUPS"
      @close="editTemplateModalVisible = false"
      @submit="handleEditTemplateSubmit"
      @generate-script="onGenerateEditScript"
      @add-var="(type) => addVariable(editForm, type)"
      @remove-var="(type, idx) => removeVariable(editForm, type, idx)"
    />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useUserDataTemplateCrud } from '../composables/useUserDataTemplateCrud.js'
import { generateContainerUserDataScript } from '../composables/useUserDataScriptGenerator.js'
import { formatDate, getOsTypeLabel, USERDATA_OS_GROUPS } from '../utils/userDataTemplate.js'
import { showRequestError, showSuccess } from '../utils/showRequestError.js'
import UserDataTemplateFormModal from './UserDataTemplateFormModal.vue'

const {
  addTemplateModalVisible,
  editTemplateModalVisible,
  loadingTemplates,
  addingTemplate,
  editingTemplate,
  userDataTemplates,
  addForm,
  editForm,
  fetchUserDataTemplates,
  handleAddTemplate,
  handleEditTemplate,
  handleEditTemplateSubmit,
  handleDeleteTemplate,
  addVariable,
  removeVariable,
} = useUserDataTemplateCrud()

onMounted(async () => {
  await fetchUserDataTemplates()
})

function onGenerateAddScript() {
  if (!addForm.os_type) {
    showRequestError('请先选择操作系统')
    return
  }
  addForm.content = generateContainerUserDataScript(addForm)
  showSuccess('容器初始化脚本已生成并填充到模板内容中')
}

function onGenerateEditScript() {
  if (!editForm.os_type) {
    showRequestError('请先选择操作系统')
    return
  }
  editForm.content = generateContainerUserDataScript(editForm)
  showSuccess('容器初始化脚本已生成并填充到模板内容中')
}
</script>

<style scoped>
/* 与 task2app 页面对齐的轻量工具类（不依赖 Tailwind 构建链） */
.fixed { position: fixed; }
.inset-0 { inset: 0; }
.z-50 { z-index: 50; }
.flex { display: flex; }
.inline-block { display: inline-block; }
.block { display: block; }
.flex-1 { flex: 1 1 0%; }
.items-center { align-items: center; }
.justify-end { justify-content: flex-end; }
.justify-between { justify-content: space-between; }
.justify-center { justify-content: center; }
.w-full { width: 100%; }
.w-4 { width: 1rem; }
.w-8 { width: 2rem; }
.h-4 { height: 1rem; }
.h-8 { height: 2rem; }
.max-w-2xl { max-width: 42rem; }
.p-6 { padding: 1.5rem; }
.px-2 { padding-left: 0.5rem; padding-right: 0.5rem; }
.px-3 { padding-left: 0.75rem; padding-right: 0.75rem; }
.px-4 { padding-left: 1rem; padding-right: 1rem; }
.py-1 { padding-top: 0.25rem; padding-bottom: 0.25rem; }
.py-2 { padding-top: 0.5rem; padding-bottom: 0.5rem; }
.py-3 { padding-top: 0.75rem; padding-bottom: 0.75rem; }
.py-10 { padding-top: 2.5rem; padding-bottom: 2.5rem; }
.mb-2 { margin-bottom: 0.5rem; }
.mb-4 { margin-bottom: 1rem; }
.mb-6 { margin-bottom: 1.5rem; }
.mt-2 { margin-top: 0.5rem; }
.ml-2 { margin-left: 0.5rem; }
.mr-3 { margin-right: 0.75rem; }
.mr-4 { margin-right: 1rem; }
.space-x-2 > * + * { margin-left: 0.5rem; }
.space-x-4 > * + * { margin-left: 1rem; }
.space-y-4 > * + * { margin-top: 1rem; }
.rounded { border-radius: 0.25rem; }
.rounded-lg { border-radius: 0.5rem; }
.rounded-full { border-radius: 9999px; }
.border { border-width: 1px; }
.border-b { border-bottom-width: 1px; }
.border-b-2 { border-bottom-width: 2px; }
.border-collapse { border-collapse: collapse; }
.border-gray-300 { border-color: #d1d5db; }
.border-primary { border-color: var(--accent); }
.shadow-md { box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.08); }
.shadow-xl { box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.12); }
.bg-black { background: #000; }
.inset-0.bg-black { background: rgba(0, 0, 0, 0.55); }
.bg-white { background: #fff; }
.bg-gray-100 { background: #f3f4f6; }
.bg-gray-200 { background: #e5e7eb; }
.bg-blue-100 { background: #dbeafe; }
.bg-green-100 { background: #dcfce7; }
.bg-green-600 { background: #16a34a; }
.bg-red-100 { background: #fee2e2; }
.bg-primary { background: linear-gradient(135deg, var(--accent), var(--accent-dim)); }
.text-left { text-align: left; }
.text-center { text-align: center; }
.text-xs { font-size: 0.75rem; }
.text-sm { font-size: 0.875rem; }
.text-lg { font-size: 1.125rem; }
.text-2xl { font-size: 1.5rem; }
.font-bold { font-weight: 700; }
.font-medium { font-weight: 500; }
.font-semibold { font-weight: 600; }
.text-gray-500 { color: #6b7280; }
.text-gray-600 { color: #4b5563; }
.text-gray-700 { color: #374151; }
.text-gray-800 { color: #1f2937; }
.text-gray-900 { color: #111827; }
.text-blue-600 { color: #2563eb; }
.text-red-600 { color: #dc2626; }
.text-red-800 { color: #991b1b; }
.text-green-800 { color: #166534; }
.text-white { color: #fff; }
.text-primary { color: var(--accent); }
.animate-spin { animation: udt-spin 0.8s linear infinite; }
@keyframes udt-spin { to { transform: rotate(360deg); } }
.transition-colors { transition-property: color, background-color, border-color; transition-duration: 0.15s; }
.transition-all { transition-property: all; transition-duration: 0.3s; }
.duration-300 { transition-duration: 0.3s; }
.hover\:bg-gray-50:hover { background: #f9fafb; }
.hover\:bg-gray-300:hover { background: #d1d5db; }
.hover\:bg-blue-200:hover { background: #bfdbfe; }
.hover\:bg-red-200:hover { background: #fecaca; }
.hover\:bg-green-700:hover { background: #15803d; }
.hover\:bg-primary-dark:hover { filter: brightness(0.92); }
.hover\:text-blue-800:hover { color: #1e40af; }
.hover\:text-red-800:hover { color: #991b1b; }
.hover\:text-gray-700:hover { color: #374151; }
.focus\:outline-none:focus { outline: none; }
.focus\:ring-2:focus { box-shadow: 0 0 0 2px var(--accent); }
.focus\:ring-primary:focus { box-shadow: 0 0 0 2px var(--accent); }
.focus\:border-transparent:focus { border-color: transparent; }
</style>
