# 价值流：任务终态硬释放 + 镜像容器迁移

- 日期：2026-07-15
- 设计：`2026-07-15-terminal-hard-release-container-migrate-design.md`
- Stream ID：`terminal-hard-release-container-migrate`

## 端到端流

```text
[成员改进度→终态] → [taskTaskService 发 TASK_STATUS_CHANGED]
  → [taskEvents 识别终态]
  → [Cloud: list instance-bindings]
  → {有兄弟 busy?} → [migrate 到所属任务机器/新建] → [更新兄弟 CSC]
  → [停终态任务容器]
  → [CLOUD_SERVER_STOPPED / clear]
  → [mark terminal_released]
  → [idle reuse 永不选中该绑定]
```

## 最小可行增量

| MVP | 内容 |
|-----|------|
| M1 | reuse 解绑 source + terminal_released 排除 |
| M2 | bindings API + Handler 先查兄弟 |
| M3 | migrate-container-off-instance + 失败重试 |
| M4 | 既有 stop 路径 + mark |

## 测试点（对齐 test-intent T1–T8）

- TP1 独占终态硬释放
- TP2 兄弟 busy 先迁后释
- TP3 migrate 失败不 stop
- TP4 terminal_released 不可 reuse
- TP5 非终态无副作用
