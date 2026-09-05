# Test Intent: Kafka 消费失败死信告警邮件

对应功能意图：`kafka_dlt_alert_email.intent.md`

## 测试分层

| 层级 | 内容 |
|---|---|
| 单元 | `ThrottledDLTAlerter`：首次 Notify 调用 Sender；冷却内第二次不调用；冷却过后再次调用 |
| 单元 | 邮件正文包含 `original_event_type` / `failure_reason` / `error` / `dlt topic` |
| 单元 | `SetDLTAlerter` + `maybeAlertDLT`：已配置时触发；未配置时 no-op |
| 回归 | 既有 DLT 投递语义不变（PublishDeadLetter 仍 best-effort，失败不返回） |

## 用例清单

1. **T1 首次告警发送** — 空冷却状态 → Notify → Sender.Send 恰好 1 次，收件人为配置地址
2. **T2 窗口内抑制** — 首次发送后立即再 Notify → Sender 仍为 1 次
3. **T3 窗口外再发** — 模拟时钟前进 ≥ cooldown → 第三次 Notify → Sender 为 2 次
4. **T4 Sender 失败不恐慌** — Sender 返回 error → Notify 不 panic，冷却仍推进（避免失败风暴反复打 SMTP）
5. **T5 未配置 alerter** — maybeAlertDLT 无副作用

## 不测

- 真实 QQ SMTP 联通（依赖密钥与外网；部署后人工抽检）
- 跨进程冷却一致性
