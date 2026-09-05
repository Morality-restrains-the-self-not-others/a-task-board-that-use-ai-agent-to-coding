# Review — 创建项目 ssh:// Git URL

- **Date:** 2026-09-02
- **CRG:** `code-review-graph update --brief` at start (risk 0). Impact: `isValidGitRepoUrl`, `normalizeGitRepoURLForBranchLookup`, `GitsiteHost`, `repoMatchKeyFromUrl`.

## Five-axis

| Axis | Result |
|------|--------|
| Correctness | ssh:// 格式通过；normalize 转 HTTP(S)；SSH 端口不进入 gitsite/match key。https/git@ 回归绿。 |
| Readability | Go 抽出 `parseSSHGitRemote` / `httpURLForGitHostPath`。 |
| Architecture | 无新 API/事件；沿用 OAuth HTTPS 探测。 |
| Security | 仍拒绝 ftp/javascript/无 path 的 ssh://；不新增出站 SSH 探测。 |
| Performance | 纯字符串解析。 |

## Intent → Event

文档例外：字段校验无新业务状态。

## Findings

- Critical: 0
- Required: 0
- Nit: `layerGitRouteHelpers.mjs` 与 `repoMatchKey.mjs` 仍双份实现 → OPT

## Security checklist

- 无密钥入仓
- 用户 URL 在边界校验 scheme
- 无新 SSRF（规范化后走既有 Git HTTP API）
