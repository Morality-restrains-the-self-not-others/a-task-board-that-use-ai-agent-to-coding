import { ref, computed } from 'vue'
import { isRouteRelayToTraeQuery } from '../../utils/relayToTraeUtils.js'

/**
 * ServerConfig 服务器信息 Tab 切换与默认 Tab 策略。
 * 硬件配置已并入镜像卡，不再作为 Tab。
 */
export function useServerConfigSections({ route, isRelayToTraeEnabled }) {
  const activeServerSection = ref(
    isRouteRelayToTraeQuery(route.query) ? 'relayDirect' : 'runtime',
  )

  const applyDefaultServerTabByRuntime = (_isServerRuntimeRunning) => {
    if (isRelayToTraeEnabled.value) {
      return 'relayDirect'
    }
    return 'runtime'
  }

  return {
    activeServerSection,
    applyDefaultServerTabByRuntime,
  }
}

/**
 * 绑定 Tab 选择与内容区展开；fetch 回调由调用方注入。
 */
export function createServerConfigSectionHandlers({
  activeServerSection,
  expandServerSectionBody,
  fetchServerContent,
  fetchServerStartHistory,
}) {
  void fetchServerContent
  const selectServerSection = (section) => {
    expandServerSectionBody()
    activeServerSection.value = section
  }

  const onServerContentTabClick = () => {
    selectServerSection('content')
  }

  const onServerHistoryTabClick = () => {
    selectServerSection('history')
    fetchServerStartHistory()
  }

  return {
    selectServerSection,
    onServerContentTabClick,
    onServerHistoryTabClick,
  }
}

export function useServerConfigRouteFlags(route) {
  const isRelayToTraeEnabled = computed(() => isRouteRelayToTraeQuery(route.query))
  return { isRelayToTraeEnabled }
}
