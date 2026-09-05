<template>
  <div class="max-w-lg mx-auto px-6 py-16 text-center text-sm text-text-light">
    正在跳回授权前的页面…
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { consumeGithubAppReturnTarget } from '../utils/githubAppReturnStorage'
import { rememberGrantTicketFromSearch } from '../utils/grantTicketSession'
import {
  hrefToRouterLocation,
  resolveGithubAppContinueHref,
} from '../utils/githubAppContinueNav'

const route = useRoute()
const router = useRouter()

onMounted(() => {
  rememberGrantTicketFromSearch(window.location.search || route.fullPath || '')
  const returnKeyRaw = route.query.returnKey
  const returnKey = returnKeyRaw != null ? String(returnKeyRaw).trim() : ''
  const provider = route.query.provider != null ? String(route.query.provider).trim() : 'github'
  const providerParam = route.query[provider] != null ? String(route.query[provider]) : 'ok'
  const traceId = String(route.query.trace_id || route.query.traceId || '').trim()

  if (!returnKey) {
    router.replace({ path: '/profile/' })
    return
  }

  const target = consumeGithubAppReturnTarget(returnKey)
  if (!target) {
    const expiredQuery = { provider, [provider]: 'return_expired' }
    if (traceId) expiredQuery.trace_id = traceId
    router.replace({ path: '/profile/git-site-oauth/', query: expiredQuery })
    return
  }

  const finalPath = resolveGithubAppContinueHref({
    returnUrl: target,
    provider,
    providerParam,
    traceId,
  })
  router.replace(hrefToRouterLocation(finalPath))
})
</script>
