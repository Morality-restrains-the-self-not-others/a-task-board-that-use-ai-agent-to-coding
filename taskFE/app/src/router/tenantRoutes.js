import { Navbar, Sidebar } from './layouts.js'

export const tenantRoutes = [
  {
    path: '/tenant/:tenant/projects/',
    name: 'projects',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/Projects.vue')
    },
  },
  {
    path: '/tenant/:tenant/projects/:id/',
    name: 'project_detail',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/ProjectDetail.vue')
    },
  },
  {
    path: '/tenant/:tenant/projects/:id/edit/',
    name: 'project_edit',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/ProjectEdit.vue')
    },
  },
  {
    path: '/tenant/:tenant/create-project/',
    name: 'create_project',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/CreateProject.vue')
    },
  },
  {  
    path: '/tenant/:tenant/create-workspace/',
    name: 'create_workspace',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/CreateWorkspace.vue')
    },
  },
  // 人员管理相关路由
  {
    path: '/tenant/:tenant/people/invite/',
    name: 'people_invite',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/PeopleInvite.vue')
    },
  },
  {
    path: '/tenant/:tenant/people/join/',
    name: 'people_join',
    components: {
      Navbar,
      default: () => import('../views/PeopleJoin.vue')
    },
  },
  {
    path: '/tenant/:tenant/people/manage/',
    name: 'people_manage',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/PeopleManage.vue')
    },
  },
  {
    path: '/tenant/:tenant/people/groups/',
    name: 'people_groups',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/PeopleGroups.vue')
    },
  },
  {
    path: '/tenant/:tenant/people/access/',
    name: 'people_access',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/PeopleAccess.vue')
    },
  },
  {
    path: '/tenant/:tenant/people/roles/',
    name: 'people_roles',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/PeopleRoles.vue')
    },
  },

  {
    path: '/tenant/:tenant/user-console/',
    name: 'user_console',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/UserConsole.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/',
    name: 'tenant_user_profile',
    components: {
      Navbar,
      default: () => import('../views/UserProfile.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/access-tokens/',
    name: 'tenant_user_access_tokens',
    components: {
      Navbar,
      default: () => import('../views/UserAccessTokens.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/inbox/',
    name: 'tenant_user_inbox',
    components: {
      Navbar,
      default: () => import('../views/UserInbox.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/login-history/',
    name: 'tenant_user_login_history',
    components: {
      Navbar,
      default: () => import('../views/UserLoginHistory.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/referral/',
    name: 'tenant_user_referral',
    components: {
      Navbar,
      default: () => import('../views/UserReferral.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/git-identities/',
    name: 'tenant_user_git_identities',
    components: {
      Navbar,
      default: () => import('../views/UserGitIdentities.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/git-site-oauth/',
    name: 'tenant_user_git_site_oauth',
    components: {
      Navbar,
      default: () => import('../views/UserGitSiteOAuthSettings.vue')
    },
  },
  {
    path: '/profile/',
    name: 'user_profile',
    components: {
      Navbar,
      default: () => import('../views/UserProfile.vue')
    },
  },
  {
    path: '/profile/referral/',
    name: 'user_referral',
    components: {
      Navbar,
      default: () => import('../views/UserReferral.vue')
    },
  },
  {
    path: '/profile/git-identities/',
    name: 'user_git_identities',
    components: {
      Navbar,
      default: () => import('../views/UserGitIdentities.vue')
    },
  },
  {
    path: '/profile/git-site-oauth/',
    name: 'user_git_site_oauth',
    components: {
      Navbar,
      default: () => import('../views/UserGitSiteOAuthSettings.vue')
    },
  },
  {
    path: '/user/:id/profile/',
    name: 'user_profile_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserProfile.vue')
    },
  },
  {
    path: '/user/:id/profile/referral/',
    name: 'user_referral_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserReferral.vue')
    },
  },
  {
    path: '/user/:id/profile/git-identities/',
    name: 'user_git_identities_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserGitIdentities.vue')
    },
  },
  {
    path: '/user/:id/profile/git-site-oauth/',
    name: 'user_git_site_oauth_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserGitSiteOAuthSettings.vue')
    },
  },
  {
    path: '/user/:id/profile/access-tokens/',
    name: 'user_access_tokens_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserAccessTokens.vue')
    },
  },
  {
    path: '/user/:id/profile/inbox/',
    name: 'user_inbox_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserInbox.vue')
    },
  },
  {
    path: '/user/:id/profile/login-history/',
    name: 'user_login_history_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserLoginHistory.vue')
    },
  },
  {
    path: '/user/:id/profile/feature-params/',
    name: 'user_feature_params_with_id',
    components: {
      Navbar,
      default: () => import('../views/PersonalFeatureParamsConfigs.vue')
    },
  },
  {
    path: '/profile/access-tokens/',
    name: 'user_access_tokens',
    components: {
      Navbar,
      default: () => import('../views/UserAccessTokens.vue')
    },
  },
  {
    path: '/profile/inbox/',
    name: 'user_inbox',
    components: {
      Navbar,
      default: () => import('../views/UserInbox.vue')
    },
  },
  {
    path: '/profile/login-history/',
    name: 'user_login_history',
    components: {
      Navbar,
      default: () => import('../views/UserLoginHistory.vue')
    },
  },
  {
    path: '/profile/feature-params/',
    name: 'user_feature_params',
    components: {
      Navbar,
      default: () => import('../views/PersonalFeatureParamsConfigs.vue')
    },
  },
  {
    path: '/profile/company-settings/',
    name: 'user_company_settings',
    components: {
      Navbar,
      default: () => import('../views/UserCompanySettings.vue')
    },
  },
  {
    path: '/user/:id/profile/company-settings/',
    name: 'user_company_settings_with_id',
    components: {
      Navbar,
      default: () => import('../views/UserCompanySettings.vue')
    },
  },
  {
    path: '/tenant/:tenant/profile/company-settings/',
    name: 'tenant_user_company_settings',
    components: {
      Navbar,
      default: () => import('../views/UserCompanySettings.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/',
    name: 'tenant_settings',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettings.vue')
    },
  },

  {
    path: '/tenant/:tenant/settings/status/',
    name: 'tenant_settings_status',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsStatus.vue')
    },
  },
  // 注销阻断项曾误指向不存在的 GitLab 计费页；别名避免 catch-all 踢回首页
  {
    path: '/tenant/:tenant/billing/gitlab-resources/',
    redirect: (to) => `/tenant/${to.params.tenant}/settings/gitlab-connection/`,
  },
  {
    path: '/tenant/:tenant/settings/members/',
    redirect: (to) => `/tenant/${to.params.tenant}/people/manage/`,
  },
  {
    path: '/tenant/:tenant/workspace/:workspaceId/task/:taskId/',
    redirect: (to) => `/tenant/${to.params.tenant}/workspace/${to.params.workspaceId}/task-detail/${to.params.taskId}/`,
  },
  // 计费管理路由
  {
    path: '/tenant/:tenant/billing/',
    name: 'billing_dashboard',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/BillingDashboard.vue')
    },
  },
  {
    path: '/tenant/:tenant/billing/orders/',
    name: 'billing_orders',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/BillingOrders.vue')
    },
  },
  {
    path: '/tenant/:tenant/billing/orders/create/',
    name: 'order_create',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/OrderCreate.vue')
    },
  },
  {
    path: '/tenant/:tenant/billing/orders/:orderId/',
    name: 'billing_order_detail',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/OrderDetail.vue')
    },
  },
  {
    path: '/tenant/:tenant/billing/transactions/',
    name: 'billing_transactions',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/BillingTransactions.vue')
    },
  },
  {
    path: '/tenant/:tenant/billing/usage/',
    name: 'billing_usage',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/BillingUsage.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/cloud-platform/',
    name: 'tenant_settings_cloud_platform',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsCloudPlatform.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/gitlab-connection/',
    name: 'tenant_settings_gitlab_connection',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsGitlabConnection.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/task-panel/',
    name: 'tenant_settings_task_panel',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsTaskPanel.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/feature-params/',
    name: 'tenant_settings_feature_params',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsFeatureParams.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/workspace/:workspace/feature-params/',
    name: 'workspace_feature_params_settings',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceFeatureParamsSettings.vue')
    },
  },
  {
    path: '/tenant/:tenant/settings/llm-budget/',
    redirect: (to) => `/tenant/${to.params.tenant}/settings/feature-params/`,
  },
  {
    path: '/tenant/:tenant/settings/company/',
    name: 'tenant_settings_company',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/TenantCompanySettings.vue')
    },
  },
  {
    path: '/tenant/:tenant/deliverable-systems/',
    name: 'tenant_deliverable_systems',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/DeliverableSystemList.vue')
    },
  },
  {
    path: '/tenant/:tenant/workspace/:id/settings/',
    name: 'workspace_settings',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettings.vue')
    },
  },
  {
    path: '/tenant/:tenant/workspace/:id/settings/cloud-platform/',
    name: 'workspace_settings_cloud_platform',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsCloudPlatform.vue')
    },
  },
  {
    path: '/tenant/:tenant/workspace/:id/settings/status/',
    name: 'workspace_settings_status',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceSettingsStatus.vue')
    },
  },

  {
    path: '/tenant/:tenant/work-panel/',
    name: 'work_panel',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkPanel.vue')
    },
  },
  {
    path: '/tenant/:tenant/queue-schedule/',
    name: 'workspace_queue_schedule',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/WorkspaceQueueSchedule.vue')
    },
  },
  {
    path: '/tenant/:tenant/image-market/',
    name: 'image_market',
    components: {
      Navbar,
      Sidebar,
      default: () => import('../views/ImageMarket.vue')
    },
  },
  {
    path: '/tenant/:tenant/workspace/:workspaceId/task-detail/:taskId/',
    name: 'task_detail',
    components: {
      Navbar,
      default: () => import('../components/TaskDetailContent.logic.vue')
    },
  },
]
