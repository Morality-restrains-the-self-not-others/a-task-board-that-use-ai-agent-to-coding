<template>
  <div class="bg-white rounded-xl shadow-sm border border-border p-6">
    <h2 class="text-lg font-semibold text-text mb-4">身份绑定（微信）</h2>
    <p class="text-sm text-text-light mb-4">
      绑定微信后可使用微信扫码或微信内一键登录。登录账号与支付账号相互独立：手机号登录的账号同样可以绑定任意微信账号。
    </p>
    <div class="space-y-4">
      <p class="text-sm text-text">
        当前绑定：<span class="font-medium">{{ hasWechat ? wechatAppsLabel : '未绑定' }}</span>
      </p>
      <div class="flex flex-wrap gap-3">
        <template v-if="!hasWechat">
          <button
            v-if="bindAvailable"
            type="button"
            :disabled="isBinding"
            class="px-4 py-2 rounded-lg bg-primary text-white text-sm hover:bg-primary/90 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
            @click="bindWechat"
          >
            {{ isBinding ? '跳转中…' : '绑定微信' }}
          </button>
          <p v-else class="text-sm text-amber-600">
            微信应用尚未配置，请联系管理员
          </p>
        </template>
        <template v-else>
          <button
            v-if="bindAvailable"
            type="button"
            :disabled="isBinding"
            class="px-4 py-2 rounded-lg border border-primary text-primary text-sm hover:bg-primary/10 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
            @click="bindWechat"
          >
            {{ isBinding ? '跳转中…' : '绑定其他微信应用' }}
          </button>
          <button
            type="button"
            :disabled="isUnbinding"
            class="px-4 py-2 rounded-lg border border-red-300 text-danger hover:bg-red-50 disabled:opacity-60 disabled:cursor-not-allowed transition-colors"
            @click="unbindWechat"
          >
            {{ isUnbinding ? '解绑中…' : '解绑微信' }}
          </button>
        </template>
      </div>
      <p v-if="inlineMessage" class="text-sm text-success">{{ inlineMessage }}</p>
      <p v-if="inlineError" class="text-sm text-danger" :data-traceId="traceId || undefined">{{ inlineError }}</p>
    </div>
  </div>
</template>

<script>
import { computed, ref } from 'vue'
import { apiFetch } from '../utils/apiUtils.js'
import { extractTraceId } from '../utils/traceId.js'
import { getActiveToken } from '../domain/auth/services/saved_accounts_store.js'
import { createClickGuard, mergeIdempotencyHeaders } from '../utils/clickGuard.js'

export default {
  name: 'UserProfileWechatBindingPanel',
  props: {
    hasWechat: { type: Boolean, default: false },
    wechatApps: { type: Array, default: () => [] },
    // bindAvailable: 平台微信 web 应用凭据（appId/appSecret）是否已配置。
    // 未配置时绑定必然失败，「绑定其他微信应用」入口无意义 → 隐藏。默认 true 向后兼容。
    bindAvailable: { type: Boolean, default: true },
  },
  emits: ['wechat-updated', 'message', 'error'],
  setup(props, { emit }) {
    const isBinding = ref(false)
    const isUnbinding = ref(false)
    const inlineMessage = ref('')
    const inlineError = ref('')
    const traceId = ref('')

    const appLabels = { web: '网页扫码', inapp: '微信内登录', miniapp: '小程序' }
    const wechatAppsLabel = computed(() => {
      const labels = (props.wechatApps || []).map((k) => appLabels[k] || k)
      return labels.length ? labels.join('、') : '—'
    })

    // 绑定微信 — 浏览器重定向需携带登录态：将 token 短暂写入 cookie（path 限定，10 分钟）
    // 后端 resolveTokenUserIDFromRequest 支持 Cookie token。
    const bindWechat = async () => {
      isBinding.value = true
      try {
        const token = await getActiveToken()
        if (!token) {
          inlineError.value = '登录状态已失效，请重新登录'
          emit('error', '登录状态已失效，请重新登录')
          return
        }
        document.cookie = `token=${encodeURIComponent(token)}; path=/api/auth/wechat; max-age=600; SameSite=Lax`
        window.location.href = '/api/auth/wechat/bind/?app=web'
      } catch (err) {
        inlineError.value = '发起微信绑定失败，请稍后重试'
        emit('error', '发起微信绑定失败')
      } finally {
        isBinding.value = false
      }
    }

    const unbindWechatGuard = createClickGuard()

    // 解绑微信
    const unbindWechat = async () => {
      inlineMessage.value = ''
      inlineError.value = ''
      // OPT-20260819-038: 解绑是账号写操作，防连点/超时重试双发
      await unbindWechatGuard.run(async ({ idempotencyKey }) => {
        isUnbinding.value = true
        try {
          const resp = await apiFetch('/api/auth/wechat/unbind/', {
            method: 'DELETE',
            headers: mergeIdempotencyHeaders({ 'Content-Type': 'application/json' }, idempotencyKey),
            body: JSON.stringify({ app_key: 'web' }),
          })
          if (!resp.ok) {
            const detail = await resp.json().then((d) => d.detail || '解绑失败').catch(() => '解绑失败')
            inlineError.value = detail
            traceId.value = extractTraceId(resp) || ''
            return
          }
          inlineMessage.value = '微信已解绑'
          emit('message', '微信已解绑')
          emit('wechat-updated')
        } catch (err) {
          inlineError.value = '解绑失败，请稍后重试'
          emit('error', '解绑失败')
        } finally {
          isUnbinding.value = false
        }
      })
    }

    return {
      isBinding, isUnbinding, inlineMessage, inlineError, traceId,
      wechatAppsLabel, bindWechat, unbindWechat,
    }
  },
}
</script>
