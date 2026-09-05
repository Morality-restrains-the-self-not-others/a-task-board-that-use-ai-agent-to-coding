/** 层快照 git_remote.last_push_error → zTree「push 失败」/「push 无权限」/「未绑定 Git 授权」 */
import {
  buildRepoOAuthStartHref,
  supportsRepoOAuthAuthorize,
} from './repoOAuthAuthorizeUtils.js'

export function isGitPushPermissionDenied(detail) {
  const t = String(detail || '')
  if (!t.trim()) return false
  return (
    /permission to \S+ denied/i.test(t)
    || /write access to repository not granted/i.test(t)
    || /you are not allowed to push/i.test(t)
    || /the requested url returned error:\s*403/i.test(t)
  )
}

export function isGitOauthBindingMissing(detail) {
  const t = String(detail || '')
  if (!t.trim()) return false
  return (
    /BINDING_MISSING/.test(t)
    || /缺少绑定/.test(t)
    || /请先完成该仓库的 Git 授权/.test(t)
    || /请先在创建或编辑任务/.test(t)
    || /请先在任务详情绑定 Git/.test(t)
  )
}

function rewriteBindingMissingCopy(text) {
  return String(text || '')
    .replace(
      /请先在任务详情「关联项目」中为每个仓库选择并保存 Git 授权账号。?/g,
      '请先在创建或编辑任务、或评论「提交并运行」时为每个仓库完成 Git 授权绑定。',
    )
    .replace(
      /请先在任务详情绑定 GitHub 授权账号/g,
      '请先完成该仓库的 Git 授权绑定',
    )
}

function bindingMissingHint(detail) {
  const raw = String(detail || '')
  const jsonStart = raw.indexOf('{')
  if (jsonStart >= 0) {
    try {
      const parsed = JSON.parse(raw.slice(jsonStart))
      const inner = typeof parsed?.detail === 'string' ? parsed.detail.trim() : ''
      if (inner) return rewriteBindingMissingCopy(inner)
      const safe = typeof parsed?.detail_safe === 'string' ? parsed.detail_safe.trim() : ''
      if (safe) return rewriteBindingMissingCopy(safe)
    } catch {
      /* keep rewritten full text */
    }
  }
  const rewritten = rewriteBindingMissingCopy(raw)
  if (rewritten.trim()) return rewritten
  return '请先在创建或编辑任务、或评论「提交并运行」时为每个仓库完成 Git 授权绑定。'
}

/**
 * 从 BINDING_MISSING 文案提取缺失绑定的仓库 match key（host/path，无 scheme）。
 * 例：…（缺少绑定: gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work, github.com/a/b）
 * @param {string} detail
 * @returns {string[]}
 */
export function missingBindingRepoMatchKeys(detail) {
  const raw = String(detail || '')
  const m = raw.match(/缺少绑定[:：]\s*([^）)。]*)/)
  if (!m) return []
  return String(m[1])
    .split(/[，,、;；\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

function repoUrlFromBindingMatchKey(matchKey) {
  const key = String(matchKey || '').trim()
  if (!key) return ''
  if (/^https?:\/\//i.test(key)) return key
  if (/^git@/i.test(key)) return key
  // match key 形如 host/path；OAuth 走站点 https，与 git 协议无关
  return `https://${key}`
}

function bindingMissingBindHref(detail) {
  const keys = missingBindingRepoMatchKeys(detail)
  for (const key of keys) {
    const repoUrl = repoUrlFromBindingMatchKey(key)
    if (!repoUrl || !supportsRepoOAuthAuthorize(repoUrl)) continue
    const href = buildRepoOAuthStartHref(repoUrl)
    if (href) return href
  }
  return ''
}

export function collectLatestLayerPushError(snapshot) {
  const layers = Array.isArray(snapshot?.layers) ? snapshot.layers : []
  const details = []
  for (const layer of layers) {
    const detail = typeof layer?.git_remote?.last_push_error === 'string'
      ? layer.git_remote.last_push_error.trim()
      : ''
    if (detail) details.push(detail)
  }
  const denied = details.find((d) => isGitPushPermissionDenied(d))
  return denied || details[details.length - 1] || ''
}

function permissionDeniedHint(detail) {
  if (!isGitPushPermissionDenied(detail)) return detail
  return `当前授权账号对该仓库无写权限。请换有写权限的账号重新授权。\n${detail}`
}

export function formatPushErrorClipboardText(node) {
  const title = typeof node?.pushErrorTitle === 'string' ? node.pushErrorTitle.trim() : ''
  const label = typeof node?.pushErrorLabel === 'string' ? node.pushErrorLabel.trim() : ''
  const body = title || label
  if (!body) return ''
  const traceId = typeof node?.pushErrorTraceId === 'string' ? node.pushErrorTraceId.trim() : ''
  if (!traceId) return body
  return `${body}\ntraceId: ${traceId}`
}

export function layerPushErrorFields(layer) {
  const empty = {
    pushErrorLabel: '',
    pushErrorTitle: '',
    pushErrorTraceId: '',
    pushErrorKind: '',
  }
  if (!layer || typeof layer !== 'object') return empty
  const gr = layer.git_remote
  if (!gr || typeof gr !== 'object') return empty
  const detail = typeof gr.last_push_error === 'string' ? gr.last_push_error.trim() : ''
  if (!detail) return empty
  const traceId =
    typeof gr.last_push_error_trace_id === 'string' ? gr.last_push_error_trace_id.trim() : ''
  const permission = isGitPushPermissionDenied(detail)
  if (permission) {
    return {
      pushErrorLabel: 'push 无权限',
      pushErrorTitle: permissionDeniedHint(detail),
      pushErrorTraceId: traceId,
      pushErrorKind: 'permission',
    }
  }
  if (isGitOauthBindingMissing(detail)) {
    return {
      pushErrorLabel: '未绑定 Git 授权',
      pushErrorTitle: bindingMissingHint(detail),
      pushErrorTraceId: traceId,
      pushErrorKind: 'binding',
      pushErrorBindHref: bindingMissingBindHref(detail),
    }
  }
  return {
    pushErrorLabel: 'push 失败',
    pushErrorTitle: permissionDeniedHint(detail),
    pushErrorTraceId: traceId,
    pushErrorKind: 'generic',
  }
}
