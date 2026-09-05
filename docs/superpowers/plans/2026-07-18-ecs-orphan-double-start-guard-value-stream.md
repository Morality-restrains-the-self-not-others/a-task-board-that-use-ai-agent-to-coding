# Value Stream：ECS 孤儿防护

Derived from design: `docs/superpowers/specs/2026-07-18-ecs-orphan-double-start-guard-design.md`

```
用户/@镜像再次启动
  → CLOUD_SERVER_START_AUTO / STARTED
  → [新增] 若 CSC 已有 instance → DeleteInstance(旧) + ClearAfterStop(旧)
  → RunInstances(新) → UpsertAfterStart → 看板显示新机
  → 停止/终态 → CLOUD_SERVER_STOPPED → DeleteInstance(当前)
  → ClearAfterStop(当前 instance) → 仅关该 instance history
  → [新增] workspace reconcile → InstanceName 对账回收漏网孤儿
```

测试点：二次启动无云侧双机；stop 后 history 仅关被删行；对账删除 name 匹配且非 CSC 当前实例。
