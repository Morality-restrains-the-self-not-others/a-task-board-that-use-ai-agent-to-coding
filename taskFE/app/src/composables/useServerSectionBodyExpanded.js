import { ref } from 'vue'

/**
 * 任务详情「服务器信息」Tab 下方内容区折叠/展开。
 * 默认收起；用户切换 Tab（selectServerSection）时应强制展开，避免选中 Tab 却看不到面板。
 */
export function useServerSectionBodyExpanded() {
  const serverSectionBodyExpanded = ref(false)

  const expandServerSectionBody = () => {
    serverSectionBodyExpanded.value = true
  }

  const toggleServerSectionBody = () => {
    serverSectionBodyExpanded.value = !serverSectionBodyExpanded.value
  }

  return {
    serverSectionBodyExpanded,
    expandServerSectionBody,
    toggleServerSectionBody,
  }
}
