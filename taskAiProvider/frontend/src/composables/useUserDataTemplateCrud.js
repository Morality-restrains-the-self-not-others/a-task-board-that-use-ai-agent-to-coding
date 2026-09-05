import { ref, reactive, watch } from 'vue'
import { api } from '../api'
import { generateTemplateContent } from './useUserDataScriptGenerator.js'
import { showRequestError, showSuccess, showConfirm } from '../utils/showRequestError.js'

export function useUserDataTemplateCrud() {
  const addTemplateModalVisible = ref(false)
  const editTemplateModalVisible = ref(false)

  const loadingTemplates = ref(false)
  const addingTemplate = ref(false)
  const editingTemplate = ref(false)

  const userDataTemplates = ref([])

  const addForm = reactive({
    name: '',
    version: '',
    os_type: '',
    userdata_variables: [
      { name: 'CONTAINER_IMAGE' }
    ],
    container_variables: [],
    content: '',
    auto_verify_script: '',
    is_active: true
  })

  const editForm = reactive({
    id: '',
    name: '',
    version: '',
    os_type: '',
    userdata_variables: [
      { name: 'CONTAINER_IMAGE' }
    ],
    container_variables: [],
    content: '',
    auto_verify_script: '',
    is_active: true
  })

  // Auto-generate script content when form variables change
  watch(() => [addForm.container_variables, addForm.userdata_variables, addForm.os_type], () => {
    if (addForm.os_type) {
      addForm.content = generateTemplateContent(addForm)
    }
  }, { deep: true })

  watch(() => [editForm.container_variables, editForm.userdata_variables, editForm.os_type], () => {
    if (editForm.os_type) {
      editForm.content = generateTemplateContent(editForm)
    }
  }, { deep: true })

  async function fetchUserDataTemplates() {
    try {
      loadingTemplates.value = true
      const data = await api('/api/admin/userdata-templates/')
      userDataTemplates.value = Array.isArray(data) ? data : []
    } catch (error) {
      console.error('获取UserData模板列表失败:', error)
      showRequestError('获取UserData模板列表失败: ' + (error.message || '网络错误'), error)
      userDataTemplates.value = []
    } finally {
      loadingTemplates.value = false
    }
  }

  function resetAddForm() {
    addForm.name = ''
    addForm.version = ''
    addForm.os_type = ''
    addForm.userdata_variables = []
    addForm.container_variables = []
    addForm.content = ''
    addForm.auto_verify_script = ''
    addForm.is_active = true
  }

  async function handleAddTemplate() {
    try {
      addingTemplate.value = true

      const userdataVariables = addForm.userdata_variables.map(v => ({
        name: v.name,
        value: v.value || ''
      }))
      const containerVariables = addForm.container_variables.map(v => ({
        name: v.name,
        value: v.value || ''
      }))

      const created = await api('/api/admin/userdata-templates/', {
        method: 'POST',
        body: JSON.stringify({
          name: addForm.name,
          version: addForm.version,
          os_type: addForm.os_type,
          variables: userdataVariables,
          container_variables: containerVariables,
          content: addForm.content,
          auto_verify_script: addForm.auto_verify_script,
          is_active: addForm.is_active
        })
      })
      userDataTemplates.value.push(created)
      addTemplateModalVisible.value = false
      resetAddForm()
      showSuccess('添加UserData模板成功')
    } catch (error) {
      console.error('添加UserData模板失败:', error)
      showRequestError('添加UserData模板失败', error)
    } finally {
      addingTemplate.value = false
    }
  }

  function handleEditTemplate(template) {
    editForm.id = template.id
    editForm.name = template.name
    editForm.version = template.version
    editForm.os_type = template.os_type
    editForm.userdata_variables = [...(template.variables || [])]
    editForm.container_variables = [...(template.container_variables || [])]
    editForm.content = template.content
    editForm.auto_verify_script = template.auto_verify_script
    editForm.is_active = template.is_active
    editTemplateModalVisible.value = true
  }

  async function handleEditTemplateSubmit() {
    try {
      editingTemplate.value = true

      const userdataVariables = editForm.userdata_variables.map(v => ({
        name: v.name,
        value: v.value || ''
      }))
      const containerVariables = editForm.container_variables.map(v => ({
        name: v.name,
        value: v.value || ''
      }))

      const updated = await api(`/api/admin/userdata-templates/${editForm.id}/`, {
        method: 'PUT',
        body: JSON.stringify({
          name: editForm.name,
          version: editForm.version,
          os_type: editForm.os_type,
          variables: userdataVariables,
          container_variables: containerVariables,
          content: editForm.content,
          auto_verify_script: editForm.auto_verify_script,
          is_active: editForm.is_active
        })
      })
      const index = userDataTemplates.value.findIndex(t => String(t.id) === String(editForm.id))
      if (index !== -1) {
        userDataTemplates.value[index] = updated
      }
      editTemplateModalVisible.value = false
      showSuccess('编辑UserData模板成功')
    } catch (error) {
      console.error('编辑UserData模板失败:', error)
      showRequestError('编辑UserData模板失败', error)
    } finally {
      editingTemplate.value = false
    }
  }

  async function handleDeleteTemplate(templateId) {
    const confirmed = await showConfirm('确定要删除这个模板吗？')
    if (!confirmed) {
      return
    }
    const idKey = String(templateId)
    try {
      await api(`/api/admin/userdata-templates/${idKey}/`, {
        method: 'DELETE'
      })
      userDataTemplates.value = userDataTemplates.value.filter(t => String(t.id) !== idKey)
      alert('删除UserData模板成功')
    } catch (error) {
      console.error('删除UserData模板失败:', error)
      showRequestError('删除UserData模板失败', error)
    }
  }

  // Parameterized variable list helpers — replaces 8 duplicative functions
  function addVariable(form, type) {
    form[type].push({ name: '' })
  }

  function removeVariable(form, type, index) {
    form[type].splice(index, 1)
  }

  return {
    // State
    addTemplateModalVisible,
    editTemplateModalVisible,
    loadingTemplates,
    addingTemplate,
    editingTemplate,
    userDataTemplates,
    addForm,
    editForm,
    // Functions
    fetchUserDataTemplates,
    resetAddForm,
    handleAddTemplate,
    handleEditTemplate,
    handleEditTemplateSubmit,
    handleDeleteTemplate,
    addVariable,
    removeVariable,
  }
}
