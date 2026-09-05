// 共享用户公司列表缓存
// 替代 window.__navbarCompanies（项目规则禁止 window 全局属性）
// Navbar.logic.vue（生产者）→ router.js（消费者）

let _userCompanies = []

/**
 * 设置当前用户的公司 ID 列表。
 * 由 Navbar.logic.vue 在 /me/ API 成功返回后调用。
 * @param {Array<string|number>} companyIds
 */
export function setUserCompanies(companyIds) {
  _userCompanies = Array.isArray(companyIds) ? companyIds.map(String) : []
}

/**
 * 获取当前用户的公司 ID 列表（字符串数组）。
 * 由 router.js 的导航守卫在 profile API 校验失败时回退使用。
 * @returns {string[]}
 */
export function getUserCompanies() {
  return _userCompanies
}
