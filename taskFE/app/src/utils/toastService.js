// Toast 通知服务
import { reactive } from 'vue'
import { extractTraceId } from './traceId.js'

const toastService = {
  // 存储 toast 状态
  state: reactive({
    show: false,
    message: '',
    type: 'success', // success, error, warning, info
    duration: 3000, // 默认显示3秒
    traceId: '',
  }),
  
  // 显示 toast 通知；error 类型可传 options.traceId → data-traceId
  show(message, type = 'success', duration = 3000, options = {}) {
    this.state.show = true
    this.state.message = message
    this.state.type = type
    this.state.duration = duration
    const tid = extractTraceId(options.traceId) || extractTraceId(options.source)
    this.state.traceId = type === 'error' ? tid : ''
    
    // 自动关闭
    setTimeout(() => {
      this.hide()
    }, duration)
  },
  
  // 成功通知
  success(message, duration = 3000) {
    this.show(message, 'success', duration)
  },
  
  // 错误通知（第三参可为 duration number，或 { duration, traceId }）
  error(message, durationOrOptions = 3000, maybeOptions) {
    if (durationOrOptions != null && typeof durationOrOptions === 'object') {
      const duration = durationOrOptions.duration ?? 3000
      this.show(message, 'error', duration, durationOrOptions)
      return
    }
    this.show(message, 'error', durationOrOptions, maybeOptions || {})
  },
  
  // 警告通知
  warning(message, duration = 3000) {
    this.show(message, 'warning', duration)
  },
  
  // 信息通知
  info(message, duration = 3000) {
    this.show(message, 'info', duration)
  },
  
  // 隐藏 toast
  hide() {
    this.state.show = false
    this.state.traceId = ''
  }
}

export default toastService
