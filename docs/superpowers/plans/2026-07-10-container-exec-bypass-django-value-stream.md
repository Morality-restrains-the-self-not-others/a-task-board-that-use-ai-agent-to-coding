# 价值流：容器执行热路径零 Django

- **日期**: 2026-07-10

## 端到端价值流（目标）

```text
[租户用户] 在任务详情发送指令 / 查看 job 日志
    → Vue 调用 cloud/compute/container-* 
    → APISIX → taskContainerGateway
    → taskAuth 鉴权（身份+租户成员）
    → taskCloudService 解析 base_url+token
    → onlineServiceJS 执行 / 返回状态
    →（SSE/Kafka）前端展示
```

## 价值增量（MVP = Phase A）

| 增量 | 用户价值 | 验收 |
|------|----------|------|
| V1 热路径不经 Django | saas-backend 日志干净；Django 宕机不阻断已注册容器执行 | T1/T2/T6 |
| V2 可观测 stage 更名 | 运维可区分 auth/cloud/upstream | tcg 日志无 django_* |

## 测试点映射

见 `docs/intents/backend/container_exec_bypass_django.test-intent.md` T1–T8。
