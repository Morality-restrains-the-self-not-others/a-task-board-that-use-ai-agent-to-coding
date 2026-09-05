import { ref } from 'vue'

/**
 * 任务详情「服务器信息」Tab 下方内容区折叠/展开。
 * 默认展开；切换 Tab 时应强制展开，避免选中 Tab 却看不到面板。
 */
export function useServerSectionBodyExpanded() {
  const serverSectionBodyExpanded = ref(true)

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
