<template>
  <aside class="w-[220px] bg-white rounded-xl shadow-sm border border-border p-4 h-fit">
    <h3 class="text-base font-semibold text-text mb-3">账号中心</h3>
    <nav class="space-y-1">
      <router-link
        :to="profilePath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'profile' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
      >
        个人资料
      </router-link>
      <router-link
        :to="inboxPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'inbox' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
        data-testid="user-center-nav-inbox"
      >
        收信箱
      </router-link>
      <router-link
        :to="loginHistoryPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'login-history' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
        data-testid="user-center-nav-login-history"
      >
        登录历史
      </router-link>
      <router-link
        :to="referralPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'referral' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
      >
        推荐
      </router-link>
      <router-link
        :to="accessTokensPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'access-tokens' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
      >
        访问令牌
      </router-link>
      <router-link
        :to="gitIdentityPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'git-identities' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
        data-testid="user-center-nav-git-identities"
      >
        Git 提交身份
      </router-link>
      <router-link
        :to="gitSiteOauthPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'git-site-oauth' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
        data-testid="user-center-nav-git-site-oauth"
      >
        Git 网站 OAuth
      </router-link>
      <router-link
        :to="featureParamsPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'feature-params' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
      >
        智能体资源配置
      </router-link>
      <router-link
        :to="companySettingsPath"
        class="block px-3 py-2 rounded-lg text-sm transition-colors"
        :class="activeMenu === 'company-settings' ? 'bg-primary/10 text-primary font-medium' : 'text-text hover:bg-primary/5 hover:text-primary'"
      >
        公司设置
      </router-link>
    </nav>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { getCookie } from '../utils/cookieUtils'

const props = defineProps({
  tenantId: {
    type: String,
    default: ''
  },
  activeMenu: {
    type: String,
    default: 'profile'
  }
})

const route = useRoute()

const profilePath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/',
    query: accessCode ? { accessCode } : {}
  }
})

const inboxPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/inbox/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/inbox/',
    query: accessCode ? { accessCode } : {}
  }
})

const loginHistoryPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/login-history/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/login-history/',
    query: accessCode ? { accessCode } : {}
  }
})

const referralPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/referral/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/referral/',
    query: accessCode ? { accessCode } : {}
  }
})

const gitIdentityPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/git-identities/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/git-identities/',
    query: accessCode ? { accessCode } : {}
  }
})

const gitSiteOauthPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/git-site-oauth/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/git-site-oauth/',
    query: accessCode ? { accessCode } : {}
  }
})

const featureParamsPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  if (uid) {
    return `/user/${uid}/profile/feature-params/`
  }
  return '/profile/feature-params/'
})

const accessTokensPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/access-tokens/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/access-tokens/',
    query: accessCode ? { accessCode } : {}
  }
})

const companySettingsPath = computed(() => {
  const uid = String(getCookie('userId') || '').trim()
  const accessCode = route.query.accessCode ? String(route.query.accessCode) : ''
  if (uid) {
    return {
      path: `/user/${uid}/profile/company-settings/`,
      query: accessCode ? { accessCode } : {}
    }
  }
  return {
    path: '/profile/company-settings/',
    query: accessCode ? { accessCode } : {}
  }
})
</script>
