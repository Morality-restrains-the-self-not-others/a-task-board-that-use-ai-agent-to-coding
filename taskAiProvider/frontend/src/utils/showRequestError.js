import { extractTraceId, setDataTraceId } from './traceId.js'

/**
 * 在可挂属性的 DOM 上展示请求错误（替代 window.alert）。
 * @param {unknown} message
 * @param {unknown} [source]
 * @param {HTMLElement|null} [targetEl] 已有错误节点；缺省则临时挂 body
 */
export function showRequestError(message, source, targetEl) {
  const text =
    message instanceof Error
      ? message.message
      : String(message ?? '请求失败')
  const tid = extractTraceId(source) || extractTraceId(message)

  if (targetEl) {
    targetEl.textContent = text
    setDataTraceId(targetEl, tid)
    return
  }

  let host = document.getElementById('aip-request-error-banner')
  if (!host) {
    host = document.createElement('div')
    host.id = 'aip-request-error-banner'
    host.className = 'err aip-request-error-banner'
    host.setAttribute('role', 'alert')
    document.body.prepend(host)
  }
  host.textContent = text
  setDataTraceId(host, tid)
  host.style.display = 'block'
}

/**
 * 展示成功/信息提示（替代 window.alert 用于非错误场景）。
 * 显示绿色横幅，3 秒后自动消失。
 * @param {string} message
 */
export function showSuccess(message) {
  let host = document.getElementById('aip-success-banner')
  if (!host) {
    host = document.createElement('div')
    host.id = 'aip-success-banner'
    host.className = 'aip-success-banner'
    host.setAttribute('role', 'status')
    document.body.prepend(host)
  }
  host.textContent = message
  host.style.display = 'block'
  // Auto-dismiss after 3s
  clearTimeout(host._dismissTimer)
  host._dismissTimer = setTimeout(() => {
    host.style.display = 'none'
  }, 3000)
}

/**
 * 确认对话框（替代 window.confirm）。
 * 展示带「确认」「取消」按钮的 banner，返回 Promise<boolean>。
 * @param {string} message
 * @returns {Promise<boolean>}
 */
export function showConfirm(message) {
  return new Promise((resolve) => {
    let host = document.getElementById('aip-confirm-banner')
    if (!host) {
      host = document.createElement('div')
      host.id = 'aip-confirm-banner'
      host.className = 'aip-confirm-banner'
      host.setAttribute('role', 'alertdialog')
      document.body.prepend(host)
    }
    host.innerHTML = ''
    host.style.display = 'flex'

    const msg = document.createElement('span')
    msg.textContent = message
    msg.style.cssText = 'flex:1;margin-right:12px'

    const confirmBtn = document.createElement('button')
    confirmBtn.textContent = '确认'
    confirmBtn.className = 'btn-confirm'
    confirmBtn.onclick = () => { host.style.display = 'none'; resolve(true) }

    const cancelBtn = document.createElement('button')
    cancelBtn.textContent = '取消'
    cancelBtn.className = 'btn-cancel'
    cancelBtn.onclick = () => { host.style.display = 'none'; resolve(false) }

    host.append(msg, confirmBtn, cancelBtn)
  })
}
