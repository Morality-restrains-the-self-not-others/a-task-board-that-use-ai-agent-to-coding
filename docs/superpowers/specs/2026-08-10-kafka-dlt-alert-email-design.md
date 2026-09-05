# Design: Kafka DLT 告警邮件（5 分钟去重）

- **Date:** 2026-08-10
- **Status:** accepted (goal-mode auto)
- **Intent:** `docs/intents/backend/kafka_dlt_alert_email.intent.md`

## Context

`taskEvents` 已在 `broker.PublishDeadLetter` / consumer runner 中实现 DLT（永久失败与重试耗尽）。运维需要在死信发生时收到邮件，且级联失败时不能每条死信一封邮件。

## Decision

1. 在 **DLT 发布成功之后** 触发告警（失败发布不发邮件，避免误报）。
2. 新增 `broker.ThrottledDLTAlerter`：进程内全局冷却，默认 **5m**；收件人默认 `contact@daydaymoney.com`。
3. 通过 `broker.SetDLTAlerter` 注入；`consumer.RunWithDelivery` 启动时用既有 SMTP 配置接线。
4. 告警 **异步 + best-effort**：不阻塞 Ack；SMTP 失败只记日志。
5. SMTP 失败仍推进冷却，防止错误配置导致 SMTP 打爆。

## Alternatives Considered

| 方案 | 拒绝原因 |
|---|---|
| 每 topic 独立冷却 | 级联时仍可能多封；需求是「不再重复发送」 |
| Redis 跨进程冷却 | 过度设计；consumer 多进程重复告警可接受 |
| 独立告警微服务 | 无新服务必要；复用现有 SMTP |
| 走 EMAIL_SENT 领域事件再消费 | 增加故障面；告警应直达 SMTP |

## Consequences

- **正向**：运维可及时感知死信；5 分钟内不刷屏
- **负向**：多 consumer 进程可能各发一封（同窗口）——记入 OPT
- **缓解**：冷却与收件人可环境变量覆盖

## Architecture

无拓扑变更（No-ADR）。数据流：

```
Handler fail → PublishDeadLetter → Kafka *-dlt
                              └→ (async) ThrottledDLTAlerter → SMTP → contact@daydaymoney.com
```
