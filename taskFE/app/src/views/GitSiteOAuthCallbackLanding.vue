<template>
  <div class="max-w-lg mx-auto px-6 py-16 text-center text-sm text-text-light">
    正在完成 Git 授权，请稍候…
  </div>
</template>

<script setup>
/**
 * 落地页：承接浏览器 OAuth 回调路径 /redirect/gitsite/<gitsite>/oauth/callback/
 * （SSOT redirect_uri，GitHub App callback 白名单注册的是该 URL）。
 *
 * 背景（OPT-20260807-071）：base 域 nginx 仅转发 /api/* 到网关，带 Host 的
 * /redirect/gitsite/* 请求被 APISIX spa-catch-all（priority 10, /*）抢匹配，
 * 返回 SPA HTML，回调永远到不了 taskGitOauth 的 /redirect/gitsite/ handler。
 * 修复不改基础设施：由本页解析参数（code/state 等）后原样转发到同域服务端
 * 交换端点 /api/git-oauth/<provider>-callback/（nginx /api 转发 ✓），服务端
 * 完成 authorization_code 换票（redirect_uri 取 state 签名内的值，与 GitHub
 * 授权时一致）后 302 回：
 *   - 成功：/oauth/github-app/callback/?returnKey=<rk>&github=<code>
 *     → GithubAppCallbackContinue 消费 localStorage 的 returnKey → 跳回 next
 *   - 失败：/profile/git-site-oauth/?github=bad_state 等 → 既有回调 toast 消费
 *
 * gitsite → provider 映射与后端 ResolveProviderByGitsite 契约一致：
 * gitsite = provider target.website 的主机名（github.com → github；
 * 自建 GitLab 域 → gitlab）。
 */
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

onMounted(() => {
  const gitsite = String(route.params.gitsite || '').trim().toLowerCase()
  // 与后端 ResolveProviderByGitsite（website hostname == gitsite）保持一致的映射；
  // 新增 provider 时须在此同步（或改为从 provider 目录 API 解析）。
  const provider = gitsite === 'github.com' ? 'github' : 'gitlab'
  if (!gitsite) {
    // 路径不完整（理论上不会发生，SPA fallback 兜底）：落到 OAuth 设置页
    window.location.replace('/profile/git-site-oauth/')
    return
  }
  const qs = new URLSearchParams(route.query).toString()
  // 原样转发全部 query（code/state/error/trace_id 等），服务端按缺参/坏 state
  // 自行归类（bad_state / exchange_rejected / exchange_failed）。
  window.location.replace(
    `/api/git-oauth/${provider}-callback/${qs ? `?${qs}` : ''}`,
  )
})
</script>
