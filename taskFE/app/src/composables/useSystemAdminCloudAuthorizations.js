import { ref, onMounted } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils'
import { adminFormatDate } from '../utils/adminFormatDate.js'
import modalService from '../utils/modalService.js'

/** @alias:view-system-admin-cloud-authorizations */
// 注意：本页面未挂载 router（孤儿页面）。regions/images 路径已对齐
// taskCloudService /api/system-admin/cloud/ 端点；server-images 后端无处理器
// （见 OPT-20260806-002），路径保留以便功能恢复时定位。
export function useSystemAdminCloudAuthorizations() {
  const addImageModalVisible = ref(false)
  const editImageModalVisible = ref(false)

  const loadingImages = ref(false)
  const loadingRegions = ref(false)
  const loadingImagesList = ref(false)
  const addingImage = ref(false)
  const editingImage = ref(false)
  const deletingImage = ref(false)

  const cloudServerImages = ref([])
  const cloudPlatforms = ref([
    { value: 'aliyun', label: '阿里云' },
    { value: 'tencentcloud', label: '腾讯云' },
    { value: 'huaweicloud', label: '华为云' },
    { value: 'ctyun', label: '天翼云' },
    { value: 'cmcc', label: '移动云' },
    { value: 'cucloud', label: '联通云' },
    { value: 'baiducloud', label: '百度智能云' },
    { value: 'aws', label: 'AWS' }
  ])
  const regions = ref([])
  const images = ref([])
  const selectedImage = ref(null)
  const editRegions = ref([])
  const editImages = ref([])
  const editSelectedImage = ref(null)
  const loadingEditData = ref(false)

  const addForm = ref({
    platform_type: '',
    region: '',
    image_name: '',
    image_id: '',
    os_type: '',
    os_version: '',
    image_type: '',
    is_active: true
  })

  const editForm = ref({
    id: '',
    platform_type: '',
    image_name: '',
    image_id: '',
    region: '',
    os_type: '',
    os_version: '',
    image_type: '',
    is_active: true
  })

  const currentUser = ref(null)

  const getCurrentUser = async () => {
    try {
      const userId = getCookie('userId')
      if (!userId) {
        console.log('Cookie中不存在userId，跳过API请求')
        return null
      }

      const response = await apiFetch(`/api/accounts/users/me/`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        return await response.json()
      }
      return null
    } catch (error) {
      console.error('获取当前用户信息失败:', error)
      return null
    }
  }

  const formatDate = adminFormatDate

  const getPlatformLabel = (value) => {
    const platform = cloudPlatforms.value.find(p => p.value === value)
    return platform ? platform.label : value
  }

  const refreshCloudServerImages = async () => {
    loadingImages.value = true
    try {
      // 注意：taskCloudService 无 server-images 处理器（OPT-20260806-002）；
      // 本页面未挂载 router（孤儿），路径保留以便功能恢复时定位。
      const uid = currentUser.value?.id || '1'
      const response = await apiFetch(`/api/system-admin/${uid}/cloud/server-images/`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        cloudServerImages.value = await response.json()
      } else {
        throw new Error('获取云平台服务器镜像列表失败')
      }
    } catch (error) {
      console.error('刷新云平台服务器镜像列表失败:', error)
      alert('刷新云平台服务器镜像列表失败')
    } finally {
      loadingImages.value = false
    }
  }

  const loadRegions = async (platformType, targetRefs = null) => {
    const regionsRef = targetRefs?.regionsRef ?? regions
    const imagesRef = targetRefs?.imagesRef ?? images
    if (!platformType) {
      regionsRef.value = []
      imagesRef.value = []
      return
    }

    const useEditTarget = !!targetRefs
    if (!useEditTarget) loadingRegions.value = true
    try {
      // 对齐 taskCloudService 后端（handleSystemAdminCloudRoutes）；旧 uid 段路径网关无路由 → 502
      const response = await apiFetch(`/api/system-admin/cloud/regions/?platform_type=${platformType}`, {
        method: 'GET',
        headers: {
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      const responseData = await response.json()

      if (Array.isArray(responseData)) {
        regionsRef.value = responseData
        imagesRef.value = []
      } else if (responseData.regions && Array.isArray(responseData.regions)) {
        regionsRef.value = responseData.regions.map(region => ({
          id: region.region_id || region.id,
          name: region.region_name || region.name
        }))
        imagesRef.value = []
      } else {
        console.warn('获取地域列表响应格式不正确，使用模拟数据')
        const mockRegionsMap = {
          aliyun: [
            { id: 'cn-hangzhou', name: '华东1(杭州)' },
            { id: 'cn-beijing', name: '华北2(北京)' },
            { id: 'cn-shenzhen', name: '华南1(深圳)' },
            { id: 'cn-hongkong', name: '中国香港' },
            { id: 'cn-shanghai', name: '华东2(上海)' }
          ]
        }
        regionsRef.value = mockRegionsMap[platformType] || []
        imagesRef.value = []
      }
    } catch (error) {
      console.error('加载地域列表失败:', error)
      const mockRegionsMap = {
        aliyun: [
          { id: 'cn-hangzhou', name: '华东1(杭州)' },
          { id: 'cn-beijing', name: '华北2(北京)' },
          { id: 'cn-shenzhen', name: '华南1(深圳)' }
        ]
      }
      regionsRef.value = mockRegionsMap[platformType] || []
      imagesRef.value = []
    } finally {
      if (!useEditTarget) loadingRegions.value = false
    }
  }

  const loadImages = async (platformType, regionId, imagesRefParam = null) => {
    const targetImages = imagesRefParam ?? images
    if (!platformType || !regionId) {
      targetImages.value = []
      return
    }

    const useEditTarget = !!imagesRefParam
    if (!useEditTarget) loadingImagesList.value = true
    try {
      // 对齐 taskCloudService 后端（handleSystemAdminCloudImages）；旧 uid 段路径网关无路由 → 502
      const response = await apiFetch(
        `/api/system-admin/cloud/images/?platform_type=${platformType}&region_id=${regionId}`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      const responseData = await response.json()

      let fetchedImages = []
      if (Array.isArray(responseData)) {
        fetchedImages = responseData
      } else if (responseData.images && Array.isArray(responseData.images)) {
        fetchedImages = responseData.images
      } else {
        console.warn('获取镜像列表响应格式不正确')
        fetchedImages = []
      }

      targetImages.value = fetchedImages.filter(image => {
        const osType = (image.os_type || '').toLowerCase()
        return !osType.includes('windows') && !osType.includes('window')
      })
    } catch (error) {
      console.error('加载镜像列表失败:', error)
      targetImages.value = []
    } finally {
      if (!useEditTarget) loadingImagesList.value = false
    }
  }

  const handlePlatformTypeChange = async () => {
    addForm.value.region = ''
    addForm.value.image_name = ''
    addForm.value.image_id = ''
    addForm.value.os_type = ''
    addForm.value.os_version = ''
    addForm.value.image_type = ''
    selectedImage.value = null
    await loadRegions(addForm.value.platform_type)
  }

  const handleRegionChange = async () => {
    addForm.value.image_name = ''
    addForm.value.image_id = ''
    addForm.value.os_type = ''
    addForm.value.os_version = ''
    addForm.value.image_type = ''
    selectedImage.value = null
    await loadImages(addForm.value.platform_type, addForm.value.region)
  }

  const handleCloudImageChange = () => {
    if (selectedImage.value) {
      const image = selectedImage.value
      addForm.value.image_name = image.name
      addForm.value.image_id = image.id
      addForm.value.os_type = image.os_type || ''
      addForm.value.os_version = image.os_version || ''
      addForm.value.image_type = image.image_type || ''
    }
  }

  const handleAddImage = async () => {
    addingImage.value = true
    try {
      const uid = currentUser.value?.id || '1'
      const response = await apiFetch(`/api/system-admin/${uid}/cloud/server-images/`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify(addForm.value)
      })

      if (response.ok) {
        addImageModalVisible.value = false
        addForm.value = {
          platform_type: '',
          region: '',
          image_name: '',
          image_id: '',
          os_type: '',
          os_version: '',
          image_type: '',
          is_active: true
        }
        regions.value = []
        images.value = []
        selectedImage.value = null
        await refreshCloudServerImages()
      } else {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || '添加云平台服务器镜像失败')
      }
    } catch (error) {
      console.error('添加云平台服务器镜像失败:', error)
      alert('添加云平台服务器镜像失败: ' + error.message)
    } finally {
      addingImage.value = false
    }
  }

  const openEditImageModal = async (image) => {
    editForm.value = {
      id: image.id,
      platform_type: image.platform_type,
      image_name: image.image_name,
      image_id: image.image_id,
      region: image.region,
      os_type: image.os_type || '',
      os_version: image.os_version || '',
      image_type: image.image_type || '',
      is_active: image.is_active
    }
    editRegions.value = []
    editImages.value = []
    editSelectedImage.value = null
    editImageModalVisible.value = true
    loadingEditData.value = true
    try {
      const targetRefs = { regionsRef: editRegions, imagesRef: editImages }
      await loadRegions(image.platform_type, targetRefs)
      await loadImages(image.platform_type, image.region, editImages)
      const imgId = String(image.image_id || '')
      const matched = editImages.value.find(img => String(img.id || img.image_id || '') === imgId)
      editSelectedImage.value = matched || null
    } catch (err) {
      console.error('加载编辑数据失败:', err)
    } finally {
      loadingEditData.value = false
    }
  }

  const handleEditPlatformTypeChange = async () => {
    editForm.value.region = ''
    editForm.value.image_name = ''
    editForm.value.image_id = ''
    editForm.value.os_type = ''
    editForm.value.os_version = ''
    editForm.value.image_type = ''
    editSelectedImage.value = null
    const targetRefs = { regionsRef: editRegions, imagesRef: editImages }
    await loadRegions(editForm.value.platform_type, targetRefs)
  }

  const handleEditRegionChange = async () => {
    editForm.value.image_name = ''
    editForm.value.image_id = ''
    editForm.value.os_type = ''
    editForm.value.os_version = ''
    editForm.value.image_type = ''
    editSelectedImage.value = null
    await loadImages(editForm.value.platform_type, editForm.value.region, editImages)
  }

  const handleEditCloudImageChange = () => {
    if (editSelectedImage.value) {
      const img = editSelectedImage.value
      editForm.value.image_name = img.name || img.image_name || ''
      editForm.value.image_id = img.id || img.image_id || ''
      editForm.value.os_type = img.os_type || ''
      editForm.value.os_version = img.os_version || ''
      editForm.value.image_type = img.image_type || ''
    }
  }

  const handleEditImage = async () => {
    editingImage.value = true
    try {
      const uid = currentUser.value?.id || '1'
      const response = await apiFetch(`/api/system-admin/${uid}/cloud/server-images/${editForm.value.id}/`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify(editForm.value)
      })

      if (response.ok) {
        editImageModalVisible.value = false
        await refreshCloudServerImages()
      } else {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || '编辑云平台服务器镜像失败')
      }
    } catch (error) {
      console.error('编辑云平台服务器镜像失败:', error)
      alert('编辑云平台服务器镜像失败: ' + error.message)
    } finally {
      editingImage.value = false
    }
  }

  const handleDeleteImage = async (id, name) => {
    try {
      await modalService.confirm(`确定要删除云平台服务器镜像 "${name}" 吗？`)

      deletingImage.value = true
      const uid = currentUser.value?.id || '1'
      const response = await apiFetch(`/api/system-admin/${uid}/cloud/server-images/${id}/`, {
        method: 'DELETE',
        headers: { 'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        await refreshCloudServerImages()
      } else {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.message || '删除云平台服务器镜像失败')
      }
    } catch (error) {
      if (error) {
        console.error('删除云平台服务器镜像失败:', error)
        modalService.alert('删除云平台服务器镜像失败: ' + error.message)
      }
    } finally {
      deletingImage.value = false
    }
  }

  onMounted(async () => {
    currentUser.value = await getCurrentUser()
    await refreshCloudServerImages()
  })

  return {
    addImageModalVisible,
    editImageModalVisible,
    loadingImages,
    addingImage,
    editingImage,
    cloudServerImages,
    cloudPlatforms,
    regions,
    images,
    selectedImage,
    editRegions,
    editImages,
    editSelectedImage,
    loadingEditData,
    addForm,
    editForm,
    formatDate,
    getPlatformLabel,
    handlePlatformTypeChange,
    handleRegionChange,
    handleCloudImageChange,
    handleAddImage,
    openEditImageModal,
    handleEditPlatformTypeChange,
    handleEditRegionChange,
    handleEditCloudImageChange,
    handleEditImage,
    handleDeleteImage
  }
}
