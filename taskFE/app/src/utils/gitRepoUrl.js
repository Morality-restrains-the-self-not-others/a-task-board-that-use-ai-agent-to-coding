/**
 * 仓库 URL 规范化比较：任务记录地址 vs 项目当前 git_repos。
 * 忽略 .git 后缀、尾斜杠与大小写（与 taskTaskService canonicalGitRepoURL 对齐）。
 */

export function canonicalGitRepoUrl(raw) {
  let s = String(raw || '').trim().toLowerCase()
  s = s.replace(/\/+$/, '')
  if (s.endsWith('.git')) {
    s = s.slice(0, -4)
  }
  return s
}

export function gitRepoUrlsEqual(a, b) {
  const ca = canonicalGitRepoUrl(a)
  const cb = canonicalGitRepoUrl(b)
  return Boolean(ca) && ca === cb
}

/** 两侧均非空且规范化后不相等 → 任务记录与项目当前地址不一致。 */
export function gitRepoAddressMismatch(stored, current) {
  const s = String(stored || '').trim()
  const c = String(current || '').trim()
  if (!s || !c) return false
  return !gitRepoUrlsEqual(s, c)
}
