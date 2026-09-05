import { ref } from 'vue'

/** 容器 BOOTSTRAP_FAILED 任务关联区失败摘要（文案 + 请求级 trace_id） */
export function createContainerBootstrapFailureState() {
  const containerBootstrapFailureMessage = ref('')
  const containerBootstrapFailureTraceId = ref('')
  return { containerBootstrapFailureMessage, containerBootstrapFailureTraceId }
}
