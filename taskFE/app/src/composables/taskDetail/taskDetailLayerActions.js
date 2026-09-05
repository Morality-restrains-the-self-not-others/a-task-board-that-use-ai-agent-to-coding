/**
 * Pure functions: layer graph action handlers for TaskDetail.
 */
import { postContainerCompute } from './containerComputeRequest.js'
import { showRequestError } from '../../utils/requestErrorDisplay.js'
import { messageFromFailedResponse } from '../../utils/httpError.js'

export {
  markLayerGitRemotePushedInSnapshot,
  onLayerGraphLayerPush,
  onLayerGraphLayerSubmitAndPush,
} from './taskDetailLayerPushActions.js'

export async function onLayerGraphLayerSubmit(node, deps) {
  const {
    containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
    taskRepoRows, repoCloneIdentityIdForUrl, layerGraphBusyActionKey,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    layerChangesByLayerId, refreshLayerGraphFromServer, refreshZTreeExecutionLog,
  } = deps
  if (!containerEndpointRegistered.value) { window.alert('容器业务端点尚未就绪，无法提交'); return }
  if (containerHttpUnreachable.value) { showRequestError('容器当前无法连接，请待恢复后重试'); return }
  const layerId = String(node?.layerId || '').trim()
  if (!layerId) return
  if (taskRepoRows.value.length) {
    const allChosen = taskRepoRows.value.every((r) => Boolean(String(repoCloneIdentityIdForUrl(r.url) || '').trim()))
    if (!allChosen) { window.alert('请为所有关联仓库选择「Git 提交身份」后再提交'); return }
  }
  layerGraphBusyActionKey.value = `submit:layer:${layerId}`
  try {
    const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) return
    const resp = await postContainerCompute(deps, 'container-layer-git-commit', {
      layer_id: layerId, container_page_url: containerPageUrl.value || '', identity_id: '',
    })
    if (!resp.ok) {
      const msg = messageFromFailedResponse(resp, `HTTP ${resp.status}`)
      markContainerTransportUnreachableIfForwardingFailed(resp.status, msg)
      showRequestError(`提交失败：${msg}`, resp); return
    }
    markContainerTransportOk()
    let commitBody = {}; try { commitBody = await resp.json() } catch { commitBody = {} }
    const commitSummary = typeof commitBody.summary === 'string' ? commitBody.summary.trim() : ''
    if (commitSummary) window.alert(commitSummary)
    else if (commitBody.status === 'noop' && commitBody.detail) window.alert(String(commitBody.detail))
    const rest = { ...layerChangesByLayerId.value }; delete rest[layerId]
    layerChangesByLayerId.value = rest
    await refreshLayerGraphFromServer(true); await refreshZTreeExecutionLog()
  } catch (err) { console.error('提交失败', err); showRequestError('网络错误，请稍后重试', err) }
  finally { layerGraphBusyActionKey.value = '' }
}

export async function onLayerChangesListCommitStaged(payload, deps) {
  const {
    containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
    taskRepoRows, repoCloneIdentityIdForUrl, layerChangesByLayerId,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    refreshLayerGraphFromServer, refreshZTreeExecutionLog, refreshSelectedLayerChanges,
    bumpProjectFileTreeRefresh,
  } = deps
  const finish = typeof payload?.finish === 'function' ? payload.finish : () => {}
  const layerId = String(payload?.layerId || '').trim()
  const message = String(payload?.message || '').trim()
  if (!layerId || !message) { finish(); return }
  if (!containerEndpointRegistered.value) { window.alert('容器业务端点尚未就绪，无法提交'); finish(); return }
  if (containerHttpUnreachable.value) { showRequestError('容器当前无法连接，请待恢复后重试'); finish(); return }
  if (taskRepoRows.value.length) {
    const allChosen = taskRepoRows.value.every((r) => Boolean(String(repoCloneIdentityIdForUrl(r.url) || '').trim()))
    if (!allChosen) { window.alert('请为所有关联仓库选择「Git 提交身份」后再提交'); finish(); return }
  }
  let finished = false
  const done = () => { if (finished) return; finished = true; finish() }
  try {
    const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) { done(); return }
    const createResp = await postContainerCompute(deps, 'container-layer-create', {
      parent_layer_id: layerId, container_page_url: containerPageUrl.value || '', layer_kind: 'git_commit', commit_message: message.slice(0, 100),
    })
    if (!createResp.ok) {
      const msg = messageFromFailedResponse(createResp, `HTTP ${createResp.status}`)
      showRequestError(`创建可写层失败：${msg}`, createResp); done(); return
    }
    markContainerTransportOk()
    let createLayerBody = {}; try { createLayerBody = await createResp.json() } catch { createLayerBody = {} }
    const newLayerId = String(createLayerBody.layer_id || '').trim()
    if (!newLayerId) { showRequestError('创建可写层失败：未返回新层ID', createLayerBody); done(); return }
    const resp = await postContainerCompute(deps, 'container-layer-git-commit', {
      layer_id: layerId, message, stage_all: false, container_page_url: containerPageUrl.value || '', identity_id: '',
    })
    if (!resp.ok) {
      const msg = messageFromFailedResponse(resp, `HTTP ${resp.status}`)
      markContainerTransportUnreachableIfForwardingFailed(resp.status, msg)
      showRequestError(`提交失败：${msg}`, resp); done(); return
    }
    markContainerTransportOk()
    let commitBody = {}; try { commitBody = await resp.json() } catch { commitBody = {} }
    const commitSummary = typeof commitBody.summary === 'string' ? commitBody.summary.trim() : ''
    if (commitSummary) window.alert(commitSummary)
    else if (commitBody.status === 'noop' && commitBody.detail) window.alert(String(commitBody.detail))
    const rest = { ...layerChangesByLayerId.value }; delete rest[layerId]
    layerChangesByLayerId.value = rest
    await refreshLayerGraphFromServer(true); await refreshZTreeExecutionLog()
    await refreshSelectedLayerChanges(); bumpProjectFileTreeRefresh()
  } catch (err) { console.error('文件变动列表提交失败', err); showRequestError('网络错误，请稍后重试', err) }
  finally { done() }
}


export async function onLayerGraphLayerMerge(node, deps) {
  const {
    containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
    layerGraphMergeTargetBranch, layerGraphBusyActionKey,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    refreshLayerGraphFromServer, refreshZTreeExecutionLog,
  } = deps
  if (!containerEndpointRegistered.value) { window.alert('容器业务端点尚未就绪，无法合并'); return }
  if (containerHttpUnreachable.value) { showRequestError('容器当前无法连接，请待恢复后重试'); return }
  const layerId = String(node?.layerId || '').trim()
  if (!layerId) return
  const targetBranch = String(layerGraphMergeTargetBranch?.value || '').trim()
  if (!targetBranch) {
    window.alert('缺少合并目标分支：请在分支策略中配置 merge_target_branch_name')
    return
  }
  layerGraphBusyActionKey.value = `merge:layer:${layerId}`
  try {
    const tenantId = effectiveTenantId.value
    const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) return
    const resp = await postContainerCompute(deps, 'container-layer-git-merge', {
      layer_id: layerId,
      target_branch: targetBranch,
      container_page_url: containerPageUrl.value || '',
    })
    if (!resp.ok) {
      const msg = messageFromFailedResponse(resp, `HTTP ${resp.status}`)
      markContainerTransportUnreachableIfForwardingFailed(resp.status, msg)
      showRequestError(`合并失败：${msg}`, resp)
      return
    }
    markContainerTransportOk()
    let body = {}
    try { body = await resp.json() } catch { body = {} }
    const summary = typeof body.summary === 'string' ? body.summary.trim() : ''
    if (body.status === 'noop' && body.detail) {
      window.alert(String(body.detail))
    } else if (summary) {
      window.alert(`合并成功：${summary}`)
    } else {
      window.alert(`合并成功：已将当前分支合并进 ${targetBranch}`)
    }
    await refreshLayerGraphFromServer(true)
    await refreshZTreeExecutionLog()
  } catch (err) {
    console.error('[LayerMerge] 合并异常:', err)
    showRequestError('网络错误，请稍后重试', err)
  } finally {
    layerGraphBusyActionKey.value = ''
  }
}

export async function onLayerGraphLayerSubmitAndMerge(node, deps) {
  const {
    containerEndpointRegistered, containerHttpUnreachable, containerPageUrl,
    taskRepoRows, repoCloneIdentityIdForUrl,
    layerGraphMergeTargetBranch, layerGraphBusyActionKey,
    effectiveTenantId, effectiveWorkspaceId, effectiveTaskId,
    markContainerTransportUnreachableIfForwardingFailed, markContainerTransportOk,
    refreshLayerGraphFromServer, refreshZTreeExecutionLog,
    layerGraphSnapshot, layerChangesByLayerId,
  } = deps
  if (!containerEndpointRegistered.value) { window.alert('容器业务端点尚未就绪，无法提交并合并'); return }
  if (containerHttpUnreachable.value) { showRequestError('容器当前无法连接，请待恢复后重试'); return }
  const layerId = String(node?.layerId || '').trim()
  if (!layerId) return
  const label = `submit_merge:layer:${layerId}`
  layerGraphBusyActionKey.value = label
  try {
    const tenantId = effectiveTenantId.value; const workspaceId = effectiveWorkspaceId.value
    const taskId = effectiveTaskId.value
    if (!tenantId || !workspaceId || !taskId) return

    // Step 1: commit if dirty
    const { normalizeLayerGitDirty } = await import('../../utils/layerZtreeNodes.js')
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

    // Step 2: merge to target branch
    const targetBranch = String(layerGraphMergeTargetBranch?.value || '').trim()
    if (!targetBranch) {
      window.alert('缺少合并目标分支：请在分支策略中配置 merge_target_branch_name')
      return
    }

    const mergeResp = await postContainerCompute(deps, 'container-layer-git-merge', {
      layer_id: layerId, target_branch: targetBranch, container_page_url: containerPageUrl.value || '',
    })

    if (!mergeResp.ok) {
      const msg = messageFromFailedResponse(mergeResp, `HTTP ${mergeResp.status}`)
      markContainerTransportUnreachableIfForwardingFailed(mergeResp.status, msg)
      showRequestError(`提交成功但合并失败：${msg}`, mergeResp); return
    }
    markContainerTransportOk()

    let body = {}
    try { body = await mergeResp.json() } catch { body = {} }
    const summary = typeof body.summary === 'string' ? body.summary.trim() : ''
    if (body.status === 'noop' && body.detail) {
      window.alert(String(body.detail))
    } else if (summary) {
      window.alert(`提交并合并成功：${summary}`)
    } else {
      window.alert(`提交并合并成功：已将变更合并进 ${targetBranch}`)
    }
    await refreshLayerGraphFromServer(true)
    await refreshZTreeExecutionLog()
  } catch (err) { console.error('[SubmitAndMerge] 异常:', err); showRequestError('网络错误，请稍后重试', err) }
  finally { layerGraphBusyActionKey.value = '' }
}
