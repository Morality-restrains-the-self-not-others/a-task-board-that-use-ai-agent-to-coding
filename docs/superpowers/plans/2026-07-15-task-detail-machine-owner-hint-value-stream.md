# 价值流：task-detail-machine-owner-hint

## 主路径

```
用户打开任务详情
  → ServerConfig 拉取 server-runtime-status
  → {container_running?}
      否 → 镜像区不展示所属任务
      是 → 展示 machine_owner_task_ids（本任务标注 / 他任务可跳转）
  → 用户理解容器所在机器节点归属
```

## 测试点

| ID | 步骤 | 期望 |
|----|------|------|
| VS-1 | 无容器 | 无「机器节点所属任务」 |
| VS-2 | 本任务独占机且容器运行 | 展示本任务 ID +「本任务」 |
| VS-3 | 同实例两任务绑定且本任务有容器 | 列出两个 task_id |
| VS-4 | 点击他任务链接 | href 指向该任务 detail |
