/**
 * Layer graph git push / submit-and-push handlers.
 */
import { postContainerCompute } from './containerComputeRequest.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import {
  gitOauthUnboundReasonForRepoUrls,
  repoUrlsFromTaskRepoRows,
} from '../../utils/gitOauthPushPrecheck.js'
import { commentOAuthGrantMissingReason } from '../../utils/commentOAuthGrantCheck.js'
import { rememberGrantTicketFromSearch } from '../../utils/grantTicketSession.js'
import { messageFromFailedResponse } from '../../utils/httpError.js'
import { handleActionChunkLoadError } from '../../utils/chunkLoadGuard.js'
import { normalizeLayerGitDirty } from '../../utils/layerZtreeNodes.js'
import {
  clearLastOAuthExchangeError,
  isOAuthExchangeFailureMessage,
  setLastOAuthExchangeError,
} from './lastOAuthExchangeErrorState.js'
import { recordGitPrReplyComment } from './taskDetailGitPrReply.js'

/**
 * HTTPS GitHub/GitLab 推送前检查 user-app-connection。
 * allowBare（dev-local-token）跳过。失败返回 true，调用方应中止且不得 commit/push。
 * @returns {Promise<boolean>}
 */
export async function blockLayerGitOauthIfUnbound(deps, actionLabel) {
  const isDevLocalToken =
    typeof deps.containerPageUrl?.value === 'string' &&
    deps.containerPageUrl.value.includes('/ui/dev-local-token')
  try {
    if (typeof window !== 'undefined' && window.location) {
      rememberGrantTicketFromSearch(window.location?.search || '')
    }
    const repoUrls = repoUrlsFromTaskRepoRows(deps.taskRepoRows?.value)
    const reason = await gitOauthUnboundReasonForRepoUrls(
      repoUrls,
      // 推送预检与评论芯片同一根因：DB connected 不等于 AccessToken 仍有效。
      // probeAccessToken 实时校验，refresh 失效时先拦截再让服务端换票失败。
      { actionLabel, allowBare: isDevLocalToken, probeAccessToken: true },
    )
    if (reason) {
      showRequestError(reason)
      return true
    }
    if (isDevLocalToken) return false
    const grantReason = commentOAuthGrantMissingReason(deps, repoUrls, actionLabel)
    if (!grantReason) return false
    showRequestError(grantReason)
    return true
  } catch (err) {
    showRequestError(err?.message || '无法检查 Git OAuth 绑定', err)
    return true
  }
}

/**
 * 推送成功后乐观更新层快照：ahead→0，并记录 last_pushed_count 供「N 个提交已推送」展示。
 * 若有 PR html_url，写入 pr_html_url 供 zTree「PR」按钮跳转审查页。
 * @param {{ value: object|null }} snapshotRef
 * @param {string} layerId
 * @param {{ pushedCount?: number, gitRemote?: object, prHtmlUrl?: string }} [opts]
 */
export function markLayerGitRemotePushedInSnapshot(snapshotRef, layerId, opts = {}) {
  const snap = snapshotRef?.value
  if (!snap || !Array.isArray(snap.layers)) return
  const lid = String(layerId || '').trim()
  if (!lid) return
  const forcedPushed =
    typeof opts.pushedCount === 'number' && Number.isFinite(opts.pushedCount) && opts.pushedCount > 0
      ? Math.floor(opts.pushedCount)
      : null
  const remoteFromResp =
    opts.gitRemote && typeof opts.gitRemote === 'object' ? opts.gitRemote : null
  const prHtmlUrl =
    typeof opts.prHtmlUrl === 'string' && opts.prHtmlUrl.trim() ? opts.prHtmlUrl.trim() : ''
  const nextLayers = snap.layers.map((layer) => {
    if (String(layer?.layer_id || '').trim() !== lid) return layer
    const prevGr =
      layer.git_remote && typeof layer.git_remote === 'object' ? { ...layer.git_remote } : {}
    let prevAhead = null
    if (typeof prevGr.ahead === 'number' && Number.isFinite(prevGr.ahead) && prevGr.ahead >= 0) {
      prevAhead = Math.floor(prevGr.ahead)
    } else if (typeof prevGr.ahead === 'string' && prevGr.ahead.trim() !== '') {
      const n = parseInt(prevGr.ahead, 10)
      if (Number.isFinite(n) && n >= 0) prevAhead = n
    }
    const prevPushed =
      typeof prevGr.last_pushed_count === 'number' &&
      Number.isFinite(prevGr.last_pushed_count) &&
      prevGr.last_pushed_count > 0
        ? Math.floor(prevGr.last_pushed_count)
        : null
    const pushed = forcedPushed ?? (prevAhead !== null && prevAhead > 0 ? prevAhead : null) ?? prevPushed ?? 1
    const mergedRemote = {
      ...prevGr,
      ...(remoteFromResp || {}),
      is_git: true,
      no_upstream: false,
      ahead: 0,
      last_pushed_count: pushed,
    }
    if (prHtmlUrl) mergedRemote.pr_html_url = prHtmlUrl
    return { ...layer, git_remote: mergedRemote }
  })
  snapshotRef.value = { ...snap, layers: nextLayers }
}

export async function onLayerGraphLayerPush(node, deps) {
  const {
    containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
    taskRepoRows, repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId,
    layerGraphPushTargetBranch, layerGraphBusyActionKey,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    refreshLayerGraphFromServer, refreshZTreeExecutionLog, fetchTaskDetail,
    layerGraphSnapshot,
  } = deps
  if (!containerEndpointRegistered.value) { window.alert('容器业务端点尚未就绪，无法推送'); return }
  if (containerHttpUnreachable.value) { showRequestError('容器当前无法连接，请待恢复后重试'); return }
  const layerId = String(node?.layerId || '').trim()
  if (!layerId) return
  const targetBranch = layerGraphPushTargetBranch.value
  if (!targetBranch) { window.alert('缺少工作分支：请配置 projects[].target_branch，或 branch_strategy.work_branch_name / target_branch_name'); return }
  if (taskRepoRows.value.length) {
    const allChosen = taskRepoRows.value.every((r) => Boolean(String(repoCloneIdentityIdForUrl(r.url) || '').trim()))
    if (!allChosen) { window.alert('请为所有关联仓库选择「Git 提交身份」后再推送'); return }
  }
  const pushIdentityId = firstTaskRepoCloneIdentityId()
  const isDevLocalToken =
    typeof containerPageUrl.value === 'string' && containerPageUrl.value.includes('/ui/dev-local-token')
  const useContainerRemoteForMultiRepo = taskRepoRows.value.length > 1
  // 多仓仍 prefer_container_remote（跳过单一 identity 门禁），但必须换到 OAuth；
  // 禁止无 token 时回退裸 git/push（会报 terminal prompts disabled）。
  const preferContainerRemote = useContainerRemoteForMultiRepo || isDevLocalToken
  const allowBareGitPush = isDevLocalToken
  const requestIdentityId = String(pushIdentityId || '')
  if (!preferContainerRemote && !pushIdentityId) { window.alert('请先为关联仓库选择「Git 提交身份」后再推送'); return }
  if (preferContainerRemote && !allowBareGitPush && !pushIdentityId && taskRepoRows.value.length) {
    window.alert('请为关联仓库选择「Git 提交身份」后再推送')
    return
  }
  layerGraphBusyActionKey.value = `push:layer:${layerId}`
  try {
    const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) return
    if (await blockLayerGitOauthIfUnbound(deps, '推送')) return
    // 多仓也传一个 GitHub repo_url 作为换票 hint，避免 collectTask 失败时 githubRows 为空 → 裸 push
    const repoUrlHint = (() => {
      const rows = Array.isArray(taskRepoRows.value) ? taskRepoRows.value : []
      if (rows.length === 1) return String(rows[0]?.url || '').trim()
      const gh = rows.find((r) => /github\.com[:/]/i.test(String(r?.url || '')))
      if (gh) return String(gh.url || '').trim()
      return String(rows[0]?.url || '').trim()
    })()
    const requestBody = {
      layer_id: layerId, target_branch: targetBranch,
      container_page_url: containerPageUrl.value || '', identity_id: requestIdentityId,
      prefer_container_remote: preferContainerRemote,
      allow_bare_git_push: allowBareGitPush,
      repo_url: repoUrlHint,
      wait_for_pr: true,
    }
    const resp = await postContainerCompute(deps, 'container-layer-git-push', requestBody)
    if (!resp.ok) {
      const msg = messageFromFailedResponse(resp, `HTTP ${resp.status}`)
      markContainerTransportUnreachableIfForwardingFailed(resp.status, msg)
      // 409/502 换票失败：写入关联项目区旁提示，避免「OAuth 已授权」绿标误导
      if (
        (resp.status === 409 || resp.status === 502 || resp.status === 401 || resp.status === 403)
        && isOAuthExchangeFailureMessage(msg)
      ) {
        setLastOAuthExchangeError(msg)
      }
      showRequestError(`推送失败：${msg}`, resp); return
    }
    const successData = await resp.json().catch(() => ({}))
    markContainerTransportOk()
    clearLastOAuthExchangeError()
    layerGraphBusyActionKey.value = ''
    const pr = successData.github_pull_request
    const prHtmlUrl = typeof pr?.html_url === 'string' ? pr.html_url.trim() : ''
    const prCompareUrl = typeof pr?.compare_url === 'string' ? pr.compare_url.trim() : ''
    const shouldOpenCompareUrl =
      prCompareUrl &&
      pr?.skipped !== 'no_github_repo' &&
      pr?.skipped !== 'disabled_by_env' &&
      (!pr?.queued || Boolean(pr?.wait_timeout))
    // 有 html_url：写入快照供 zTree「PR」按钮跳转，不自动打开
    if (!prHtmlUrl && (pr?.skipped === 'pr_api_error' || pr?.skipped === 'exception')) {
      const detail = pr?.skipped === 'pr_api_error' ? 'GitHub API 未创建 PR（常见：GitHub App 对目标仓库无写权限，或仓库未在 App 安装范围内）。可点击下方对比链接手动发起 PR。' : 'GitHub 自动创建 PR 时发生异常。若需 PR 请从仓库端手动创建。'
      window.alert(`推送成功。${detail}`)
      if (shouldOpenCompareUrl) window.open(prCompareUrl, '_blank', 'noopener,noreferrer')
    } else if (!prHtmlUrl && shouldOpenCompareUrl) window.open(prCompareUrl, '_blank', 'noopener,noreferrer')
    else if (!prHtmlUrl && pr?.skipped === 'credential_not_approved') window.alert('推送成功。未自动创建 PR：请先在任务详情「批准在本任务使用凭据」并连接 GitHub。')
    else if (!prHtmlUrl && pr?.skipped === 'github_app_not_connected') window.alert('推送成功。未自动创建 PR：请先在任务详情绑定 GitHub App 用户授权。')
    else if (!prHtmlUrl && pr?.skipped === 'github_account_not_selected_for_repo') {
      window.alert('推送成功。未自动创建 PR：当前仓库尚未选择 GitHub 授权账号，请在任务详情按仓库完成选择。')
      if (shouldOpenCompareUrl) window.open(prCompareUrl, '_blank', 'noopener,noreferrer')
    } else if (!prHtmlUrl && pr?.skipped === 'disabled_by_env') console.info('[LayerPush] PR 自动创建已按服务端配置跳过')
    else if (!prHtmlUrl && pr?.wait_timeout && pr?.queued) {
      window.alert('推送成功。Pull Request 仍在创建中，请稍后在仓库或任务评论中查看链接。')
    }
    let pushedCount = null
    if (layerGraphSnapshot?.value && Array.isArray(layerGraphSnapshot.value.layers)) {
      const layerRow = layerGraphSnapshot.value.layers.find(
        (l) => String(l?.layer_id || '').trim() === layerId,
      )
      const rawAhead = layerRow?.git_remote?.ahead
      if (typeof rawAhead === 'number' && Number.isFinite(rawAhead) && rawAhead > 0) {
        pushedCount = Math.floor(rawAhead)
      } else if (typeof rawAhead === 'string' && rawAhead.trim() !== '') {
        const n = parseInt(rawAhead, 10)
        if (Number.isFinite(n) && n > 0) pushedCount = n
      }
    }
    // 立即更新 zTree：隐藏推送按钮、展示「N 个提交已推送」，有 PR 则露出 PR 按钮
    if (layerGraphSnapshot) {
      markLayerGitRemotePushedInSnapshot(layerGraphSnapshot, layerId, {
        pushedCount: pushedCount ?? undefined,
        gitRemote: successData.git_remote && typeof successData.git_remote === 'object' ? successData.git_remote : undefined,
        prHtmlUrl: prHtmlUrl || undefined,
      })
    }
    await refreshLayerGraphFromServer(true)
    // 刷新可能带回 ahead:0 但无 last_pushed_count / pr_html_url，再次合并
    if (layerGraphSnapshot) {
      markLayerGitRemotePushedInSnapshot(layerGraphSnapshot, layerId, {
        pushedCount: pushedCount ?? undefined,
        gitRemote: successData.git_remote && typeof successData.git_remote === 'object' ? successData.git_remote : undefined,
        prHtmlUrl: prHtmlUrl || undefined,
      })
    }
    void refreshZTreeExecutionLog()
    if (prHtmlUrl) {
      await recordGitPrReplyComment(deps, prHtmlUrl)
    }
    void fetchTaskDetail()
  } catch (err) { console.error('[LayerPush] 推送异常:', err); showRequestError('网络错误，请稍后重试', err) }
  finally { layerGraphBusyActionKey.value = '' }
}

export async function onLayerGraphLayerSubmitAndPush(node, deps) {
  const {
    containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
    taskRepoRows, repoCloneIdentityIdForUrl, firstTaskRepoCloneIdentityId,
    layerGraphPushTargetBranch, layerGitGithubAppOauthConnected, layerGraphBusyActionKey,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    refreshLayerGraphFromServer, refreshZTreeExecutionLog, fetchTaskDetail,
    layerGraphSnapshot, layerChangesByLayerId,
  } = deps
  if (!containerEndpointRegistered.value) { window.alert('容器业务端点尚未就绪，无法提交并推送'); return }
  if (containerHttpUnreachable.value) { showRequestError('容器当前无法连接，请待恢复后重试'); return }
  const layerId = String(node?.layerId || '').trim()
  if (!layerId) return
  const label = `submit_push:layer:${layerId}`
  layerGraphBusyActionKey.value = label
  try {
    const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) return
    if (await blockLayerGitOauthIfUnbound(deps, '提交并推送')) return

    // Step 1: commit if dirty
    const snap = layerGraphSnapshot?.value
    const layers = Array.isArray(snap?.layers) ? snap.layers : []
    const layer = layers.find((l) => String(l?.layer_id || '').trim() === layerId)
    const dirty = layer ? normalizeLayerGitDirty(layer.git_worktree_dirty) : null

    if (dirty === true) {
      if (taskRepoRows.value.length) {
        const allChosen = taskRepoRows.value.every((r) => Boolean(String(repoCloneIdentityIdForUrl(r.url) || '').trim()))
        if (!allChosen) { window.alert('请为所有关联仓库选择「Git 提交身份」后再提交'); return }
      }
      const commitResp = await postContainerCompute(deps, 'container-layer-git-commit', {
        layer_id: layerId, container_page_url: containerPageUrl.value || '', identity_id: '',
      })
      if (!commitResp.ok) {
        const msg = messageFromFailedResponse(commitResp, `HTTP ${commitResp.status}`)
        markContainerTransportUnreachableIfForwardingFailed(commitResp.status, msg)
        showRequestError(`提交失败：${msg}`, commitResp); return
      }
      markContainerTransportOk()
      const rest = { ...layerChangesByLayerId.value }; delete rest[layerId]
      layerChangesByLayerId.value = rest
      await refreshLayerGraphFromServer(true)
    }

    // Step 2: push + create PR
    const snap2 = layerGraphSnapshot?.value
    const layers2 = Array.isArray(snap2?.layers) ? snap2.layers : []
    const layer2 = layers2.find((l) => String(l?.layer_id || '').trim() === layerId)
    const gr = layer2?.git_remote
    const hasUnpushed = !!(gr && typeof gr === 'object' && gr.is_git !== false && gr.no_upstream !== true
      && (() => { const a = gr.ahead; return (typeof a === 'number' && a > 0) || (typeof a === 'string' && parseInt(a, 10) > 0) })())

    if (!hasUnpushed) {
      window.alert('提交成功。当前分支无可推送的提交。')
      return
    }

    const targetBranch = layerGraphPushTargetBranch.value
    if (!targetBranch) { window.alert('缺少工作分支：请配置 projects[].target_branch，或 branch_strategy.work_branch_name / target_branch_name'); return }
    if (taskRepoRows.value.length) {
      const allChosen = taskRepoRows.value.every((r) => Boolean(String(repoCloneIdentityIdForUrl(r.url) || '').trim()))
      if (!allChosen) { window.alert('请为所有关联仓库选择「Git 提交身份」后再推送'); return }
    }
    const pushIdentityId = firstTaskRepoCloneIdentityId()
    const isDevLocalToken =
      typeof containerPageUrl.value === 'string' && containerPageUrl.value.includes('/ui/dev-local-token')
    const useContainerRemoteForMultiRepo = taskRepoRows.value.length > 1
    const preferContainerRemote = useContainerRemoteForMultiRepo || isDevLocalToken
    const allowBareGitPush = isDevLocalToken
    const requestIdentityId = String(pushIdentityId || '')
    if (!preferContainerRemote && !pushIdentityId) { window.alert('请先为关联仓库选择「Git 提交身份」后再推送'); return }
    if (preferContainerRemote && !allowBareGitPush && !pushIdentityId && taskRepoRows.value.length) {
      window.alert('请为关联仓库选择「Git 提交身份」后再推送')
      return
    }

    const repoUrlHint = (() => {
      const rows = Array.isArray(taskRepoRows.value) ? taskRepoRows.value : []
      if (rows.length === 1) return String(rows[0]?.url || '').trim()
      const gh = rows.find((r) => /github\.com[:/]/i.test(String(r?.url || '')))
      if (gh) return String(gh.url || '').trim()
      return String(rows[0]?.url || '').trim()
    })()

    const pushResp = await postContainerCompute(deps, 'container-layer-git-push', {
      layer_id: layerId, target_branch: targetBranch,
      container_page_url: containerPageUrl.value || '', identity_id: requestIdentityId,
      prefer_container_remote: preferContainerRemote,
      allow_bare_git_push: allowBareGitPush,
      repo_url: repoUrlHint,
      wait_for_pr: true,
    })

    if (!pushResp.ok) {
      const msg = messageFromFailedResponse(pushResp, `HTTP ${pushResp.status}`)
      markContainerTransportUnreachableIfForwardingFailed(pushResp.status, msg)
      if (
        (pushResp.status === 409 || pushResp.status === 502 || pushResp.status === 401 || pushResp.status === 403)
        && isOAuthExchangeFailureMessage(msg)
      ) {
        setLastOAuthExchangeError(msg)
      }
      showRequestError(`提交成功但推送失败：${msg}`, pushResp); return
    }

    markContainerTransportOk()
    clearLastOAuthExchangeError()

    const successData = await pushResp.json().catch(() => ({}))
    const pr = successData.github_pull_request
    const prHtmlUrl = typeof pr?.html_url === 'string' ? pr.html_url.trim() : ''
    const prCompareUrl = typeof pr?.compare_url === 'string' ? pr.compare_url.trim() : ''
    const shouldOpenCompareUrl =
      prCompareUrl &&
      pr?.skipped !== 'no_github_repo' &&
      pr?.skipped !== 'disabled_by_env' &&
      (!pr?.queued || Boolean(pr?.wait_timeout))

    if (!prHtmlUrl && (pr?.skipped === 'pr_api_error' || pr?.skipped === 'exception')) {
      const detail = pr?.skipped === 'pr_api_error' ? 'GitHub API 未创建 PR（常见：GitHub App 对目标仓库无写权限，或仓库未在 App 安装范围内）。可点击下方对比链接手动发起 PR。' : 'GitHub 自动创建 PR 时发生异常。若需 PR 请从仓库端手动创建。'
      window.alert(`提交并推送成功。${detail}`)
      if (shouldOpenCompareUrl) window.open(prCompareUrl, '_blank', 'noopener,noreferrer')
    } else if (!prHtmlUrl && shouldOpenCompareUrl) window.open(prCompareUrl, '_blank', 'noopener,noreferrer')
    else if (!prHtmlUrl && pr?.skipped === 'credential_not_approved') window.alert('提交并推送成功。未自动创建 PR：请先在任务详情「批准在本任务使用凭据」并连接 GitHub。')
    else if (!prHtmlUrl && pr?.skipped === 'github_app_not_connected') window.alert('提交并推送成功。未自动创建 PR：请先在任务详情绑定 GitHub App 用户授权。')
    else if (!prHtmlUrl && pr?.skipped === 'github_account_not_selected_for_repo') {
      window.alert('提交并推送成功。未自动创建 PR：当前仓库尚未选择 GitHub 授权账号，请在任务详情按仓库完成选择。')
      if (shouldOpenCompareUrl) window.open(prCompareUrl, '_blank', 'noopener,noreferrer')
    } else if (!prHtmlUrl && pr?.skipped === 'disabled_by_env') console.info('[SubmitAndPush] PR 自动创建已按服务端配置跳过')
    else if (!prHtmlUrl && pr?.wait_timeout && pr?.queued) {
      window.alert('提交并推送成功。Pull Request 仍在创建中，请稍后在仓库或任务评论中查看链接。')
    } else if (prHtmlUrl) {
      window.alert(`提交并推送成功！PR 已创建：${prHtmlUrl}`)
    }

    // Update snapshot
    if (layerGraphSnapshot) {
      let pushedCount = null
      const snap3 = layerGraphSnapshot.value
      if (snap3 && Array.isArray(snap3.layers)) {
        const lr = snap3.layers.find((l) => String(l?.layer_id || '').trim() === layerId)
        const rawAhead = lr?.git_remote?.ahead
        if (typeof rawAhead === 'number' && Number.isFinite(rawAhead) && rawAhead > 0) pushedCount = Math.floor(rawAhead)
      }
      markLayerGitRemotePushedInSnapshot(layerGraphSnapshot, layerId, {
        pushedCount: pushedCount ?? undefined,
        gitRemote: successData.git_remote && typeof successData.git_remote === 'object' ? successData.git_remote : undefined,
        prHtmlUrl: prHtmlUrl || undefined,
      })
    }
    await refreshLayerGraphFromServer(true)
    if (layerGraphSnapshot) {
      markLayerGitRemotePushedInSnapshot(layerGraphSnapshot, layerId, {
        pushedCount: undefined,
        gitRemote: successData.git_remote && typeof successData.git_remote === 'object' ? successData.git_remote : undefined,
        prHtmlUrl: prHtmlUrl || undefined,
      })
    }
    void refreshZTreeExecutionLog()
    if (prHtmlUrl) {
      await recordGitPrReplyComment(deps, prHtmlUrl)
    }
    void fetchTaskDetail()
  } catch (err) {
    console.error('[SubmitAndPush] 异常:', err)
    if (handleActionChunkLoadError(err, { showError: showRequestError })) return
    showRequestError('网络错误，请稍后重试', err)
  }
  finally { layerGraphBusyActionKey.value = '' }
}
