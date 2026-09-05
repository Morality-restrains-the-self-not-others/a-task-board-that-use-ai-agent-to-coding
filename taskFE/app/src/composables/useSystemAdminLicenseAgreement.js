import { ref, onMounted } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { showRequestError } from '../utils/requestErrorDisplay.js'
import { fetchCurrentPaymentTerms } from '../utils/paymentTermsConsent.js'

export const DOCUMENT_KIND_SERVICE = 'service'
export const DOCUMENT_KIND_RECHARGE_POINTS = 'recharge_cents'

export const DOCUMENT_KIND_TABS = [
  { value: DOCUMENT_KIND_SERVICE, label: '服务协议' },
  { value: DOCUMENT_KIND_RECHARGE_POINTS, label: '支付服务条款' }
]

export function useSystemAdminLicenseAgreement() {
  const documentKindFilter = ref(DOCUMENT_KIND_SERVICE)

  const createModalVisible = ref(false)
  const editModalVisible = ref(false)
  const deleteModalVisible = ref(false)

  const loading = ref(false)
  const creating = ref(false)
  const editing = ref(false)
  const deleting = ref(false)

  const emptyForm = () => ({
    title: '',
    version: '',
    content: '',
    document_kind: documentKindFilter.value,
    is_active: true,
    is_material_change: false
  })

  const createForm = ref(emptyForm())
  const editForm = ref({
    id: '',
    title: '',
    version: '',
    content: '',
    document_kind: DOCUMENT_KIND_SERVICE,
    is_active: true,
    is_material_change: false
  })

  const consentUserId = ref('')
  const consentRows = ref([])
  const consentLoading = ref(false)
  const consentError = ref('')
  const consentSearched = ref(false)

  const deleteTarget = ref({ id: '', title: '' })
  const agreements = ref([])

  // OPT-20260819-026：无生效支付服务条款时运营告警。
  // 服务端门禁 fail-open（仅结构化日志 payment_terms_consent_skipped_no_active），
  // 此处让管理员在法律文档页直接看到缺口，避免用户「不签即付」长期无人发现。
  const paymentTermsPublished = ref(true)
  const paymentTermsChecking = ref(false)

  const checkPaymentTermsPublished = async () => {
    paymentTermsChecking.value = true
    try {
      const doc = await fetchCurrentPaymentTerms()
      paymentTermsPublished.value = !!doc
    } catch (error) {
      // fetchCurrentPaymentTerms 对 404 返回 null；其余网络错误不触发告警，避免误报
      paymentTermsPublished.value = true
    } finally {
      paymentTermsChecking.value = false
    }
  }

  const formatDate = (dateString) => {
    if (!dateString) return '-'
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const CONSENT_CONTEXT_LABELS = {
    register_phone: '手机号注册',
    register_email: '邮箱注册',
    login_password: '密码登录',
    post_login_reconsent: '登录后实质性更新确认'
  }

  const consentContextLabel = (code) => CONSENT_CONTEXT_LABELS[code] || String(code || '—')

  const documentKindLabel = (kind) => {
    const hit = DOCUMENT_KIND_TABS.find((t) => t.value === kind)
    return hit ? hit.label : String(kind || '—')
  }

  const setDocumentKindFilter = (kind) => {
    documentKindFilter.value = kind
    refreshAgreements()
  }

  const openCreateModal = () => {
    createForm.value = emptyForm()
    createModalVisible.value = true
  }

  const fetchUserConsents = async () => {
    const uid = String(consentUserId.value || '').trim()
    if (!uid) {
      consentError.value = '请输入用户 ID'
      return
    }
    consentLoading.value = true
    consentError.value = ''
    consentSearched.value = false
    try {
      const response = await apiFetch(
        `/api/system-admin/license-agreement/user-consents/?user_id=${encodeURIComponent(uid)}`,
        {
          method: 'GET',
          headers: { Accept: 'application/json' },
          credentials: 'include'
        }
      )
      if (!response.ok) {
        const errData = await response.json().catch(() => ({}))
        const error = new Error(errData.detail || '查询失败')
        error.traceId = response.traceId || errData._traceId || ''
        throw error
      }
      consentRows.value = await response.json()
      consentSearched.value = true
    } catch (err) {
      consentError.value = err.message || '查询失败'
      consentRows.value = []
    } finally {
      consentLoading.value = false
    }
  }

  const refreshAgreements = async () => {
    loading.value = true
    try {
      const kind = encodeURIComponent(documentKindFilter.value)
      const response = await apiFetch(`/api/system-admin/license-agreement/?kind=${kind}`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        agreements.value = await response.json()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const error = new Error(errorData.detail || '获取协议列表失败')
        error.traceId = response.traceId || errorData._traceId || ''
        throw error
      }
    } catch (error) {
      console.error('获取协议列表失败:', error)
      showRequestError('获取协议列表失败: ' + error.message, error)
    } finally {
      loading.value = false
    }
  }

  const handleCreate = async () => {
    creating.value = true
    try {
      const response = await apiFetch('/api/system-admin/license-agreement/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify(createForm.value)
      })

      if (response.ok) {
        const kind = createForm.value.document_kind
        createModalVisible.value = false
        documentKindFilter.value = kind || documentKindFilter.value
        createForm.value = emptyForm()
        await refreshAgreements()
        await checkPaymentTermsPublished()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const error = new Error(errorData.detail || '创建协议失败')
        error.traceId = response.traceId || errorData._traceId || ''
        throw error
      }
    } catch (error) {
      console.error('创建协议失败:', error)
      showRequestError('创建协议失败: ' + error.message, error)
    } finally {
      creating.value = false
    }
  }

  const openEditModal = (agreement) => {
    editForm.value = {
      id: agreement.id,
      title: agreement.title,
      version: agreement.version,
      content: agreement.content,
      document_kind: agreement.document_kind || DOCUMENT_KIND_SERVICE,
      is_active: agreement.is_active,
      is_material_change: Boolean(agreement.is_material_change)
    }
    editModalVisible.value = true
  }

  const handleEdit = async () => {
    editing.value = true
    try {
      const { id, title, version, content, document_kind, is_active, is_material_change } =
        editForm.value
      const response = await apiFetch(`/api/system-admin/license-agreement/${id}/`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify({
          title,
          version,
          content,
          document_kind,
          is_active,
          is_material_change
        })
      })

      if (response.ok) {
        editModalVisible.value = false
        documentKindFilter.value = document_kind
        await refreshAgreements()
        await checkPaymentTermsPublished()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const error = new Error(errorData.detail || '更新协议失败')
        error.traceId = response.traceId || errorData._traceId || ''
        throw error
      }
    } catch (error) {
      console.error('更新协议失败:', error)
      showRequestError('更新协议失败: ' + error.message, error)
    } finally {
      editing.value = false
    }
  }

  const openDeleteModal = (agreement) => {
    deleteTarget.value = {
      id: agreement.id,
      title: agreement.title
    }
    deleteModalVisible.value = true
  }

  const handleDelete = async () => {
    deleting.value = true
    try {
      const response = await apiFetch(
        `/api/system-admin/license-agreement/${deleteTarget.value.id}/`,
        {
          method: 'DELETE',
          headers: {
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      if (response.ok || response.status === 204) {
        deleteModalVisible.value = false
        await refreshAgreements()
        await checkPaymentTermsPublished()
      } else {
        const errorData = await response.json().catch(() => ({}))
        const error = new Error(errorData.detail || '删除协议失败')
        error.traceId = response.traceId || errorData._traceId || ''
        throw error
      }
    } catch (error) {
      console.error('删除协议失败:', error)
      showRequestError('删除协议失败: ' + error.message, error)
    } finally {
      deleting.value = false
    }
  }

  onMounted(async () => {
    await refreshAgreements()
    await checkPaymentTermsPublished()
  })

  return {
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
    checkPaymentTermsPublished,
    formatDate,
    consentContextLabel,
    openCreateModal,
    fetchUserConsents,
    refreshAgreements,
    handleCreate,
    openEditModal,
    handleEdit,
    openDeleteModal,
    handleDelete
  }
}
