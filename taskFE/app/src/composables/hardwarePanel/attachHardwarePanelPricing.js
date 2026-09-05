import { computed, watch } from 'vue'
import { apiFetch } from '../../utils/apiUtils.js'
import {
  runTemplateToServerConfigShape,
  resolveRunTemplateInstanceType,
  buildRunTemplateHardwareSpecSummary,
  formatGpuSpec,
  formatInstanceHourlyPrice,
  summarizeRunTemplateHardwareSpecs,
} from '../../utils/projectRunTemplateUtils.js'

export function attachHardwarePanelPricing(ctx) {
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

  ctx.parseBandwidthLimitationResponse = (data) => {
    if (!data || typeof data !== 'object') return null
    if (data.BandwidthInfo && (data.BandwidthInfo.min_bandwidth != null || data.BandwidthInfo.max_bandwidth != null)) {
      return {
        min_bandwidth: data.BandwidthInfo.min_bandwidth ?? 0,
        max_bandwidth: data.BandwidthInfo.max_bandwidth ?? 0,
        bandwidth_options: data.BandwidthInfo.bandwidth_options,
      }
    }
    if (data.min_bandwidth != null || data.max_bandwidth != null) {
      return {
        min_bandwidth: data.min_bandwidth ?? 0,
        max_bandwidth: data.max_bandwidth ?? 0,
        bandwidth_options: data.bandwidth_options,
      }
    }
    return null
  }

  ctx.clampBandwidthValue = (instanceType, rawValue) => {
    const limitation = bandwidthLimitations.value[instanceType]
    const min = limitation?.min_bandwidth ?? 0
    const max = limitation?.max_bandwidth ?? 100
    let num = Number(rawValue)
    if (Number.isNaN(num)) {
      num = ctx.DEFAULT_BANDWIDTH
    }
    return Math.min(max, Math.max(min, Math.round(num)))
  }

  ctx.onBandwidthInput = (instanceType, event) => {
    const raw = event.target.value
    if (raw === '') {
      selectedBandwidth.value[instanceType] = ''
      return
    }
    selectedBandwidth.value[instanceType] = Number(raw)
  }

  ctx.onBandwidthBlur = (instanceType) => {
    if (!bandwidthLimitations.value[instanceType]) return
    const clamped = ctx.clampBandwidthValue(instanceType, selectedBandwidth.value[instanceType])
    if (selectedBandwidth.value[instanceType] !== clamped) {
      selectedBandwidth.value[instanceType] = clamped
    }
  }

  ctx.normalizeInstancePriceRecord = (data) => {
    if (!data || typeof data !== 'object' || data.error) return null
    const priceObj = data.Price || {}
    const currency = priceObj.Currency || data.currency || data.Currency || 'CNY'
    const tradePrice = priceObj.TradePrice ?? data.price ?? data.trade_price ?? null
    if (tradePrice == null || tradePrice === '无法获取价格') return null
    const originalPrice = priceObj.OriginalPrice ?? data.original_price ?? null
    const discountPrice = priceObj.DiscountPrice ?? data.discount_price ?? null
    const detailInfos = priceObj.DetailInfos || data.detail_infos || data.DetailInfos || {}
    const detailList = detailInfos.detail_info || detailInfos.DetailInfo || detailInfos.detailInfo || []
    const details = (Array.isArray(detailList) ? detailList : []).map((detail) => ({
      resource: detail.resource || detail.Resource || '',
      amount: detail.trade_price ?? detail.TradePrice ?? detail.original_price ?? detail.OriginalPrice ?? 0,
    })).filter((detail) => detail.resource)
    return {
      currency,
      tradePrice,
      originalPrice,
      discountPrice,
      details,
    }
  }

  ctx.getInstancePriceView = (instanceType) => ctx.normalizeInstancePriceRecord(instancePrices.value[instanceType])

  ctx.selectedInstancePriceView = computed(() => {
    const instanceType = selectedInstance.value?.instance_type
    if (!instanceType) return null
    return ctx.getInstancePriceView(instanceType)
  })

  ctx.formatPriceAmount = (currency, amount) => {
    const formatted = formatInstanceHourlyPrice(currency, amount)
    if (formatted) return formatted.replace('元/时', ' 元/小时')
    return '—'
  }

  ctx.projectTemplateHardwareSummary = computed(() => {
    const tpl = props.projectServerRunTemplate
    if (!tpl || typeof tpl !== 'object') return ''
    const instanceType = resolveRunTemplateInstanceType(tpl)
    const hw = tpl.hardware_config && typeof tpl.hardware_config === 'object' ? tpl.hardware_config : {}
    const priceView = instanceType ? ctx.getInstancePriceView(instanceType) : null
    const priceLoadingNow = instanceType ? Boolean(priceLoading.value[instanceType]) : false
    const priceText = priceView
      ? formatInstanceHourlyPrice(priceView.currency, priceView.tradePrice)
      : null

    const summary = buildRunTemplateHardwareSpecSummary({
      cpuCores: hw.cpu_cores,
      memoryGb: hw.memory_gb,
      gpuSpec: formatGpuSpec(hw),
      priceText: showProjectTemplateSummaryOnly.value ? null : priceText,
      priceLoading: showProjectTemplateSummaryOnly.value ? false : priceLoadingNow,
    })
    if (summary) return summary
    return summarizeRunTemplateHardwareSpecs(tpl)
  })

  ctx.projectTemplatePriceHint = computed(() => {
    if (!showProjectTemplateSummaryOnly.value) return ''
    const instanceType = resolveRunTemplateInstanceType(props.projectServerRunTemplate)
    if (!instanceType) return ''
    if (priceLoading.value[instanceType]) return '按量付费价格：加载中…'
    const priceView = ctx.getInstancePriceView(instanceType)
    if (!priceView) return ''
    const formatted = ctx.formatPriceAmount(priceView.currency, priceView.tradePrice)
    if (!formatted || formatted === '—') return ''
    return `按量付费价格：${formatted}（仅供参考，实际以云厂商为准）`
  })

  ctx.runTemplateSpecSummary = computed(() => {
    if (!props.runTemplateMode) return ''
    const inst = selectedInstance.value
    const hw = hardwareConfig.value
    const tplHw = props.projectServerRunTemplate?.hardware_config || {}
    const instanceType = inst?.instance_type || resolveRunTemplateInstanceType(props.projectServerRunTemplate)

    let cpuCores = inst?.cpu_cores ?? null
    let memoryGb = inst?.memory_gb ?? null
    let gpuSpec = formatGpuSpec(inst)

    if (!inst) {
      const tplCpu = tplHw?.cpu_cores ?? hw?.cpu_cores
      const tplMem = tplHw?.memory_gb ?? hw?.memory_gb
      const looksLikePlaceholder = instanceType
        && String(tplCpu ?? '') === '1'
        && String(tplMem ?? '') === '1'
      if (!looksLikePlaceholder && (tplCpu || tplMem)) {
        cpuCores = tplCpu ?? null
        memoryGb = tplMem ?? null
        gpuSpec = formatGpuSpec(tplHw || hw)
      }
    }

    const priceView = instanceType ? ctx.getInstancePriceView(instanceType) : null
    const priceLoadingNow = instanceType ? Boolean(priceLoading.value[instanceType]) : false
    const priceText = priceView
      ? formatInstanceHourlyPrice(priceView.currency, priceView.tradePrice)
      : null

    const summary = buildRunTemplateHardwareSpecSummary({
      cpuCores,
      memoryGb,
      gpuSpec,
      priceText,
      priceLoading: priceLoadingNow,
    })
    if (summary) return summary
    if (instanceType) return `实例 ${instanceType}`
    return ''
  })

  watch(
    ctx.runTemplateSpecSummary,
    (summary) => {
      if (props.runTemplateMode) {
        emit('spec-summary-change', summary)
      }
    },
    { immediate: true },
  )

  // 获取带宽限制
  ctx.fetchBandwidthLimitation = async (instance) => {
    if (!instance || !instance.instance_type) return;
  
    // 防止重复请求
    if (bandwidthLoading.value[instance.instance_type]) {
      return;
    }
  
    try {
      bandwidthLoading.value[instance.instance_type] = true;
      const tenantId = route.params.tenant;
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig();
    
      if (!defaultConfig || !selectedRegion.value) {
        return;
      }
    
      // 构建API URL
      const url = `/api/cloud/cloud-platform/${defaultConfig.authorization_id}/bandwidth-limitation/tenant_id/${tenantId}/?platform_type=${defaultConfig.platform_type}&region_id=${selectedRegion.value}&instance_type=${instance.instance_type}`;
    
      const response = await apiFetch(url, {
        credentials: 'include',
        headers: {
          'Accept': 'application/json'
        }
      });
    
      if (response.ok) {
        const data = await response.json();
        const bandwidthInfo = ctx.parseBandwidthLimitationResponse(data);
        if (bandwidthInfo) {
            bandwidthLimitations.value[instance.instance_type] = bandwidthInfo;
            // 默认选择 ctx.DEFAULT_BANDWIDTH，如果不在范围内则使用最小带宽
            if (selectedBandwidth.value[instance.instance_type] == null || selectedBandwidth.value[instance.instance_type] === '') {
              const minBandwidth = bandwidthInfo.min_bandwidth;
              const maxBandwidth = bandwidthInfo.max_bandwidth;
              let defaultBandwidth = ctx.DEFAULT_BANDWIDTH;
            
              // 确保默认带宽在范围内
              if (defaultBandwidth < minBandwidth) {
                defaultBandwidth = minBandwidth;
              } else if (defaultBandwidth > maxBandwidth) {
                defaultBandwidth = maxBandwidth;
              }
            
              selectedBandwidth.value[instance.instance_type] = defaultBandwidth;
              // 当设置默认带宽后，重新获取价格信息
              if (instance.storage_type) {
                await ctx.fetchInstancePrice(
                  instance.instance_type, 
                  instance.storage_type, 
                  instance.storage_gb, 
                  defaultBandwidth
                );
              }
            }
          }
      }
    } catch (error) {
      console.error('获取带宽限制失败:', error);
    } finally {
      bandwidthLoading.value[instance.instance_type] = false;
    }
  };
  watch(
    () => ({
      cores: filterOptions.value.cores,
      memory: filterOptions.value.memory,
      io_optimized: filterOptions.value.io_optimized,
      system_disk_category: filterOptions.value.system_disk_category,
      data_disk_category: filterOptions.value.data_disk_category,
      spot_strategy: filterOptions.value.spot_strategy,
      network_category: filterOptions.value.network_category,
      image_architecture: filterOptions.value.image_architecture,
      selectedZone: selectedZone.value,
      imageArch: containerImageArchitectureForCloudQuery.value
    }), 
    async (newValues, oldValues) => {
      // 检查参数是否真正发生变化
      if (!oldValues || !ctx.deepEqual(newValues, oldValues)) {
        ctx.scheduleFetchAvailableInstances()
      }
    }, 
    { deep: true }
  )

  // 监听带宽选择变化，重新获取价格信息
  watch(
    () => selectedInstance.value ? selectedBandwidth.value[selectedInstance.value.instance_type] : null, 
    async (newBandwidth, oldBandwidth) => {
      // 检查是否有带宽值变化
      if (selectedInstance.value && newBandwidth !== oldBandwidth) {
        // 当带宽值变化时，重新获取价格信息
        await ctx.fetchInstancePrice(
          selectedInstance.value.instance_type, 
          selectedInstance.value.storage_type, 
          selectedInstance.value.storage_gb, 
          newBandwidth
        );
      }
    }
  )
  ctx.fetchInstancePrice = async (instanceType, storageType, storageGb, bandwidth) => {
    try {
      // 设置加载状态
      priceLoading.value[instanceType] = true
    
      const tenantId = route.params.tenant
      const defaultConfig = ctx.resolveCloudPlatformDefaultConfig()
    
      // 找到对应的实例以获取存储大小
      const instance = pageDataCache.value.find(item => item.instance_type === instanceType)
      const storage_gb = storageGb || (instance ? instance.storage_gb : 40)
      const bandwidth_value = bandwidth || (selectedBandwidth.value[instanceType] || ctx.DEFAULT_BANDWIDTH)
    
      if (defaultConfig && defaultConfig.authorization_id) {
        const priceUrl = `/api/cloud/cloud-platform/${defaultConfig.authorization_id}/instance-price/tenant_id/${tenantId}/?platform_type=${defaultConfig.platform_type}&region_id=${selectedRegion.value}&instance_type=${instanceType}&system_disk_category=${storageType}&storage_gb=${storage_gb}&bandwidth=${bandwidth_value}&spot_strategy=${filterOptions.value.spot_strategy}`
      
        console.log('请求实例价格信息:', priceUrl)
      
        const priceResponse = await apiFetch(priceUrl, {
          credentials: 'include',
          headers: {
            'Accept': 'application/json'
          }
        })
      
        if (priceResponse.ok) {
          const priceData = await priceResponse.json()
          console.log('获取到实例价格信息:', instanceType, priceData)
        
          // 检查是否包含错误信息
          if (priceData.error) {
            console.error('获取实例价格信息失败:', priceData.error)
            // 可以设置一个默认值或者不更新价格信息
            instancePrices.value[instanceType] = {
              Price: {
                Currency: '',
                TradePrice: '无法获取价格'
              }
            }
          } else {
            instancePrices.value[instanceType] = priceData
            const resolvedDiskCategory = priceData.system_disk_category
            if (resolvedDiskCategory && resolvedDiskCategory !== storageType) {
              const cachedInstance = pageDataCache.value.find(item => item.instance_type === instanceType)
              if (cachedInstance) {
                cachedInstance.storage_type = resolvedDiskCategory
                if (!cachedInstance.storage_types?.includes(resolvedDiskCategory)) {
                  cachedInstance.storage_types = ctx.resolveInstanceStorageTypes(
                    { ...cachedInstance, storage_type: resolvedDiskCategory },
                    resolvedDiskCategory,
                  )
                }
              }
              if (selectedInstance.value?.instance_type === instanceType) {
                selectedInstance.value = {
                  ...selectedInstance.value,
                  storage_type: resolvedDiskCategory,
                  storage_types: cachedInstance?.storage_types || [resolvedDiskCategory],
                }
              }
            }
          }
        } else {
          console.error('获取实例价格信息失败:', await priceResponse.text())
          // 设置默认值
          instancePrices.value[instanceType] = {
            Price: {
              Currency: '',
              TradePrice: '无法获取价格'
            }
          }
        }
      }
    } catch (error) {
      console.error('获取实例价格信息出错:', error)
    } finally {
      // 清除加载状态
      priceLoading.value[instanceType] = false
    }
  }

  ctx.fetchProjectTemplateSummaryPrice = async () => {
    const tpl = props.projectServerRunTemplate
    if (!tpl || !showProjectTemplateSummaryOnly.value) return

    const instanceType = resolveRunTemplateInstanceType(tpl)
    const shape = runTemplateToServerConfigShape(tpl)
    if (!instanceType || !shape?.region) return
    if (priceLoading.value[instanceType] || ctx.getInstancePriceView(instanceType)) return

    await ctx.bootstrapCloudPlatformsIfNeeded()
    const shapeCopy = { ...shape }
    ctx.resolveShapePlatformId(shapeCopy, tpl)
    if (!shapeCopy.platform_id) return

    const savedPlatform = selectedCloudPlatform.value
    const savedRegion = selectedRegion.value
    try {
      selectedCloudPlatform.value = String(shapeCopy.platform_id)
      ctx.ensureDefaultConfigForPlatform(shapeCopy.platform_id, shapeCopy.authorization_id)
      selectedRegion.value = String(shape.region)
      const filterOptions = tpl.filter_options && typeof tpl.filter_options === 'object'
        ? tpl.filter_options
        : {}
      const hw = shape.hardware_config || {}
      await ctx.fetchInstancePrice(
        instanceType,
        filterOptions.system_disk_category || ctx.DEFAULT_SYSTEM_DISK_CATEGORY,
        hw.storage_gb || 40,
        Number(tpl.bandwidth ?? ctx.DEFAULT_BANDWIDTH),
      )
    } finally {
      if (!temporaryConfigExpanded.value) {
        selectedCloudPlatform.value = savedPlatform
        selectedRegion.value = savedRegion
      }
    }
  }
}
