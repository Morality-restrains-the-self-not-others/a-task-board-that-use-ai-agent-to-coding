/**
 * useQrCodeCanvas — 统一的 QR 码 Canvas 渲染 composable
 *
 * OPT-20260726-028: 提取自 PayOrderModal.vue 和 OrderCreate.vue 的重复代码。
 * 使用顶层静态 import 将 qrcode 打入主 bundle，消除动态 chunk 加载失败风险。
 * 回退链：本地 qrcode.toCanvas → 外部 api.qrserver.com 图片 → 错误提示文字
 */

import { nextTick } from 'vue'
import QRCodeLib from 'qrcode'

// qrcode CJS 模块经 Vite 转换后 default 为 exports 对象 { toCanvas, toDataURL, ... }
// 同时 named exports 也可用；优先使用 default.toCanvas
const QRCode = (QRCodeLib && typeof QRCodeLib.toCanvas === 'function')
  ? QRCodeLib
  : (QRCodeLib && QRCodeLib.default && typeof QRCodeLib.default.toCanvas === 'function')
    ? QRCodeLib.default
    : null

/**
 * 在 Canvas 上渲染 QR 码
 *
 * @param {HTMLCanvasElement} canvas - 目标 canvas 元素
 * @param {string} text - 要编码的文本（如微信支付 code_url）
 * @param {{ width?: number, margin?: number }} [opts]
 * @returns {Promise<{ success: boolean, method: 'qrcode'|'external'|'error', error?: string }>}
 */
export async function renderQrToCanvas(canvas, text, opts = {}) {
  const { width = 200, margin = 1 } = opts

  if (!canvas || !text) {
    return { success: false, method: 'error', error: '缺少 canvas 或 text 参数' }
  }

  await nextTick()

  // ———— 方法 1: 本地 qrcode npm 包（顶层静态 import，打入主 bundle）————
  if (QRCode && typeof QRCode.toCanvas === 'function') {
    try {
      await QRCode.toCanvas(canvas, text, { width, margin })
      return { success: true, method: 'qrcode' }
    } catch (e) {
      console.warn('[useQrCodeCanvas] qrcode.toCanvas 失败:', e.message)
    }
  } else {
    console.warn('[useQrCodeCanvas] qrcode 模块不可用，回退到外部服务')
  }

  // ———— 方法 2: 外部 QR 服务图片回退 ————
  try {
    await new Promise((resolve, reject) => {
      const ctx = canvas.getContext('2d')
      const img = new Image()
      img.crossOrigin = 'anonymous'
      const timeout = setTimeout(() => {
        img.src = ''
        reject(new Error('外部 QR 服务超时'))
      }, 8000)

      img.onload = () => {
        clearTimeout(timeout)
        ctx.clearRect(0, 0, canvas.width, canvas.height)
        ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
        resolve()
      }
      img.onerror = () => {
        clearTimeout(timeout)
        reject(new Error('外部 QR 图片加载失败'))
      }
      img.src = `https://api.qrserver.com/v1/create-qr-code/?size=${width}x${width}&data=${encodeURIComponent(text)}`
    })
    return { success: true, method: 'external' }
  } catch (e) {
    console.warn('[useQrCodeCanvas] 外部服务回退失败:', e.message)
  }

  // ———— 方法 3: 显示错误提示 ————
  const ctx = canvas.getContext('2d')
  ctx.fillStyle = '#f0f0f0'
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  ctx.fillStyle = '#999'
  ctx.font = '12px sans-serif'
  ctx.textAlign = 'center'
  ctx.fillText('QR 加载失败', canvas.width / 2, canvas.height / 2)

  return { success: false, method: 'error', error: '所有 QR 渲染方式均失败' }
}
