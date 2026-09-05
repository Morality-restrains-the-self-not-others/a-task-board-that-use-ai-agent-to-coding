import { Navbar, Sidebar } from './layouts.js'

const Login = () => import('../views/Login.vue')
const AdminLogin = () => import('../views/AdminLogin.vue')
const Register = () => import('../views/Register.vue')
const Onboarding = () => import('../views/Onboarding.vue')

export const publicRoutes = [
  {
    path: '/onboarding/',
    name: 'onboarding',
    component: Onboarding,
  },
  {
    path: '/',
    name: 'home',
    components: {
      Navbar,
      default: () => import('../views/Home.vue')
    },
  },
  {
    path: '/pricing/',
    name: 'pricing',
    components: {
      Navbar,
      default: () => import('../views/Pricing.vue')
    },
  },
  {
    path: '/acknowledgments/',
    name: 'acknowledgments',
    components: {
      Navbar,
      default: () => import('../views/Acknowledgments.vue')
    },
  },
  {
    path: '/faq/',
    name: 'faq',
    components: {
      Navbar,
      default: () => import('../views/Faq.vue')
    },
  },
  { 
    path: '/login/',
    name: 'login',
    components: {
      Navbar,
      default: Login
    },
  },
  {
    path: '/auth/login/',
    name: 'auth_login',
    components: {
      Navbar,
      default: Login
    },
  },
  {
    path: '/auth/admin-login/',
    name: 'auth_admin_login',
    components: {
      default: AdminLogin
    },
  },
  { 
    path: '/register/',
    name: 'register',
    components: {
      Navbar,
      default: Register
    },
  },
  { 
    path: '/projects/',
    name: 'projects_root',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/Projects.vue')
    },
  },
  { 
    path: '/auth/register/',
    name: 'auth_register',
    components: {
      Navbar,
      default: Register
    },
  },
  {
    path: '/auth/reset-password-request/',
    name: 'reset_password_request',
    components: {
      Navbar,
      default: () => import('../views/ResetPasswordRequest.vue')
    },
  },
  {
    path: '/auth/reset-password/',
    name: 'reset_password',
    components: {
      Navbar,
      default: () => import('../views/ResetPassword.vue')
    },
  },
  {
    path: '/auth/reset-password/:token/',
    redirect: to => {
      return {
        path: '/auth/reset-password/',
        query: { token: to.params.token }
      }
    },
  },
  {
    path: '/reset-password/',
    name: 'reset_password_alt',
    components: {
      Navbar,
      default: () => import('../views/ResetPassword.vue')
    },
  },
  {
    path: '/reset-password/:token/',
    redirect: to => {
      return {
        path: '/reset-password/',
        query: { token: to.params.token }
      }
    },
  },
  {
    path: '/oauth/github-app/callback/',
    name: 'github_app_callback_continue',
    components: {
      Navbar,
      default: () => import('../views/GithubAppCallbackContinue.vue')
    },
  },
  {
    // Git OAuth 回调落地页（OPT-20260807-071）：/redirect/gitsite/* 被 APISIX
    // spa-catch-all 落入 SPA（nginx 仅转发 /api/*），由本页解析参数后转
    // 服务端交换端点 /api/git-oauth/<provider>-callback/，完成换票闭环。
    path: '/redirect/gitsite/:gitsite/oauth/callback/',
    name: 'git_site_oauth_callback_landing',
    components: {
      Navbar,
      default: () => import('../views/GitSiteOAuthCallbackLanding.vue')
    },
  },
  {
    path: '/activate/:token',
    redirect: to => ({ path: `/activate/${to.params.token}/` }),
  },
  {
    path: '/activate/:token/',
    name: 'activate',
    components: {
      Navbar,
      default: () => import('../views/Activation.vue')
    },
  },
  {
    path: '/auth/activate/:token',
    redirect: to => ({ path: `/auth/activate/${to.params.token}/` }),
  },
  {
    path: '/auth/activate/:token/',
    name: 'auth_activate',
    components: {
      Navbar,
      default: () => import('../views/Activation.vue')
    },
  },
  {
    path: '/auth/unsubscribe/',
    name: 'auth_unsubscribe',
    components: {
      Navbar,
      default: () => import('../views/UnsubscribeConfirm.vue')
    },
  },
]
