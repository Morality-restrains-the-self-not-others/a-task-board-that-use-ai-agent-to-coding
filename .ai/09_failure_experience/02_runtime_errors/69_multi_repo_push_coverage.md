# [运行时] 多仓推送：全仓遍历、部分成败明细、父子仓同逻辑

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-21
- 编号：69
- 维护者：Trae AI 团队

## 现象 / 需求

- 推送时是否多个仓库都会推？
- 部分成功、部分失败时，须展示**哪些成功、哪些失败**；父仓与嵌套子仓同一套逻辑与文案（`github_slug` + `rel_prefix` 路径）。

## 结论

| 路径 | 多仓？ | 说明 |
|------|--------|------|
| `oauth-access-push` | **是** | 扫描全部 git 根；`workdirNeedsPush` 过滤；单仓失败**不中断**其余仓 |
| Cloud prepare 换票 | **是** | 任务全部 GitHub/GitLab 仓填 token |
| auto_run 交付 | **是** | 同一 OAuth 多仓推送 |
| 裸 `git/push` | **否** | 仅主仓（见 OPT 裸推多仓） |

## 展示契约

失败响应（HTTP 400）：

- `detail`：多行文案，含「部分仓库推送未成功（成功 N，失败 M）」及每仓「成功：… / 失败：… — 原因」
- `github_oauth_multirepo.repos[]`：`push_ok`、`github_slug`、`rel_prefix`、`detail`
- 前端 `formatLayerGitPushFailureMessage` → `showRequestError`；`Modal.ui` 使用 `whitespace-pre-wrap`

## 解决方案摘要

1. 禁止「一仓成功即 200」；结束时汇总成败。
2. 单仓 token/push 失败改为 `continue`，继续推其它仓。
3. `formatOauthMultiRepoPushDetail`（容器）与前端同语义格式化。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test src/layerGitOauthPush.test.mjs
cd task2app/front_project/app && npx vitest run \
  src/composables/taskDetail/formatLayerGitPushMultiRepoDetail.test.js \
  src/composables/taskDetail/taskDetailLayerActions.test.js
```

## 关联

- `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`
- `trae-agent/onlineServiceJS/src/layerGitOauthPushDetail.mjs`
- `task2app/front_project/app/src/composables/taskDetail/formatLayerGitPushMultiRepoDetail.js`
- `67_ztree_push_terminal_prompts_disabled.md`
