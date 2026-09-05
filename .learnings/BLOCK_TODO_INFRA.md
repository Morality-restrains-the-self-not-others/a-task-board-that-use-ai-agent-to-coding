# Blocked TODOs — INFRA

> 本文件存放因 **基础设施或外部资产（腾讯云 COS、正式品牌物料等新组件）** 阻塞而从开放清单分流的 OPT 条目。
> 阻塞解除后移回 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md) 执行，或完成后迁入 [OPTIMIZATION_TODOS_COMPLETED.md](./OPTIMIZATION_TODOS_COMPLETED.md)。
> 分流规则见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「阻塞项分流」。
> **解除方式**：COS 桶与密钥已配置、适配器可联调后移回开放清单；本会话已完成则直接 `completed`。

- **Category**: `INFRA`
- **Count**: 3

---

### OPT-20260810-016 — 厂商证照改腾讯云 COS 并加密静态落盘

- **Status**: pending
- **Blocked-By**: INFRA
- **Created**: 2026-08-10
- **Decision**: 2026-08-13 — 存储选型为腾讯云 COS（参照 `sdk/tencent/cos-go-sdk-v5`），不用 MinIO/S3。管理员后台可配置对象存储路径规则。先在 `conf/ai/ai-provider/` 落配置骨架，密钥等敏感项后续再填（`config.local.yaml`，勿提交）。
- **Context**: 证照现落在 taskAiProvider 本地 `vendorDocsDir`（`SaveVendorDocument` 写盘）。对象存储选型已定为腾讯云 COS；仍缺 COS 桶/密钥，适配器与后台路径规则尚未落地。
- **Action**: (1) 在 `conf/ai/ai-provider/config.yaml` 增加 COS 骨架（bucket、region、keyPrefix/pathRule；密钥键名占位，值留空）；(2) 管理员后台增加存储路径规则配置并写回 conf；(3) 用 `sdk/tencent/cos-go-sdk-v5` 将 `SaveVendorDocument` 迁 COS 适配器，静态加密落盘，出站 Client 直连禁止环境 Proxy；(4) 存量本地 `file_key` 兼容或一次性迁移；(5) 密钥到位后填 `config.local.yaml` 联调。落地时补 ADR（第三方对象存储选型）。
- **Why**: 多实例不可共享本地盘；证照为高敏 PII。
- **How to apply**: `conf/ai/ai-provider/config.yaml`；`taskAiProvider/infrastructure/vendor_docs.go`；`sdk/tencent/cos-go-sdk-v5`；`.ai/01_project_constraints/16_payment_kyc_compliance.md`；`.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`；`.ai/01_project_constraints/23_app_startup_no_env_proxy.md`。

### OPT-20260828-020 — 用具备 PutBucketCORS 的密钥给 ai-provider 桶写入 CORS

- **Status**: pending
- **Blocked-By**: INFRA
- **Created**: 2026-08-28
- **Context**: 编辑镜像组图标浏览器 OPTIONS 到 `ai-provider-1259712831.cos.ap-shanghai.myqcloud.com` 403。`GetCORS` 为 `NoSuchCORSConfiguration`；现网对象密钥 `PutCORS` 为 `AccessDenied`。本会话已改为同源 `local-put` 服务端 `Object.Put`，图标可上传，但浏览器直传仍不可用。
- **Action**: (1) 在腾讯云 CAM 给桶管理密钥 `cos:PutBucketCORS`/`cos:GetBucketCORS`；(2) 用该密钥 `PutCORS`，AllowedOrigin 含 `${scheme}://${subdomains.provider}`，AllowedMethod PUT/GET/HEAD/POST，AllowedHeader 为 Content-Type、Content-Length、x-cos-server-side-encryption、Origin（禁止 Authorization）；(3) `GetCORS` 非 404 后 `LIVE_COS_CORS=1 go test ./infrastructure -run TestLiveEnsureVendorDocsCORS` 通过。
- **Why**: 无桶 CORS 时任何浏览器直传 COS 都会 OPTIONS 403；当前对象密钥写不了 CORS，进程 EnsureCORS 无法自愈。
- **How to apply**: 桶 `ai-provider-1259712831`；`taskAiProvider/infrastructure/vendor_docs_cors.go`；失败经验 `.ai/09_failure_experience/02_runtime_errors/122_provider_image_group_icon_cos_options_403.md`。

### OPT-20260901-017 — 排查 clone-run 主机 systemd-resolved 对 smtp.qq.com 的查询失败

- **Status**: pending
- **Blocked-By**: INFRA
- **Created**: 2026-09-01
- **Context**: 密码重置邮件死信窗口（2026-09-01 17:07–17:10 CST）Loki `{job="task-events-email-sent-1-send-email"}` 出现 `dial tcp: lookup smtp.qq.com: i/o timeout` 与 `lookup smtp.qq.com on 127.0.0.53:53: server misbehaving`。同一进程前后又能连上 QQ 并返回 535，说明不是 SMTP 配置缺失。
- **Action**: (1) 在 DEPLOY_ROOT 主机查 `resolvectl query smtp.qq.com` 与 `journalctl -u systemd-resolved` 同时段；(2) 确认业务进程未继承环境 Proxy（元规则 23）；(3) 若 127.0.0.53 不稳定，给发信 worker 配稳定上游 DNS 或修复 resolved，禁止把业务 listen 改回 127.0.0.1。
- **Why**: DNS 抖动会把可重试错误叠在 535 重试上，邮件全部进 DLT。
- **How to apply**: 主机 `resolvectl statistics`；Loki `{job="task-events-email-sent-1-send-email"} |= "smtp.qq.com"` 连续 1h 无 `server misbehaving`。
- **Related**: OPT-20260901-015
