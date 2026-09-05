import { ref, onMounted } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { getCookie } from '../utils/cookieUtils'
import { adminFormatDate } from '../utils/adminFormatDate.js'

/** @alias:view-system-admin-container-images */
// 注意：本页面未挂载 router（孤儿页面）。container-images/server-images 后端
// 无处理器（taskCloudService 仅实现 regions/instance-types/images，见
// OPT-20260806-002）；路径保留以便功能恢复时定位。
export function useSystemAdminContainerImages() {
  const uploadModalVisible = ref(false)
  const editModalVisible = ref(false)
  const deleteModalVisible = ref(false)
  const environmentModalVisible = ref(false)

  const loadingEdit = ref(false)
  const loadingEnvironment = ref(false)
  const uploading = ref(false)
  const editing = ref(false)
  const deleting = ref(false)
  const settingEnvironment = ref(false)

  const uploadForm = ref({
    name: '',
    description: '',
    image_url: '',
    size: ''
  })

  const editForm = ref({
    name: '',
    description: '',
    image_url: '',
    size: ''
  })

  const environmentForm = ref({})
  const deleteImageId = ref(null)
  const editImageId = ref(null)
  const environmentImageId = ref(null)

  const containerImages = ref([])
  const serverImages = ref([])

  const cloudPlatforms = [
    { value: 'aliyun', label: '阿里云' },
    { value: 'tencentcloud', label: '腾讯云' },
    { value: 'huaweicloud', label: '华为云' },
    { value: 'ctyun', label: '天翼云' },
    { value: 'cmcc', label: '移动云' },
    { value: 'cucloud', label: '联通云' },
    { value: 'baiducloud', label: '百度智能云' },
    { value: 'aws', label: 'AWS' }
  ]

  const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = adminFormatDate

  const refreshContainerImages = async () => {
    try {
      const response = await apiFetch('/api/system-admin/cloud/container-images/', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        containerImages.value = await response.json()
      } else {
        throw new Error('获取镜像列表失败')
      }
    } catch (error) {
      console.error('刷新镜像列表失败:', error)
      alert('刷新镜像列表失败')
    }
  }

  const fetchServerImages = async () => {
    try {
      const response = await apiFetch('/api/system-admin/cloud/server-images/', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        serverImages.value = await response.json()
      }
    } catch (error) {
      console.error('获取服务器镜像失败:', error)
    }
  }

  const handleUploadImage = async () => {
    uploading.value = true
    try {
      const response = await apiFetch('/api/system-admin/cloud/container-images/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify(uploadForm.value)
      })

      if (response.ok) {
        uploadModalVisible.value = false
        await refreshContainerImages()
        uploadForm.value = {
          name: '',
          description: '',
          image_url: '',
          size: ''
        }
      } else {
        throw new Error('上传镜像失败')
      }
    } catch (error) {
      console.error('上传镜像失败:', error)
      alert('上传镜像失败')
    } finally {
      uploading.value = false
    }
  }

  const openEditImageModal = async (imageId) => {
    loadingEdit.value = true
    editImageId.value = imageId
    editModalVisible.value = true

    try {
      const response = await apiFetch(`/api/system-admin/cloud/container-images/${imageId}/`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        const image = await response.json()
        editForm.value = {
          name: image.name,
          description: image.description || '',
          image_url: image.image_url,
          size: image.size || ''
        }
      } else {
        throw new Error('获取镜像详情失败')
      }
    } catch (error) {
      console.error('打开编辑模态框失败:', error)
      alert('打开编辑模态框失败: ' + error.message)
      editModalVisible.value = false
    } finally {
      loadingEdit.value = false
    }
  }

  const handleEditImage = async () => {
    editing.value = true
    try {
      const response = await apiFetch(`/api/system-admin/cloud/container-images/${editImageId.value}/`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include',
        body: JSON.stringify(editForm.value)
      })

      if (response.ok) {
        editModalVisible.value = false
        await refreshContainerImages()
      } else {
        throw new Error('更新镜像失败')
      }
    } catch (error) {
      console.error('更新镜像失败:', error)
      alert('更新镜像失败: ' + error.message)
    } finally {
      editing.value = false
    }
  }

  const openDeleteImageModal = (imageId) => {
    deleteImageId.value = imageId
    deleteModalVisible.value = true
  }

  const handleDeleteImage = async () => {
    deleting.value = true
    try {
      const response = await apiFetch(`/api/system-admin/cloud/container-images/${deleteImageId.value}/`, {
        method: 'DELETE',
        headers: { 'X-Requested-With': 'XMLHttpRequest'
        },
        credentials: 'include'
      })

      if (response.ok) {
        deleteModalVisible.value = false
        await refreshContainerImages()
        deleteImageId.value = null
      } else {
        throw new Error('删除镜像失败')
      }
    } catch (error) {
      console.error('删除镜像失败:', error)
      alert('删除镜像失败: ' + error.message)
    } finally {
      deleting.value = false
    }
  }

  const openSetEnvironmentModal = async (imageId) => {
    loadingEnvironment.value = true
    environmentImageId.value = imageId
    environmentModalVisible.value = true

    cloudPlatforms.forEach(platform => {
      environmentForm.value[platform.value] = ''
    })

    try {
      const associationsResponse = await apiFetch(
        `/api/system-admin/cloud/container-images/${imageId}/cloud-server-image-associations/`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include'
        }
      )

      if (associationsResponse.ok) {
        const associations = await associationsResponse.json()
        associations.forEach(association => {
          environmentForm.value[association.platform_type] = association.cloud_server_image.id
        })
      }

      await fetchServerImages()
    } catch (error) {
      console.error('打开设置环境模态框失败:', error)
      alert('打开设置环境模态框失败: ' + error.message)
      environmentModalVisible.value = false
    } finally {
      loadingEnvironment.value = false
    }
  }

  const handleSetEnvironment = async () => {
    settingEnvironment.value = true
    try {
      const associations = []
      cloudPlatforms.forEach(platform => {
        const imageId = environmentForm.value[platform.value]
        if (imageId) {
          associations.push({
            platform_type: platform.value,
            cloud_server_image: imageId
          })
        }
      })

      const response = await apiFetch(
        `/api/system-admin/cloud/container-images/${environmentImageId.value}/set-cloud-server-images/`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest'
          },
          credentials: 'include',
          body: JSON.stringify({ associations })
        }
      )

      if (response.ok) {
        environmentModalVisible.value = false
        await refreshContainerImages()
      } else {
        throw new Error('设置运行环境失败')
      }
    } catch (error) {
      console.error('设置运行环境失败:', error)
      alert('设置运行环境失败: ' + error.message)
    } finally {
      settingEnvironment.value = false
    }
  }

  onMounted(async () => {
    await refreshContainerImages()
    await fetchServerImages()
  })

  return {
    uploadModalVisible,
    editModalVisible,
    deleteModalVisible,
    environmentModalVisible,
    loadingEdit,
    loadingEnvironment,
    uploading,
    editing,
    deleting,
    settingEnvironment,
    uploadForm,
    editForm,
    environmentForm,
    containerImages,
    serverImages,
    cloudPlatforms,
    formatFileSize,
    formatDate,
    handleUploadImage,
    openEditImageModal,
    handleEditImage,
    openDeleteImageModal,
    handleDeleteImage,
    openSetEnvironmentModal,
    handleSetEnvironment
  }
}
