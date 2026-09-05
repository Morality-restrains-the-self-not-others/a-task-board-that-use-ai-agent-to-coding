# 权限分析：启动日志默认折叠 + zTree「推送并创建PR」

- **日期**: 2026-07-12
- **基于**: `2026-07-12-relay-logs-collapse-and-ztree-push-create-pr-design.md`
- **状态**: approved (auto)

## 变更点权限矩阵

| 改动点 | 角色 | 权限边界 | 结论 |
|--------|------|----------|------|
| 启动日志折叠默认态 | 任务可读用户 | 仅 UI 默认态；无新 API | ✅ 无权限变更 |
| zTree「推送并创建PR」 | 任务可写 + 容器可达 | 沿用既有 `container-layer-git-push` 鉴权与 Git 身份校验 | ✅ 无新端点 |
| `wait_for_pr` | 同上 | 仅改变是否同步等待 PR；PR 仍走既有 GitHub App 批准/选账号门控 | ✅ 门控不变 |
| 打开 PR URL | 浏览器客户端 | 打开 GitHub 外链；无服务端提权 | ✅ |

## 风险

- 同步等待延长 HTTP 占用：限制超时（~50s），超时返回已推送成功 + PR queued/compare_url，不回滚推送。
- 凭据未批准等 skipped 场景：保持 alert，不强制跳转。

## 结论

无需新增 RBAC 规则；沿用 push + GitHub PR follow-up 既有权限链。
