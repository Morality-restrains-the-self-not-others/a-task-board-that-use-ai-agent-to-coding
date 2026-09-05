// @ts-check
/**
 * 任务详情 Playwright：统一补齐查询参数，与手工联调 URL 对齐
 * （如 `?accessCode=...&github=ok`）。
 *
 * @param {string} pathWithOptionalQuery 以 / 开头的 path，可带或不带 ?
 * @returns {string}
 */
export function taskDetailPathWithMockQuery(pathWithOptionalQuery) {
  const raw = pathWithOptionalQuery.startsWith('/')
    ? pathWithOptionalQuery
    : `/${pathWithOptionalQuery}`
  const qi = raw.indexOf('?')
  const pathOnly = qi === -1 ? raw : raw.slice(0, qi)
  const sp = new URLSearchParams(qi === -1 ? '' : raw.slice(qi + 1))
  sp.set('github', 'ok')
  return `${pathOnly}?${sp.toString()}`
}
