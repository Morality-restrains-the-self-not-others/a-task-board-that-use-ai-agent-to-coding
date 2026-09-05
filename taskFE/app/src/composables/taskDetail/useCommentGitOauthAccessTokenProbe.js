import { computed, ref, watch } from 'vue'
import { extractTraceId } from '../../utils/traceId.js'
import {
  collectDisplayedCommentOauthRepoUrls,
  commentOauthUrlsNeedingAccessTokenProbe,
  mergeReadinessAfterAccessTokenProbe,
} from '../../utils/commentExecutionGitOauth.js'
import { probeCommentGitOauthAccessTokensOnce } from '../../utils/probeCommentGitOauthAccessTokensOnce.js'

/**
 * 评论区显示后，对 DB 显示有效的 Git OAuth 绑定做一次 AccessToken 探测（禁止轮询）。
 * check_failed 只记一次，不自动重试；用户可对失败 URL 显式「重试」（OPT-20260902-011）。
 *
 * @param {object} args
 * @param {import('vue').Ref<unknown[]>|import('vue').ComputedRef<unknown[]>} args.displayComments
 * @param {import('vue').Ref<unknown[]>|import('vue').ComputedRef<unknown[]>} args.fallbackRepoIdentities
 * @param {import('vue').Ref<object|null>|import('vue').ComputedRef<object|null>} args.repoOAuthReadiness
 * @param {(event: string, payload: object) => void} [args.emit]
 */
export function useCommentGitOauthAccessTokenProbe({
  displayComments,
  fallbackRepoIdentities,
  repoOAuthReadiness,
  emit,
}) {
  const probing = ref(false)
  const invalidRepoUrls = ref([])
  const unreachableRepoUrls = ref([])
  const checkFailedRepoUrls = ref([])
  const probeError = ref('')
  const probeTraceId = ref('')
  const probedUrls = new Set()
  let inFlight = false

  const oauthReadinessForDetails = computed(() => mergeReadinessAfterAccessTokenProbe(
    repoOAuthReadiness.value,
    {
      probing: probing.value,
      invalidRepoUrls: invalidRepoUrls.value,
      unreachableRepoUrls: unreachableRepoUrls.value,
      checkFailedRepoUrls: checkFailedRepoUrls.value,
      probeError: probeError.value,
      probeTraceId: probeTraceId.value,
    },
  ))

  function applyProbeResult(result) {
    const invalid = Array.isArray(result?.invalidRepoUrls) ? result.invalidRepoUrls : []
    const unreachable = Array.isArray(result?.unreachableRepoUrls) ? result.unreachableRepoUrls : []
    const failed = Array.isArray(result?.checkFailedRepoUrls) ? result.checkFailedRepoUrls : []
    if (invalid.length > 0) {
      const next = new Set(invalidRepoUrls.value)
      for (const url of invalid) next.add(url)
      invalidRepoUrls.value = Array.from(next)
    }
    if (unreachable.length > 0) {
      const next = new Set(unreachableRepoUrls.value)
      for (const url of unreachable) next.add(url)
      unreachableRepoUrls.value = Array.from(next)
    }
    if (failed.length > 0) {
      const next = new Set(checkFailedRepoUrls.value)
      for (const url of failed) next.add(url)
      checkFailedRepoUrls.value = Array.from(next)
      if (!probeError.value) {
        probeError.value = String(result?.probeError || '').trim()
          || '无法确认 Git OAuth AccessToken 是否有效'
        probeTraceId.value = String(result?.probeTraceId || '').trim()
      }
    }
  }

  /**
   * 对给定 URL 跑一次探测。默认仅探测「摘要仍为已绑定」的 URL（自动路径）；
   * force=true 时忽略该门禁，用于用户显式重试已 check_failed 的 URL。
   */
  async function probeRepoUrls(urls, { force = false } = {}) {
    if (inFlight || urls.length === 0) return
    inFlight = true
    probing.value = true
    try {
      const result = await probeCommentGitOauthAccessTokensOnce({
        urls,
        readiness: repoOAuthReadiness.value,
        probedUrls,
        force,
      })
      applyProbeResult(result)
    } catch (error) {
      const next = new Set(checkFailedRepoUrls.value)
      for (const url of urls) next.add(url)
      checkFailedRepoUrls.value = Array.from(next)
      if (!probeError.value) {
        probeError.value = error?.message || '无法确认 Git OAuth AccessToken 是否有效'
        probeTraceId.value = extractTraceId(error)
      }
    } finally {
      probing.value = false
      inFlight = false
      emit?.('repo-oauth-readiness', oauthReadinessForDetails.value)
    }
  }

  watch(
    [displayComments, fallbackRepoIdentities, repoOAuthReadiness],
    async () => {
      if (inFlight) return
      const urls = collectDisplayedCommentOauthRepoUrls(
        displayComments.value,
        fallbackRepoIdentities.value,
      )
      const need = commentOauthUrlsNeedingAccessTokenProbe(urls, repoOAuthReadiness.value)
      const toProbe = need.filter((url) => !probedUrls.has(url))
      if (toProbe.length === 0) return
      await probeRepoUrls(toProbe)
    },
    { immediate: true },
  )

  /**
   * 用户点击 check_failed 徽标旁「重试」：把这些 URL 移出 probedUrls 并清掉失败态，
   * 然后强制再探测一次（仍不弹错误窗）。OPT-20260902-011。
   *
   * @param {string|string[]} targetUrls
   */
  function retryProbeFor(targetUrls) {
    const urls = (Array.isArray(targetUrls) ? targetUrls : [targetUrls])
      .map((url) => String(url || '').trim())
      .filter(Boolean)
    if (urls.length === 0 || inFlight) return
    const urlSet = new Set(urls)
    for (const url of urlSet) probedUrls.delete(url)
    checkFailedRepoUrls.value = checkFailedRepoUrls.value.filter((url) => !urlSet.has(url))
    unreachableRepoUrls.value = unreachableRepoUrls.value.filter((url) => !urlSet.has(url))
    invalidRepoUrls.value = invalidRepoUrls.value.filter((url) => !urlSet.has(url))
    if (checkFailedRepoUrls.value.length === 0) {
      probeError.value = ''
      probeTraceId.value = ''
    }
    emit?.('repo-oauth-readiness', oauthReadinessForDetails.value)
    void probeRepoUrls(urls, { force: true })
  }

  return { oauthReadinessForDetails, retryProbeFor }
}
