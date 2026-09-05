# Pipeline artifacts: Kafka DLT 告警邮件

goal-mode 自动流水线压缩产出（Step 2/4/5/6/7）。

## Step 2 — Role / Permission

- **无新 HTTP endpoint**；无角色/权限矩阵变更
- 告警为运维旁路，不改变业务用户可见面
- SMTP 凭据仍仅来自 `conf/events/domain-events/email.yaml`（服务本目录 sync 产物）

## Step 4 — Value Stream

```
消费失败 → DLT 投递 → [新] 冷却判定 → SMTP 报告邮件 → 运维查 DLT topic
```

测试点：T1–T5（见 test.intent）

## Step 5 — NFR（L2 Standard）

| 类别 | 等级 | 说明 |
|---|---|---|
| 可靠性 | L2 | 邮件失败不阻断 Ack |
| 性能 | L2 | 异步发送；冷却减少 SMTP 压力 |
| 安全 | L2 | 邮件正文不含密码/token；可含 event_type/error/key |
| 可观测 | L2 | 发送/抑制/失败均打结构化日志 |

## Step 6 — DDD

- **无新聚合/领域事件**
- 技术适配：`MailSender` 端口在 broker 包；SMTP 适配器在 notifications
- 冷却为基础设施策略，非领域规则

## Step 7 — Implementation Plan

- [x] Red: `dlt_alert_test.go`（T1–T5 + Resolve）
- [x] Green: `dlt_alert.go` + `PublishDeadLetter` 钩子 + consumer 接线
- [x] 跑 `go test ./broker/ ./consumer/ -count=1`
- [x] 格式验证 gofmt / go vet
- [x] 登记 precise restart: taskEvents
- [x] OPT 落盘（跨进程冷却等）
