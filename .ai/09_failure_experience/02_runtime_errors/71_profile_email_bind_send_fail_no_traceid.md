# [运行时] Profile 邮箱绑定「邮件发送失败」+ 错误无 data-traceId

## 基本信息

- 案例编号：FE-20260812-EMAIL-BIND-SEND-FAIL
- 录入日期：2026-08-12
- 关联服务：`task-auth`、`docker-kafka`、`taskFE`、`task-events-email-sent-1-send-email`

## 失败现象

- 页面：`/profile/?sso_error=email_required#rg=profile.email_binding`
- 内联错误：`邮件发送失败，请稍后重试`（`p.text-sm.text-danger`）
- 错误 DOM **无** `data-traceId`，无法一键 Loki 检索

## 根因

1. **Kafka**：broker 曾 `Exited (137)`；且 `KAFKA_ADVERTISED_LISTENERS` 为 `localhost/127.0.0.1`，客户端经 `${INFRA_HOST}:9093` 拿到错误 broker 地址后 Produce 失败。
2. **SMTP 回退**：`publishEmailSent` → `sendEmailSMTP` 仅读 `EMAIL_HOST_USER` 等环境变量；taskAuth **未 sync** `conf/core/email` → `empty from address`。
3. **前端**：`UserProfileEmailBindingPanel` 在 `!response.ok` 时 `throw new Error(msg)`，`extractTraceId(error)` 拿不到响应头/体 `trace_id`。

## 解决方案

1. `conf/auth/task-auth/sync.manifest.yaml` 同步 `email.yaml`；`loadEmailConfigFromSyncedFragment` + `loadSMTPConfig` 读片段（EMAIL_* 仍优先）。
2. `dockerInfra/kafka`：`PLAINTEXT_HOST://${INFRA_HOST:-10.2.150.68}:9093`；`run.sh` 导出 `INFRA_HOST`。
3. 前端改 `safeResponseJson`，失败时设置 `inlineErrorTraceId` → `:data-traceId`。
4. 投递失败 HTTP 状态改为 `502 Bad Gateway`。

## 预防

- 新增「仅环境变量」的运维配置时，对照 conf SSOT + 本服务 sync 片段。
- 内联 API 错误禁止裸 `throw new Error` 丢 Response；统一 `safeResponseJson`。
- Kafka advertised 禁止 localhost（见网络拓扑元规则）。
