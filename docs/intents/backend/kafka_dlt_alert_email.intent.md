# Intent: Kafka 消费失败死信告警邮件

当 Kafka 消费判定永久失败或重试耗尽时，事件已投入 `{topic}-dlt`；在此基础上向运维邮箱发送报告邮件，并在 5 分钟冷却窗口内抑制重复发送，避免级联失败刷屏。

## 范围

| 项 | 说明 |
|---|---|
| 服务 | `taskEvents`（consumer runner + broker DLT） |
| 落点 | 扩展既有 `PublishDeadLetter` 成功路径；复用 `conf/events/domain-events/email.yaml` SMTP |
| 收件人 | 默认 `contact@daydaymoney.com`（可用 `DLT_ALERT_EMAIL` 覆盖） |
| 冷却 | 默认 5 分钟进程内全局冷却（可用 `DLT_ALERT_COOLDOWN` 覆盖，如 `5m`） |

## 非目标

- 不新增 HTTP API
- 不改变 DLT topic 命名与投递语义
- 不引入跨进程共享冷却状态（多 consumer 进程各自冷却，可接受）
- 不聚合多封 DLT 为 digests（首封报告即告警；窗口内后续静默）

## 验收

1. 永久失败 / 重试耗尽 → 事件写入对应 `-dlt` topic（既有行为保持）
2. DLT 发布成功后尝试发送报告邮件至配置收件人
3. 同一进程内 5 分钟窗口内第二次及以后 DLT **不**再发邮件
4. 邮件发送失败仅打日志，**不**阻断 DLT 成功路径与 Ack
5. 单元测试覆盖：冷却放行/抑制、邮件内容含事件类型与失败原因

## 业务意图 → 事件对照

**无新增业务事件**：本意图是既有 DLT 副作用上的运维告警，不发布新领域事件。

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---|---|---|---|---|---|
| — | — | — | — | — | 运维告警副作用，非业务意图投递 |

## 架构影响

- **无新 Application_Component**；复用既有 SMTP 与 DLT
- **No-ADR**: trivial ops alert on existing path, no architectural impact
- 不更新 enterprise-landscape / application-integration 拓扑版本
