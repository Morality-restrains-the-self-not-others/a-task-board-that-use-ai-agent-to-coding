import { createRouter, createWebHistory } from 'vue-router'
// OPT-20260811-005：陈旧租户统一路由守卫（清库/删公司/书签残留后 URL 仍指向已删除公司时
// 按现有公司跳转）。仅此一处网络校验，其余鉴权仍延迟到后端 API（见文件头注释）。
import { installTenantRouteGuard } from './utils/tenantRouteGuard.js'
import { publicRoutes } from './router/publicRoutes.js'
import { tenantRoutes } from './router/tenantRoutes.js'
import { adminRoutes } from './router/adminRoutes.js'
// 鉴权模型：无统一 beforeEach guard — 认证/租户校验全部延迟到后端 API，
// 401/403 由各组件自行处理（跳登录/提示）。路由不再携带 requiresAuth 等
// meta 标记（OPT-20260806-006：此前 25+ 路由的 meta 无任何消费者，纯文档性
// 标记且具误导性，如 /onboarding/ 标 requiresAuth 但必须可未登录访问）。

const getUserInfo = () => {
  if (typeof window !== 'undefined' && window.currentUser) {
    return window.currentUser
  }
  return {
    isAuthenticated: false,
    isSuperuser: false,
    username: ''
  }
}

const routes = [
  ...publicRoutes,
  ...tenantRoutes,
  ...adminRoutes,
]

const handleNotFound = () => {
  const path = window.location.pathname
  const projectMatch = path.match(/^\/projects\/(\d+)\/?$/)

  if (projectMatch) {
    const userInfo = getUserInfo()
    void userInfo
    console.log('[Router] 检测到格式错误的项目详情URL，重定向到首页')
    return { name: 'home' }
  }

  console.log('[Router] 未找到匹配的路由，重定向到首页')
  return { name: 'home' }
}

routes.push({
  path: '/:pathMatch(.*)*',
  name: 'not-found',
  redirect: handleNotFound
})

const router = createRouter({
  history: createWebHistory(),
  routes
})

installTenantRouteGuard(router)

export default router
