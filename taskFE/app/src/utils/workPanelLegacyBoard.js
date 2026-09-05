/** 仅在需要遗留 DOM 看板模态时加载（减小登录等页面首包）。
 *  使用直接 import() 字符串字面量，确保 Vite 构建时可以静态分析并正确分块。 */
export async function loadLegacyBoardModalAndComments() {
  if (window.modalModule?.initModals && window.commentsModule?.renderComments) {
    return
  }
  try {
    await import('../js/comments.js')
    await import('../js/modal.js')
  } catch (e) {
    console.warn('遗留看板 modal/comments 加载失败', e)
  }
}
