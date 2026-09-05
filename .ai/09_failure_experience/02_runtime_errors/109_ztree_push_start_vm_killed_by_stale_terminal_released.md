# [运行时] 评论容器重开：旧 terminal_released 把「进行中」任务新实例当 cancelled 释放

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：109
- 维护者：Trae AI 团队

## 现象

- 修完 [108](./108_ztree_push_gitlab_borrow_other_ce_token.md) 后换票成功，但容器已 Released，「提交并创建PR」502。
- 对进行中任务 `task_878583341551480832` POST `start-vm` 成功，`instance_id`/`public_ip` 回填。
- 约 25s 后 `event=request_machine_release reason=inbound_task_terminal terminal_kind=cancelled`。
- 新实例 `:8765` 永不就绪（UserData `boot-progress` HTTP 410）。

## 根因

1. `TerminalReleasedFlag` 按 `company+workspace+task` 查任意一行；任务级/其它评论 `terminal_released=1` 污染当前评论。
2. `refuseInboundIfTaskTerminal` 在任务并非终态时，把 leftover flag 默认成 `cancelled` 并 `destroy` 新实例。
3. `persistStartVmInstanceBinding` 绑定新 `instance_id` 时未清 `terminal_released`，且曾保留 `last_runtime_status=Released`。

## 解决方案

1. Flag 按 CSC `id` 或 `task+comment` 读取。
2. 进行中任务 + 已绑定活实例：忽略 leftover flag，禁止入站杀机。
3. 新实例 persist / 公网 IP 回填时 `terminal_released=0`，Released 状态改为 Starting。

## 验证

```bash
cd taskCloudService && go test ./src -count=1 -run 'PersistStartVmInstanceBindingClearsTerminalReleased|TerminalReleasedFlagIsCommentScoped|RefuseInboundOpenTaskIgnoresLeftoverFlagOnLiveInstance'
```

## 关联

- `.ai/09_failure_experience/02_runtime_errors/108_ztree_push_gitlab_borrow_other_ce_token.md`
