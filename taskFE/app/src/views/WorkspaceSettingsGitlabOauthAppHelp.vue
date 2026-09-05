<template>
  <details class="text-sm border border-border rounded-lg p-3" data-testid="gitlab-oauth-app-help">
    <summary class="cursor-pointer font-medium text-text">如何在自建 GitLab 创建 OAuth Application（Git 网站授权）</summary>
    <ol class="mt-3 list-decimal pl-5 space-y-2 text-text-light">
      <li>GitLab Admin → Applications → New application。</li>
      <li>
        Redirect URI 必须与本页只读框完全一致：
        <code class="font-mono text-xs break-all text-text">{{ redirectUri || '（加载后显示）' }}</code>
      </li>
      <li>勾选 Confidential。</li>
      <li>Scopes 选择 <code class="font-mono text-xs">api</code>、<code class="font-mono text-xs">read_user</code>、<code class="font-mono text-xs">read_repository</code>、<code class="font-mono text-xs">write_repository</code>。</li>
      <li>不要把 <code class="font-mono text-xs">openid</code> 当成「用平台账号登录 GitLab」——那是下面另一条 OIDC 链路。</li>
    </ol>
    <!-- Anti-Replay-OK: 只读说明，无写操作 -->
  </details>
</template>

<script setup>
defineProps({
  redirectUri: { type: String, default: '' },
})
</script>
