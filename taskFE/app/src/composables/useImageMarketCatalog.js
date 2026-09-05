import { ref, computed } from 'vue'
import { apiFetch, extractErrorMessage } from '../utils/apiUtils'
import { safeResponseJson } from '../utils/safeResponseJson.js'
import {
  coerceSnowflakeId,
  normalizeMarketplaceImageList,
  readJsonPreservingSnowflakeIds,
} from '../utils/snowflakeId'
import toastService from '../utils/toastService'
import modalService from '../utils/modalService'
import { resolveInstalledImageIconSrc } from '../utils/imageGroupIcon.js'

const matchesSearch = (q, parts) => {
  if (!q) return true
  const s = q.toLowerCase()
  return parts.some((p) => String(p ?? '').toLowerCase().includes(s))
}

/** 兼容 DRF 数组与网关误路由时 taskCloudService 的 { installed_images: [] } 包裹格式 */
export function normalizeImageList(data) {
  return normalizeMarketplaceImageList(
    Array.isArray(data)
      ? data
      : data && typeof data === 'object' && Array.isArray(data.installed_images)
        ? data.installed_images
        : data && typeof data === 'object' && Array.isArray(data.results)
          ? data.results
          : [],
  )
}

/**
 * 按镜像组分组：同组不同版本叠在一张卡内（组卡 + 版本列表）。
 * 后端目录接口每组返回一条版本记录（name/description 为组字段、version 为版本字段），
 * 开发中目录可能同组多版本平铺；此处按 image_group.id 聚合成组。
 * 无 image_group 字段时回退 name 分组，最后兜底按自身 id 独立成组（保证不合并错项）。
 * 组内顺序保持入参顺序（接口已按 updated_at DESC）。
 */
export function groupImageVersions(images) {
  const groups = new Map()
  const list = Array.isArray(images) ? images : []
  for (const img of list) {
    const key = img?.image_group?.id || img?.name || img?.id || `img-${String(img?.id ?? '')}`
    if (!groups.has(key)) {
      groups.set(key, { ...img, groupKey: key, versions: [] })
    }
    groups.get(key).versions.push(img)
  }
  return [...groups.values()]
}

export function useImageMarketCatalog(tenantId) {
  const catalogImages = ref([])
  const devCatalogImages = ref([])
  const installedImages = ref([])
  const isLoadingCatalog = ref(false)
  const isLoadingDevCatalog = ref(false)
  const isLoadingInstalled = ref(false)
  const installingKey = ref(null)
  const uninstallingId = ref(null)
  const catalogError = ref('')
  const devCatalogError = ref('')
  const catalogErrorTraceId = ref('')
  const devCatalogErrorTraceId = ref('')
  const installedError = ref('')
  const installedErrorTraceId = ref('')
  const catalogSearch = ref('')
  const devSearch = ref('')
  const installedSearch = ref('')

  const pubKey = (id) => `pub-${id}`
  const devKey = (id) => `dev-${id}`
  const installedIconSrc = (image) =>
    resolveInstalledImageIconSrc(image, catalogImages.value, devCatalogImages.value)

  const filteredCatalogImages = computed(() => {
    const q = catalogSearch.value
    return catalogImages.value.filter((img) =>
      matchesSearch(q, [img.name, img.description, img.vendor?.company_name, img.version]),
    )
  })

  const filteredDevImages = computed(() => {
    const q = devSearch.value
    return devCatalogImages.value.filter((img) =>
      matchesSearch(q, [img.name, img.description, img.vendor?.company_name, img.version]),
    )
  })

  const filteredInstalledImages = computed(() => {
    const q = installedSearch.value
    return installedImages.value.filter((img) =>
      matchesSearch(q, [img.name, img.description, img.vendor_name, img.version]),
    )
  })

  // 同组不同版本叠列：一组一张卡，卡内为版本列表（开发中目录可能同组多版本）。
  const devImageGroups = computed(() => groupImageVersions(filteredDevImages.value))
  const catalogImageGroups = computed(() => groupImageVersions(filteredCatalogImages.value))

  const formatDateTime = (dateStr) => {
    if (!dateStr) return '—'
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  const loadCatalog = async () => {
    isLoadingCatalog.value = true
    catalogError.value = ''
    catalogErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/cloud/installed-images/catalog/tenant_id/${tenantId}/`, {
        credentials: 'include',
      })
      if (response.ok) {
        catalogImages.value = normalizeImageList(await readJsonPreservingSnowflakeIds(response))
      } else {
        const { data, traceId } = await safeResponseJson(response, { fallback: {} })
        catalogErrorTraceId.value = traceId
        catalogError.value = extractErrorMessage(data, response) || '加载失败'
      }
    } catch (e) {
      catalogErrorTraceId.value = e?.traceId || ''
      catalogError.value = '网络错误，请稍后重试'
    } finally {
      isLoadingCatalog.value = false
    }
  }

  const loadDevCatalog = async () => {
    isLoadingDevCatalog.value = true
    devCatalogError.value = ''
    devCatalogErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/cloud/installed-images/dev-catalog/tenant_id/${tenantId}/`, {
        credentials: 'include',
      })
      if (response.ok) {
        devCatalogImages.value = normalizeImageList(await readJsonPreservingSnowflakeIds(response))
      } else {
        const { data, traceId } = await safeResponseJson(response, { fallback: {} })
        devCatalogErrorTraceId.value = traceId
        devCatalogError.value = extractErrorMessage(data, response) || '加载开发中镜像失败'
      }
    } catch (e) {
      devCatalogErrorTraceId.value = e?.traceId || ''
      devCatalogError.value = '网络错误，请稍后重试'
    } finally {
      isLoadingDevCatalog.value = false
    }
  }

  const loadInstalledImages = async () => {
    isLoadingInstalled.value = true
    installedError.value = ''
    installedErrorTraceId.value = ''
    try {
      const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
        credentials: 'include',
      })
      if (response.ok) {
        installedImages.value = normalizeImageList(await readJsonPreservingSnowflakeIds(response))
      } else {
        const { data, traceId } = await safeResponseJson(response, { fallback: {} })
        installedErrorTraceId.value = traceId
        installedError.value = extractErrorMessage(data, response) || '加载已安装镜像失败'
      }
    } catch (error) {
      console.error('加载已安装镜像失败:', error)
      installedErrorTraceId.value = error?.traceId || ''
      installedError.value = '网络错误，请稍后重试'
    } finally {
      isLoadingInstalled.value = false
    }
  }

  const reloadAll = () => Promise.all([loadInstalledImages(), loadCatalog(), loadDevCatalog()])

  const installPublishedImage = async (externalImageId) => {
    installingKey.value = pubKey(externalImageId)
    try {
      const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ external_image_id: coerceSnowflakeId(externalImageId) }),
        credentials: 'include',
      })
      if (response.ok) {
        toastService.success('镜像安装成功')
        await reloadAll()
      } else {
        const { data, traceId } = await safeResponseJson(response, { fallback: {} })
        toastService.error(data.detail || '安装失败', { traceId })
      }
    } catch {
      toastService.error('网络错误，请稍后重试')
    } finally {
      installingKey.value = null
    }
  }

  const installDevelopmentImage = async (image) => {
    const vid = image.vendor?.id
    if (!vid || !image.id) return
    installingKey.value = devKey(image.id)
    try {
      const response = await apiFetch(`/api/cloud/installed-images/tenant_id/${tenantId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          vendor_id: coerceSnowflakeId(vid),
          container_id: coerceSnowflakeId(image.id),
        }),
        credentials: 'include',
      })
      if (response.ok) {
        toastService.success('开发中镜像安装成功')
        await reloadAll()
      } else {
        const { data, traceId } = await safeResponseJson(response, { fallback: {} })
        toastService.error(data.detail || '安装失败', { traceId })
      }
    } catch {
      toastService.error('网络错误，请稍后重试')
    } finally {
      installingKey.value = null
    }
  }

  const uninstallImage = async (imageId) => {
    try {
      await modalService.confirm('确定要卸载此镜像吗？卸载后相关任务可能无法再选用该镜像。', '卸载镜像', '卸载', '取消')
    } catch {
      return
    }
    uninstallingId.value = imageId
    try {
      const response = await apiFetch(`/api/cloud/installed-images/${imageId}/tenant_id/${tenantId}/`, {
        method: 'DELETE',
        credentials: 'include',
      })
      if (response.ok) {
        toastService.success('已卸载镜像')
        await reloadAll()
      } else {
        const { data, traceId } = await safeResponseJson(response, { fallback: {} })
        toastService.error(data.detail || '卸载失败', { traceId })
      }
    } catch {
      toastService.error('网络错误，请稍后重试')
    } finally {
      uninstallingId.value = null
    }
  }

  return {
    catalogImages,
    devCatalogImages,
    installedImages,
    isLoadingCatalog,
    isLoadingDevCatalog,
    isLoadingInstalled,
    installingKey,
    uninstallingId,
    catalogError,
    devCatalogError,
    catalogErrorTraceId,
    devCatalogErrorTraceId,
    installedError,
    installedErrorTraceId,
    catalogSearch,
    devSearch,
    installedSearch,
    filteredCatalogImages,
    filteredDevImages,
    filteredInstalledImages,
    devImageGroups,
    catalogImageGroups,
    formatDateTime,
    installedIconSrc,
    pubKey,
    loadCatalog,
    loadDevCatalog,
    loadInstalledImages,
    installPublishedImage,
    installDevelopmentImage,
    uninstallImage,
  }
}
