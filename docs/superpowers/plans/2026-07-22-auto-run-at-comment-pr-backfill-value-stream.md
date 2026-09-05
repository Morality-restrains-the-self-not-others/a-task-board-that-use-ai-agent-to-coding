# auto_run @ 评论 PR 回填 — 价值流

## 端到端流

```text
用户创建/更新任务 auto_run=true
  → TTS 门禁 + ensureAutoRunAtComment（人类@ + Agent pending）
  → start-vm-auto
  → 容器 bootstrap → task-detail 注入 at_mention_run(source=auto_run)
  → kickoff: auto_run 首指令 job（挂载 agent_comment_id）
  → job completed → delivery(commit/push/PR)
  → complete Agent：assistant_response 含 PR 链接
  → 任务详情 Feed 可见 @ 评论与回复中的 PR
```

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | ensureAutoRunAtComment | 插入 mentions + Agent；无 Kafka mention 事件 |
| T2 | 幂等 | 已有 active auto_run Agent → 不重复插评论 |
| T3 | kickoff source=auto_run | kind=auto_run 且 job 带 mount ids |
| T4 | kickoff 用户 at_mention | 仍优先 at_mention |
| T5 | delivery + PR | complete 调用含 html_url |
| T6 | delivery 无 PR（clean skip） | complete 文案说明无 PR / 已跳过 |
