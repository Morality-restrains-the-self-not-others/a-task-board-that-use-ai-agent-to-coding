import { apiFetch } from '../../utils/apiUtils.js'
import {
  buildAvailableInstancesFilterParams,
  hasRequiredAvailableInstancesContext,
} from '../../utils/availableInstancesQueryParams.js'

export function attachHardwarePanelInstancesFetch(ctx) {
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

  ctx.fetchAvailableInstances = async (intentGeneration) => {
    ctx.abortPendingAvailableInstancesFetch()
    const { fetchGeneration, isStale } = availableInstancesFetchScheduler.beginFetch(intentGeneration)
    ctx.availableInstancesFetchGeneration = fetchGeneration
    const abortController = new AbortController()
    ctx.availableInstancesAbortController = abortController

    try {
      ctx.isFetchingInstances = true
    
      if (!nextToken.value) {
        // 首次加载或参数变动，清空临时数据区域
        isLoadingInstances.value = true
        currentPage.value = 1
        pageDataCache.value = []
        totalPages.value = 0
        nextToken.value = null
      } else {
        // 加载下一页，保留临时数据区域
        isLoadingNextPage.value = true
      }
      // 重置选中的实例，因为数据将被重新拉取
      selectedInstance.value = null
      const tenantId = ctx.resolveTenantId()
      ctx.ensureDefaultConfigForPlatform(selectedCloudPlatform.value)
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
    
      console.log('开始获取可用实例列表');
      console.log('selectedCloudPlatform:', selectedCloudPlatform.value);
      console.log('selectedRegion:', selectedRegion.value);
      console.log('selectedZone:', selectedZone.value);
      console.log('cores:', filterOptions.value.cores);
      console.log('memory:', filterOptions.value.memory);
      console.log('system_disk_category:', filterOptions.value.system_disk_category);
      console.log('data_disk_category:', filterOptions.value.data_disk_category);
      console.log('spot_strategy:', filterOptions.value.spot_strategy);
      console.log('defaultConfig:', defaultConfig);
    
      // 复用共享必填校验（数据盘类型允许空字符串「不要数据盘」，仅 undefined/null 视为未选）
      if (!hasRequiredAvailableInstancesContext({
        selectedCloudPlatform: selectedCloudPlatform.value,
        selectedRegion: selectedRegion.value,
        selectedZone: selectedZone.value,
        defaultConfig,
        filterOptions: filterOptions.value,
      })) {
        console.log('缺少必要参数，返回空列表');
        availableInstances.value = []
        return
      }
      let url = `/api/cloud/cloud-platform/${defaultConfig.authorization_id}/available-instances/tenant_id/${tenantId}/?platform_type=${defaultConfig.platform_type}&region_id=${selectedRegion.value}`
    
      // 添加zoneId参数
      if (selectedZone.value) {
        // 从selectedZone中提取真正的zoneId（可能包含vswitchId）
        let zoneId = selectedZone.value
        if (zoneId.includes(':')) {
          zoneId = zoneId.split(':')[0]
        }
        url += `&zone_id=${zoneId}`
      }
    
      // 添加过滤选项（复用共享工具，避免与 availableInstancesQueryParams 双份逻辑漂移）
      const filterParams = buildAvailableInstancesFilterParams({
        filterOptions: filterOptions.value,
        nextToken: nextToken.value,
      })

      const filterString = filterParams.toString()
      if (filterString) {
        url += `&${filterString}`
      }
      if (containerImageArchitectureForCloudQuery.value) {
        url += `&image_architecture=${encodeURIComponent(containerImageArchitectureForCloudQuery.value)}`
      }
      if (selectedImageId.value) {
        url += `&container_image_id=${encodeURIComponent(selectedImageId.value)}`
      }
    
      console.log('API URL:', url);
    
      const response = await apiFetch(url, {
        credentials: 'include',
        headers: {
          'Accept': 'application/json'
        },
        signal: abortController.signal,
      })
    
      if (isStale()) {
        return
      }
    
      console.log('API响应状态:', response.status);
    
      if (response.ok) {
        const data = await response.json()
        if (isStale()) {
          return
        }
        console.log('API响应数据:', data);
        if (data.status !== 'error') {
          let newInstances = []
          if (Array.isArray(data)) {
            // 旧格式：直接是实例数组
            console.log('获取到可用实例数量:', data.length);
            // 为每个实例添加必要的字段
            newInstances = data.map(instance => ({
              ...instance,
              storage_gb: hardwareConfig.value.storage_gb || 40,
              storage_type: ctx.resolveInstanceStorageType(instance),
              storage_types: ctx.resolveInstanceStorageTypes(instance)
            }))
            nextToken.value = null
          } else if (typeof data === 'object') {
            // 检查是否有NextToken
            if (data.NextToken) {
              nextToken.value = data.NextToken
              console.log('获取到NextToken:', nextToken.value);
            } else {
              nextToken.value = null
            }
          
            if (data.InstanceTypes && data.InstanceTypes.InstanceType) {
              // 新格式：原始JSON响应，包含InstanceTypes字段
              const instanceTypes = data.InstanceTypes.InstanceType
              console.log('获取到可用实例数量:', instanceTypes.length);
              // 转换字段名称以匹配前端期望的格式
              newInstances = instanceTypes.map(instance => ({
                instance_type: instance.InstanceTypeId || '',
                cpu_cores: instance.CpuCoreCount || 0,
                memory_gb: instance.MemorySize || 0,
                storage_gb: hardwareConfig.value.storage_gb || 40,
                storage_type: ctx.resolveInstanceStorageType(instance),
                storage_types: ctx.resolveInstanceStorageTypes(instance),
                storage_min: instance.diskSupport?.storage_min || instance.StorageMin || instance.storage_min || 20,
                storage_max: instance.diskSupport?.storage_max || instance.StorageMax || instance.storage_max || 2048,
                gpu_cores: instance.GPUAmount || instance.GpuCoreCount || 0,
                gpu_type: instance.GpuSpec || '',
                instance_type_family: instance.InstanceTypeFamily || '',
                status: instance.Status === 'Available' ? 'available' : 'unavailable',
                instance_category: instance.InstanceTypeCategory || '',
                local_disk_size: instance.LocalDiskSize || 0,
                local_disk_type: instance.LocalDiskCategory || '',
                nvme_support: instance.NvmeSupport || '',
                network_performance: instance.NetworkPerformance || '',
                max_bandwidth_out: instance.MaxBandwidthOut || 0,
                max_bandwidth_in: instance.MaxBandwidthIn || 0,
                max_pps: instance.MaxPps || 0,
                eni_quota: instance.EniQuota || 0,
                max_internet_bandwidth_out: instance.MaxInternetBandwidthOut || 0,
                architecture: instance.Architecture || '',
                cpu_type: instance.CpuType || '',
                instance_charge_type_supported: instance.InstanceTypeChargeTypeSupported || '',
                spot_strategy_supported: instance.SpotStrategySupported || '',
                gpu_memory: instance.GpuMemory || 0,
                local_storage_amount: instance.LocalStorageAmount || 0,
                local_storage_category: instance.LocalStorageCategory || '',
                network_type: instance.NetworkType || '',
                primary_network_interface_quota: instance.PrimaryNetworkInterfaceQuota || 0,
                secondary_private_ip_address_quota_per_eni: instance.SecondaryPrivateIpAddressQuotaPerEni || 0,
                storage_amount: instance.StorageAmount || 0,
                storage_category: instance.StorageCategory || '',
                instance_family_level: instance.InstanceFamilyLevel || '',
                instance_type_charge_type: instance.InstanceTypeChargeType || '',
                gpu_count: instance.GpuCount || 0,
                local_disk_count: instance.LocalDiskCount || 0,
                vcpu_core_count: instance.VcpuCoreCount || 0,
                price_info: instance.PriceInfo || instance.price_info || {}
              }))
            } else if (data.instance_types) {
              // 新格式：包含instance_types数组和pagination信息
              console.log('获取到可用实例类型数量:', data.instance_types.length);
              console.log('分页信息:', data.pagination);
              // 保存SDK方法调用记录
              if (data.sdk_methods) {
                sdkMethods.value = data.sdk_methods;
                console.log('SDK方法调用记录:', sdkMethods.value);
              }
              // 更新分页信息
              if (data.pagination) {
                totalPages.value = parseInt(data.pagination.total_pages) || 1
              }
            
              // 为所有实例类型创建基本对象
              const basicInstances = data.instance_types.map(instanceType => ({
                instance_type: instanceType,
                cpu_cores: 0,
                memory_gb: 0,
                storage_gb: hardwareConfig.value.storage_gb || 40,
                storage_type: filterOptions.value.system_disk_category || ctx.DEFAULT_SYSTEM_DISK_CATEGORY,
                storage_types: [filterOptions.value.system_disk_category || ctx.DEFAULT_SYSTEM_DISK_CATEGORY],
                storage_min: 20,
                storage_max: 2048,
                gpu_cores: 0,
                gpu_type: '',
                instance_type_family: '',
                status: 'available',
                instance_category: '',
                local_disk_size: 0,
                local_disk_type: '',
                nvme_support: '',
                network_performance: '',
                max_bandwidth_out: 0,
                max_bandwidth_in: 0,
                max_pps: 0,
                eni_quota: 0,
                max_internet_bandwidth_out: 0,
                architecture: '',
                cpu_type: '',
                instance_charge_type_supported: '',
                spot_strategy_supported: '',
                gpu_memory: 0,
                local_storage_amount: 0,
                local_storage_category: '',
                network_type: '',
                primary_network_interface_quota: 0,
                secondary_private_ip_address_quota_per_eni: 0,
                storage_amount: 0,
                storage_category: '',
                instance_family_level: '',
                instance_type_charge_type: '',
                gpu_count: 0,
                local_disk_count: 0,
                vcpu_core_count: 0,
                price_info: {}
              }));
            
              // 先使用基本对象
              newInstances = basicInstances;
            
              // 将新数据添加到缓存
              pageDataCache.value = [...pageDataCache.value, ...newInstances];
              // 计算总页数
              totalPages.value = Math.ceil(pageDataCache.value.length / pageSize.value);
              // 确保totalPages至少为1
              if (totalPages.value === 0) {
                totalPages.value = 1;
              }
            
              // 只请求当前页的实例详细信息
              await ctx.fetchCurrentPageInstancesDetails(data.instance_types, { fetchGeneration })
            } else {
              console.log('API返回错误或非预期数据格式');
              newInstances = []
            }
          } else {
            console.log('API返回错误或非预期数据格式');
            newInstances = []
          }
        
          if (newInstances.length > 0 && !data.instance_types) {
            pageDataCache.value = [...pageDataCache.value, ...newInstances]
          }

          // 处理分页逻辑
          if (newInstances.length > 0) {
            // 记录加载前的总页数
            const oldTotalPages = totalPages.value
            // 计算总页数
            totalPages.value = Math.ceil(pageDataCache.value.length / pageSize.value)
            // 确保totalPages至少为1
            if (totalPages.value === 0) {
              totalPages.value = 1
            }
            // 如果是通过NextToken加载的，且总页数增加了，则自动跳转到新的一页
            if (nextToken.value && totalPages.value > oldTotalPages) {
              currentPage.value = oldTotalPages + 1
            }
            // 显示当前页数据
            ctx.updateCurrentPageData()
            void ctx.tryRestorePendingSelectedInstance()
          
            // 为每个实例获取价格信息
            newInstances.forEach(instance => {
              // 检查实例是否已经包含价格信息
              if (instance.price_info) {
                // 使用API返回的价格信息，不额外发起请求
                instancePrices.value[instance.instance_type] = instance.price_info
                console.log('使用API返回的价格信息:', instance.instance_type, instance.price_info);
              } else if (instance.PriceInfo) {
                // 兼容旧格式
                instancePrices.value[instance.instance_type] = instance.PriceInfo
                console.log('使用API返回的价格信息 (旧格式):', instance.instance_type, instance.PriceInfo);
              }
              // 不再发起价格请求，因为接口已经返回了价格信息
            })
          } else {
            // 当API返回空数组时，清空缓存并更新界面
            pageDataCache.value = []
            availableInstances.value = []
            totalPages.value = 1
            currentPage.value = 1
          }
        } else {
          console.log('API返回错误:', data.message);
          availableInstances.value = []
          nextToken.value = null
        }
      } else {
        console.error('获取可用实例列表失败，状态码:', response.status);
        availableInstances.value = []
        nextToken.value = null
      }
    } catch (error) {
      if (error?.name === 'AbortError') {
        console.log('可用实例列表请求已取消');
        return
      }
      if (isStale()) {
        return
      }
      console.error('获取可用实例列表出错:', error);
      availableInstances.value = []
    } finally {
      ctx.releaseAvailableInstancesFetchState(fetchGeneration)
    }
  }
}
