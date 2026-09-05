/**
 * 仓库 URL 展示工具：http(s) 地址渲染为可跳转外链，其余保持纯文本。
 * 供任务详情关联项目、评论身份区、创建任务仓库行等复用（OPT-20260820-011）。
 * @param {*} url
 * @returns {boolean}
 */
export function isHttpRepoUrl(url) {
  return /^https?:\/\//i.test(String(url || '').trim())
}
