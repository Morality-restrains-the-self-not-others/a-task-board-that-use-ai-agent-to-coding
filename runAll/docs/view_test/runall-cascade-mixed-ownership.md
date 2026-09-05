# runAll 链式关闭混合所有权 — View Test

## 前置

- runAll 运行于 `http://localhost:9999/`
- platform 链：git-oauth → saas-backend → taskFE

## 场景：bootstrap + UI session 混用

1. 确认 `taskFE`、`saas-backend` 由 bootstrap 启动（`.runall/ownership.json` 中 OwnerSessionID = `runall-bootstrap`）
2. 在 UI 对 `git-oauth` 点击「启动」（所有权变为浏览器 session）
3. 对 `git-oauth` 点击「关闭」

## 期望

- 无 `requires explicit takeover` 弹窗
- `taskFE`、`saas-backend`、`git-oauth` 均为 `stopped`
- 关闭顺序：下游先于上游

## 自动化

`cd runAll && go test ./src/... -run TestRunner_StopServiceCascade_DelegatesOwnershipPerStep -v`

## 回归

单点 `cascade=false` 非 owner 停止仍被拒绝：`TestStopService_RejectsNonOwnerSession`
