/**
 * 创建任务 / 任务详情：标题翻译失败时的用户可见文案。
 * 上游超时或空 JSON 不得把 json.Unmarshal / fanyi_agent 细节展示到红字；
 * 但须保留后端分类后的「为什么失败」原因。
 */

export const TITLE_TRANSLATION_LOCAL_HINT =
  '已使用本地规则生成分支名。请稍后重试或手动修改。'

export const TITLE_TRANSLATION_USER_ERROR =
  `任务标题暂时无法自动翻译。${TITLE_TRANSLATION_LOCAL_HINT}`

const TECHNICAL_RE =
  /unexpected end of json|fanyi_agent|json input|client\.timeout|context deadline|i\/o timeout|响应无效|响应为空|eof|finish_reason|reasoning_len|reasoning_content/i

function withLocalBranchHint(reason) {
  const base = String(reason || '').trim().replace(/[。．.]+$/u, '')
  if (!base) return TITLE_TRANSLATION_USER_ERROR
  if (base.includes('本地规则生成分支名')) return `${base}${/[。．.]$/u.test(reason) ? '' : '。'}`
  return `${base}。${TITLE_TRANSLATION_LOCAL_HINT}`
}

export function userFacingTitleTranslationError(raw) {
  const s = String(raw || '').trim()
  if (!s) return TITLE_TRANSLATION_USER_ERROR
  if (s === TITLE_TRANSLATION_USER_ERROR) return s
  if (s.includes('本地规则生成分支名') && !TECHNICAL_RE.test(s)) return s
  if (TECHNICAL_RE.test(s)) return TITLE_TRANSLATION_USER_ERROR
  // Backend categorized why: 「任务标题自动翻译失败：…」
  if (s.includes('自动翻译失败：')) {
    return withLocalBranchHint(s)
  }
  // Legacy generic backend copy (no specific why)
  if (s.includes('请稍后重试') || s === '任务标题翻译失败') {
    return TITLE_TRANSLATION_USER_ERROR
  }
  return s
}

export function isTitleTranslateAbortError(error) {
  return error?.name === 'AbortError' || error?.code === 'ABORT_ERR'
}
