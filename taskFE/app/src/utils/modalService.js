// 模态弹窗服务
import { reactive } from 'vue'
import { extractTraceId } from './traceId.js'

const modalService = {
  // 存储模态弹窗状态（使用 reactive 使状态变为响应式）
  state: reactive({
    show: false,
    title: '提示',
    message: '',
    type: 'alert', // alert or confirm
    onConfirm: null,
    onCancel: null,
    confirmText: '确定',
    cancelText: '取消',
    additionalActions: [], // 额外的操作按钮
    traceId: '', // 请求失败时挂到 Modal 根节点 data-traceId
    showCloseButton: true,
  }),
  
  // 显示提示弹窗；options.traceId 或 Error.traceId 用于 data-traceId
  alert(message, title = '提示', options = {}) {
    return new Promise((resolve) => {
      this.state.show = true
      let traceId = extractTraceId(options.traceId) || extractTraceId(options.source)
      
      // 处理message为对象的情况
      if (typeof message === 'object' && message !== null) {
        if (message instanceof Error) {
          traceId = traceId || extractTraceId(message)
          this.state.message = message.message
          this.state.title = title
          this.state.onConfirm = resolve
          this.state.additionalActions = []
        } else if (message.message) {
          this.state.message = message.message
          traceId = traceId || extractTraceId(message)
          if (message.title) {
            this.state.title = message.title
          } else {
            this.state.title = title
          }
          if (message.onConfirm) {
            this.state.onConfirm = message.onConfirm
          } else {
            this.state.onConfirm = resolve
          }
          if (message.additionalActions) {
            this.state.additionalActions = message.additionalActions
          } else {
            this.state.additionalActions = []
          }
        } else {
          this.state.message = JSON.stringify(message)
          this.state.title = title
          this.state.onConfirm = resolve
          this.state.additionalActions = []
        }
      } else {
        this.state.title = title
        this.state.message = message
        this.state.onConfirm = resolve
        this.state.additionalActions = Array.isArray(options.additionalActions)
          ? options.additionalActions
          : []
      }

      this.state.traceId = traceId
      this.state.type = 'alert'
      this.state.onCancel = null
      this.state.confirmText = options.confirmText || '确定'
      this.state.showCloseButton = options.showCloseButton !== false
      
      // 如果有自动关闭选项
      if (options.autoClose) {
        setTimeout(() => {
          this.close()
          resolve()
        }, options.autoClose)
      }
    })
  },
  
  // 显示确认弹窗
  confirm(message, title = '确认', confirmText = '确定', cancelText = '取消') {
    return new Promise((resolve, reject) => {
      this.state.show = true
      this.state.title = title
      this.state.message = message
      this.state.type = 'confirm'
      this.state.confirmText = confirmText
      this.state.cancelText = cancelText
      this.state.onConfirm = resolve
      this.state.onCancel = reject
      this.state.additionalActions = []
      this.state.traceId = ''
      this.state.showCloseButton = true
    })
  },
  
  // 关闭弹窗
  close() {
    this.state.show = false
    this.state.traceId = ''
  },
  
  // 处理确认
  handleConfirm() {
    if (this.state.onConfirm) {
      this.state.onConfirm(true)
    }
    this.close()
  },
  
  // 处理取消
  handleCancel() {
    if (this.state.onCancel) {
      this.state.onCancel(false)
    }
    this.close()
  }
}

export default modalService
