import { Navbar, SystemAdminSidebar } from './layouts.js'

export const adminRoutes = [
  {
    path: '/system-admin/',
    name: 'system_admin',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdmin.vue')
    },
  },
  {
    path: '/system-admin/users/',
    name: 'system_admin_users',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminUsers.vue')
    },
  },
  {
    path: '/system-admin/users/:userId/login-history/',
    name: 'system_admin_user_login_history',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminUserLoginHistory.vue')
    },
  },
  {
    path: '/system-admin/tenants/:tenantId/',
    name: 'system_admin_tenant_detail',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminTenantDetail.vue')
    },
  },
  {
    path: '/system-admin/oidc-extension/',
    name: 'system_admin_oidc_extension',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminBrowserExtension.vue')
    },
  },
  {
    path: '/system-admin/grant-points/',
    name: 'system_admin_grant_points',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminGrantPoints.vue')
    },
  },
  {
    path: '/system-admin/order-records/',
    name: 'system_admin_order_records',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminOrderRecords.vue')
    },
  },
  {
    // Compulsory redirect：独立退款页已并入订单与退款 Tab
    path: '/system-admin/refund-applications/',
    name: 'system_admin_refund_applications',
    redirect: { path: '/system-admin/order-records/', query: { tab: 'refund' } },
  },
  {
    path: '/system-admin/referral-management/',
    name: 'system_admin_referral_management',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminReferralManagement.vue')
    },
  },
  {
    path: '/system-admin/deliverable-system/',
    name: 'system_admin_deliverable_system',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminDeliverableSystem.vue')
    },
  },
  {
    path: '/system-admin/default-column-management/',
    name: 'system_admin_default_column_management',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminDefaultColumnManagement.vue')
    },
  },
  {
    path: '/system-admin/price-management/',
    name: 'system_admin_price_management',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminPriceManagement.vue')
    },
  },
  {
    path: '/system-admin/feedback-links/',
    name: 'system_admin_feedback_links',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminFeedbackLinks.vue')
    },
  },
  {
    path: '/system-admin/gitlab-resources/',
    name: 'system_admin_gitlab_resources',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminGitlabResources.vue')
    },
  },
  {
    path: '/system-admin/recommended-llm-providers/',
    name: 'system_admin_recommended_llm_providers',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminRecommendedLLMProviders.vue')
    },
  },
  {
    path: '/system-admin/sub-token-providers/',
    name: 'system_admin_sub_token_providers',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminSubTokenProviders.vue')
    },
  },
  {
    path: '/system-admin/privacy-policy/',
    name: 'system_admin_privacy_policy',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminPrivacyPolicy.vue')
    },
  },
  {
    path: '/system-admin/license-agreement/',
    name: 'system_admin_license_agreement',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminLicenseAgreement.vue')
    },
  },
  {
    path: '/system-admin/login-payment-policy/',
    name: 'system_admin_login_payment_policy',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminLoginPaymentPolicy.vue')
    },
  },
  // OPT-20260806-046: 容器镜像管理（孤儿页面挂载）
  {
    path: '/system-admin/container-images/',
    name: 'system_admin_container_images',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminContainerImages.vue')
    },
  },
  {
    path: '/system-admin/step-full-cos/',
    name: 'system_admin_step_full_cos',
    components: {
      Navbar,
      Sidebar: SystemAdminSidebar,
      default: () => import('../views/SystemAdminStepFullCOS.vue')
    },
  },
]
