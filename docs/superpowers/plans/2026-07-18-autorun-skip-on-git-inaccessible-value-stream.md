# 价值流：Git 不可用时自动运行不启服

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-autorun-skip-on-git-inaccessible-design.md`

## 端到端增量

```
用户开启 auto_run 并保存任务
  → taskTaskService 硬门禁（云/镜像/模版）
  → 探测关联仓 Git/子 Git 可达性
  → [阻断] 不可达：持久化任务 + 跳过 start-vm + 返回 skip 字段
  → [放行] 可达：持久化 + schedule start-vm
  → 任务详情展示子仓错误（若有）时，服务器未自动启动
```

## 最小价值切片

| ID | 切片 | 验收 |
|----|------|------|
| VS-1 | 无授权时 create auto_run 不打 cloud | mock nested error → cloudCalls=0 |
| VS-2 | 探测 OK 时仍启动 | nested error="" → cloudCalls≥1 |
| VS-3 | auto_run 仍为 true | 响应/DB auto_run=true 且 skip 字段存在 |

## 测试点映射

| 测试点 | 用例 |
|--------|------|
| T-skip-auth | `TestCreateTaskAutoRunSkipsStartWhenNestedGitNeedsAuth` |
| T-ok | 现有 `TestCreateTaskAutoRunTriggersStart`（mock 增加 nested OK） |
| T-update | `TestUpdateTaskAutoRunSkipsStartWhenGitInaccessible` |
