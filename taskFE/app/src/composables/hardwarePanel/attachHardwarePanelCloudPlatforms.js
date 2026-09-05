import { apiFetch } from '../../utils/apiUtils.js'
import { fetchWorkspaceCloudPlatforms } from '../../utils/workspaceCloudPlatformsApi.js'

export function attachHardwarePanelCloudPlatforms(ctx) {
  const {
    props,
    emit,
    selectedImageId,
    route,
    cloudPlatforms,
    regions,
    zones,
    vpcs,
    vswitches,
    securityGroups,
    selectedRegion,
    selectedZone,
    selectedVpc,
    selectedSecurityGroup,
    selectedCloudPlatform,
    hardwareBootstrapDone,
    isLoadingPlatforms,
    isLoadingRegions,
    isLoadingVpcs,
    isLoadingZones,
    previousServerConfig,
    projectRunTemplateApplied,
    panelExpanded,
    temporaryConfigExpanded,
    showProjectTemplateSummaryOnly,
    showHardwareConfigForm,
    useProjectTemplateForStart,
    hardwareConfigSource,
    activeHardwareConfigSourceLabel,
    runTemplateMeta,
    hardwareConfig,
    autoReleaseEnabled,
    autoReleaseMinutes,
    filterOptions,
    cloudPlatformDefaultConfigs,
    availableInstances,
    isLoadingInstances,
    selectedInstance,
    pendingRestoreInstanceType,
    nextToken,
    isLoadingNextPage,
    currentPage,
    pageSize,
    totalPages,
    pageDataCache,
    instancePrices,
    priceLoading,
    bandwidthLimitations,
    bandwidthLoading,
    selectedBandwidth,
    selectedBandwidthChargingMode,
    sdkMethods,
    imageArchitectureFromSelection,
    imageArchitectureDisplayLabel,
    imageArchitectureDisplayHint,
    containerImageArchitectureForCloudQuery,
    isServerRuntimeRunning,
    startServerDisabledReason,
    isStartServerDisabled,
    startServerButtonLabel,
    installedImages,
    runTemplateMode,
    isServerRunning,
    isServerStarting,
    serverJumpUrl,
    serverJumpDefaultPort,
    availableInstancesFetchScheduler,
  } = ctx

  ctx.loadCloudPlatforms = async ({ skipHandleChange = false } = {}) => {
    if (ctx.loadCloudPlatformsPromise) {
      await ctx.loadCloudPlatformsPromise
      return
    }

    ctx.loadCloudPlatformsPromise = (async () => {
      try {
        ctx.isLoadingPlatformsFlag = true
        isLoadingPlatforms.value = true
        const tenantId = ctx.resolveTenantId()
        const workspaceId = ctx.resolveWorkspaceId()
        if (!tenantId || !workspaceId) {
          console.warn('ctx.loadCloudPlatforms: 缺少 tenantId 或 workspaceId', { tenantId, workspaceId })
          return
        }
        const platforms = await fetchWorkspaceCloudPlatforms(tenantId, workspaceId)
        if (platforms.length) {
          cloudPlatforms.value = platforms
          await ctx.loadAllCloudPlatformsDefaultConfig()

          if (cloudPlatforms.value.length > 0) {
            if (!selectedCloudPlatform.value) {
              selectedCloudPlatform.value = String(cloudPlatforms.value[0].id)
            }
            if (!skipHandleChange) {
              await ctx.handleCloudPlatformChange()
            }
          }
        } else {
          console.error('获取云平台列表失败')
        }
      } catch (error) {
        console.error('获取云平台列表出错:', error)
      } finally {
        isLoadingPlatforms.value = false
        ctx.isLoadingPlatformsFlag = false
        ctx.loadCloudPlatformsPromise = null
      }
    })()

    await ctx.loadCloudPlatformsPromise
  }

  // 获取任务的上一次服务器配置
  ctx.fetchPreviousServerConfig = async () => {
    try {
      const taskId = ctx.resolveTaskId()
      if (!taskId) return

      const tenantId = route.params.tenant
      // 优先使用props传递的workspaceId，其次使用路由参数
      const workspaceId = props.workspaceId || route.params.workspaceId
    
      const response = await apiFetch(`/api/cloud/compute/previous-server-config/tenant_id/${tenantId}/workspace_id/${workspaceId}?task_id=${taskId}`, {
        credentials: 'include',
        headers: {
          'Accept': 'application/json'
        }
      })
    
      if (response.ok) {
        const data = await response.json()
        if (data.status === 'success' && data.server_config) {
          previousServerConfig.value = data.server_config
        }
      }
      if (!ctx.hasConfiguredProjectRunTemplate() || temporaryConfigExpanded.value) {
        await ctx.applyProjectRunTemplateIfNeeded()
      }
    } catch (error) {
      console.error('获取上一次服务器配置出错:', error)
      if (!ctx.hasConfiguredProjectRunTemplate() || temporaryConfigExpanded.value) {
        await ctx.applyProjectRunTemplateIfNeeded()
      }
    }
  }

  // 加载所有云平台的默认配置
  ctx.loadAllCloudPlatformsDefaultConfig = async () => {
    if (ctx.loadDefaultConfigsPromise) {
      await ctx.loadDefaultConfigsPromise
      return
    }

    ctx.loadDefaultConfigsPromise = (async () => {
      try {
        ctx.isLoadingDefaultConfigsFlag = true
        const tenantId = ctx.resolveTenantId()
        const workspaceId = ctx.resolveWorkspaceId()
        if (!tenantId || !workspaceId) return

        const response = await apiFetch(`/api/cloud/platforms/default-config/tenant_id/${tenantId}/workspace_id/${workspaceId}/`, {
          credentials: 'include',
          headers: {
            Accept: 'application/json',
          },
        })

        if (response.ok) {
          const data = await response.json()
          if (data.status === 'success' && data.default_configs) {
            const configs = ctx.normalizeDefaultConfigsList(data.default_configs)
            for (const platform of cloudPlatforms.value) {
              let defaultConfig = configs.find(
                (config) =>
                  config.platform_type === platform.platform_type
                  && config.remark === platform.remark,
              )
              if (!defaultConfig) {
                defaultConfig = configs.find((config) => config.platform_type === platform.platform_type)
              }
              if (defaultConfig) {
                cloudPlatformDefaultConfigs.value[String(platform.id)] = {
                  ...defaultConfig,
                  authorization_id: String(
                    defaultConfig.authorization_id || platform.authorization_id || '',
                  ).trim(),
                }
              } else if (platform.authorization_id) {
                cloudPlatformDefaultConfigs.value[String(platform.id)] = {
                  platform_type: platform.platform_type,
                  authorization_id: String(platform.authorization_id).trim(),
                  remark: platform.remark || '',
                  config: null,
                }
              }
            }
          }
        }
      } catch (error) {
        console.error('加载所有云平台默认配置出错:', error)
      } finally {
        ctx.isLoadingDefaultConfigsFlag = false
        ctx.loadDefaultConfigsPromise = null
      }
    })()

    await ctx.loadDefaultConfigsPromise
  }
  // 处理云平台切换事件
  ctx.handleCloudPlatformChange = async (event) => {
    // 重置地域、VPC、交换机和可用区
    selectedRegion.value = ''
    selectedVpc.value = ''
    selectedZone.value = ''
    zones.value = []
    vpcs.value = []
    vswitches.value = []
  
    // 重置可用实例列表
    availableInstances.value = []
  
    // 应用云平台的默认配置
    if (selectedCloudPlatform.value) {
      let defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
      const platform = cloudPlatforms.value.find(
        (p) => String(p.id) === String(selectedCloudPlatform.value),
      )
      if (!defaultConfig && platform?.authorization_id) {
        cloudPlatformDefaultConfigs.value[String(selectedCloudPlatform.value)] = {
          platform_type: platform.platform_type,
          authorization_id: String(platform.authorization_id).trim(),
          remark: platform.remark || '',
          config: null,
        }
        defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
      }
      if (defaultConfig?.authorization_id || platform?.authorization_id) {
        if (defaultConfig?.config?.bandwidth_charging_mode) {
          selectedBandwidthChargingMode.value = defaultConfig.config.bandwidth_charging_mode
        }
        // 重新加载地域列表（ctx.loadRegions 内会按已保存默认选中地域/VPC/可用区）
        await ctx.loadRegions()
      }
    }
  }

  // 应用上一次运行配置
  ctx.applyPreviousConfig = () => {
    if (previousServerConfig.value) {
      void ctx.applyServerConfigShape(previousServerConfig.value)
    }
  }
}
