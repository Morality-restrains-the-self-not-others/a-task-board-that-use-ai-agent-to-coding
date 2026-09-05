import { apiFetch } from '../../utils/apiUtils.js'

export function attachHardwarePanelInstancesPage(ctx) {
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

  // 更新当前页数据
  ctx.updateCurrentPageData = () => {
    const startIndex = (currentPage.value - 1) * pageSize.value
    const endIndex = startIndex + pageSize.value
    availableInstances.value = pageDataCache.value.slice(startIndex, endIndex)
  }

  // 上一页
  ctx.prevPage = async () => {
    if (currentPage.value > 1) {
      currentPage.value--
      ctx.updateCurrentPageData()
      // 请求当前页的实例详细信息
      if (pageDataCache.value.length > 0) {
        const allInstanceTypes = pageDataCache.value.map(instance => instance.instance_type);
        await ctx.fetchCurrentPageInstancesDetails(allInstanceTypes);
      }
    }
  }

  // 下一页
  ctx.nextPage = async () => {
    // 如果当前页是已缓存的最后一页且有NextToken，则加载更多数据
    if (currentPage.value === totalPages.value && nextToken.value) {
      await ctx.fetchAvailableInstances()
    } else if (currentPage.value < totalPages.value) {
      // 如果当前页不是最后一页，则直接切换到下一页
      currentPage.value++
      ctx.updateCurrentPageData()
      // 请求当前页的实例详细信息
      if (pageDataCache.value.length > 0) {
        const allInstanceTypes = pageDataCache.value.map(instance => instance.instance_type);
        await ctx.fetchCurrentPageInstancesDetails(allInstanceTypes);
      }
    }
  }

  // 处理存储类型变更
  ctx.handleStorageTypeChange = async (instance, storageType) => {
    // 更新实例的存储类型
    instance.storage_type = storageType
  
    // 如果该实例是当前选中的实例，重新获取价格信息
    if (selectedInstance.value && selectedInstance.value.instance_type === instance.instance_type) {
      selectedInstance.value = instance
      if (instance.instance_type && storageType) {
        await ctx.fetchInstancePrice(instance.instance_type, storageType, instance.storage_gb)
      }
    }
  }

  // 处理存储大小变更
  ctx.handleStorageSizeChange = async (instance) => {
    // 如果该实例是当前选中的实例，重新获取价格信息
    if (selectedInstance.value && selectedInstance.value.instance_type === instance.instance_type) {
      selectedInstance.value = instance
      if (instance.instance_type && instance.storage_type) {
        await ctx.fetchInstancePrice(instance.instance_type, instance.storage_type, instance.storage_gb)
      }
    }
  }

  // 选择实例
  ctx.selectInstance = async (instance) => {
    // 如果当前实例已经被选中，则取消选择
    if (selectedInstance.value && selectedInstance.value.instance_type === instance.instance_type) {
      selectedInstance.value = null
      pendingRestoreInstanceType.value = ''
      return
    } else {
      // 选中新实例
      selectedInstance.value = instance

      if (instance.cpu_cores != null && Number(instance.cpu_cores) > 0) {
        hardwareConfig.value.cpu_cores = String(instance.cpu_cores)
      }
      if (instance.memory_gb != null && Number(instance.memory_gb) > 0) {
        hardwareConfig.value.memory_gb = String(instance.memory_gb)
      }
    
      // 获取实例价格信息
      if (instance.instance_type && instance.storage_type) {
        await ctx.fetchInstancePrice(instance.instance_type, instance.storage_type, instance.storage_gb)
      }
    
      // 自动获取带宽限制
      ctx.fetchBandwidthLimitation(instance)
    }
  }
  // 获取当前页的实例详细信息
  ctx.fetchCurrentPageInstancesDetails = async (allInstanceTypes, { fetchGeneration } = {}) => {
    ctx.abortPendingInstanceDetailsFetch()
    const detailsController = new AbortController()
    ctx.availableInstancesDetailsAbortController = detailsController
    const isStale = () => (
      fetchGeneration != null && fetchGeneration !== ctx.availableInstancesFetchGeneration
    )

    try {
      // 计算当前页的实例类型范围
      const startIndex = (currentPage.value - 1) * pageSize.value;
      const endIndex = startIndex + pageSize.value;
      const currentPageInstanceTypes = allInstanceTypes.slice(startIndex, endIndex);
    
      console.log('当前页实例类型:', currentPageInstanceTypes);
    
      if (currentPageInstanceTypes.length > 0) {
        // 构建实例类型参数，最多一次请求10个实例
        const instanceTypesChunks = [];
        for (let i = 0; i < currentPageInstanceTypes.length; i += 10) {
          instanceTypesChunks.push(currentPageInstanceTypes.slice(i, i + 10));
        }
      
        const tenantId = route.params.tenant;
        const defaultConfig = ctx.resolveCloudPlatformDefaultConfig();
      
        if (defaultConfig && defaultConfig.authorization_id) {
          // 处理每个实例类型 chunk
          for (const chunk of instanceTypesChunks) {
            try {
              const instanceTypesParam = chunk.join(',');
              const detailUrl = `/api/cloud/cloud-platform/${defaultConfig.authorization_id}/instance-details/tenant_id/${tenantId}/?platform_type=${defaultConfig.platform_type}&region_id=${selectedRegion.value}&instance_types=${instanceTypesParam}`;
            
              console.log('请求当前页实例详细信息:', detailUrl);
            
              const detailResponse = await apiFetch(detailUrl, {
                credentials: 'include',
                headers: {
                  'Accept': 'application/json'
                },
                signal: detailsController.signal,
              });
            
              if (isStale()) {
                return
              }
            
              if (detailResponse.ok) {
                const detailData = await detailResponse.json();
                if (isStale()) {
                  return
                }
                console.log('获取到当前页实例详细信息:', detailData.length);
              
                // 直接更新pageDataCache中的所有实例
                // 遍历所有缓存的实例
                pageDataCache.value = pageDataCache.value.map(instance => {
                  // 查找API响应中对应的实例详细信息
                  const detailInstance = detailData.find(detail => detail.instance_type === instance.instance_type);
                
                  if (detailInstance) {
                    // 转换API响应字段名以匹配前端期望的字段名
                    const transformedInstance = {
                      ...instance,
                      ...detailInstance,
                      // 映射字段名
                      gpu_cores: detailInstance.gpu_amount || detailInstance.gpu_cores,
                      gpu_type: detailInstance.gpu_spec || detailInstance.gpu_type,
                      instance_category: detailInstance.instance_type_category || detailInstance.instance_category,
                      // 确保必要的字段存在
                      storage_gb: hardwareConfig.value.storage_gb || 40,
                      storage_type: ctx.resolveInstanceStorageType(detailInstance),
                      storage_types: ctx.resolveInstanceStorageTypes(detailInstance),
                      // 确保CPU和内存字段存在且不为空
                      cpu_cores: detailInstance.cpu_cores || 0,
                      memory_gb: detailInstance.memory_gb || 0,
                      instance_type_family: detailInstance.instance_type_family || '',
                      status: detailInstance.status || 'unavailable'
                    };
                  
                    console.log('更新实例信息:', transformedInstance.instance_type, 'CPU:', transformedInstance.cpu_cores, '内存:', transformedInstance.memory_gb, '实例家族:', transformedInstance.instance_type_family);
                    return transformedInstance;
                  }
                
                  return instance;
                });
              
                // 重新更新当前页数据
                ctx.updateCurrentPageData();
                void ctx.tryRestorePendingSelectedInstance();
              } else {
                console.error('获取当前页实例详细信息失败:', await detailResponse.text());
              }
            } catch (error) {
              if (error?.name === 'AbortError') {
                console.log('实例详细信息请求已取消');
                return
              }
              console.error('获取当前页实例详细信息出错:', error);
            }
          }
        }
      }
    } catch (error) {
      if (error?.name === 'AbortError') {
        return
      }
      console.error('获取当前页实例详细信息出错:', error);
    } finally {
      if (ctx.availableInstancesDetailsAbortController === detailsController) {
        ctx.availableInstancesDetailsAbortController = null
      }
    }
  }

  // 获取资源名称的中文翻译
  ctx.getResourceName = (resource) => {
    const resourceMap = {
      'instanceType': '实例类型',
      'systemDisk': '系统盘',
      'bandwidth': '带宽',
      'image': '镜像'
    }
    return resourceMap[resource] || resource
  }

  // 加载下一页实例
  ctx.loadNextPage = async () => {
    await ctx.nextPage()
  }
}
