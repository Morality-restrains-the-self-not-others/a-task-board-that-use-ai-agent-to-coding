import { ref, watch, onBeforeUnmount } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import { resolveTaskWorkspaceId, resolveServerConfigTaskId } from '../../utils/serverConfigRouteHelpers.js'
import { pendingImageMention } from './commentImageMentionState.js'
import { bindServerConfigSelectedImage } from './taskDetailImageSelectionBridge.js'

/**
 * ServerConfig 已安装镜像列表与选中镜像 PATCH 同步。
 */
export function useServerConfigImages({ props, route }) {
  const installedImages = ref([])
  const selectedImageId = ref('')
  const isInitialized = ref(false)

  let isFetchingInstalledImages = false
  const stopBridge = bindServerConfigSelectedImage(selectedImageId)
  onBeforeUnmount(() => stopBridge())

  // 评论区 $镜像 → 同步任务级选中镜像（跨兄弟组件）
  watch(pendingImageMention, (mention) => {
    if (!mention?.id) return
    const next = String(mention.id)
    if (selectedImageId.value !== next) {
      selectedImageId.value = next
    }
  })

  const fetchInstalledImages = async () => {
    if (isFetchingInstalledImages) {
      console.log('正在获取已安装镜像列表，跳过重复请求')
      return
    }
    try {
      isFetchingInstalledImages = true
      const tenantId = route.params.tenant
      const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
        credentials: 'include',
        headers: {
          Accept: 'application/json',
        },
      })
      if (response.ok) {
        const data = await response.json()
        installedImages.value = Array.isArray(data) ? data : data.results || []
      } else {
        console.error('获取已安装镜像失败')
      }
    } catch (error) {
      console.error('获取已安装镜像出错:', error)
    } finally {
      isFetchingInstalledImages = false
    }
  }

  watch(selectedImageId, async (newImageId) => {
    if (!isInitialized.value || !newImageId || !props.task) {
      return
    }
    try {
      const selectedImage = installedImages.value.find((img) => img.id === newImageId)
      if (!selectedImage) {
        return
      }
      const tenantId = route.params.tenant || (props.task && props.task.tenant_id)
      const workspaceId = resolveTaskWorkspaceId({
        route,
        task: props.task,
        workspaceId: props.workspaceId,
      })
      const taskId = resolveServerConfigTaskId({
        route,
        task: props.task,
        taskId: props.taskId,
        tenantId: props.tenantId,
        workspaceId: props.workspaceId,
      })
      if (!taskId) {
        return
      }
      const url = `/api/tasks/todos/tenant_id/${tenantId}/workspace_id/${workspaceId}/${taskId}/`
      const response = await apiFetch(url, {
        method: 'PATCH',
        credentials: 'include',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          container_image_id: newImageId,
        }),
      })
      if (!response.ok) {
        console.error('更新任务镜像失败')
      }
    } catch (error) {
      console.error('更新任务镜像出错:', error)
    }
  })

  return {
    installedImages,
    selectedImageId,
    isInitialized,
    fetchInstalledImages,
  }
}
