/** 路由布局懒加载（避免 router.js 同步解析大模块） */
export const Navbar = () => import('../components/Navbar.logic.vue')
export const Sidebar = () => import('../components/Sidebar.vue')
export const SystemAdminSidebar = () => import('../components/SystemAdminSidebar.vue')
