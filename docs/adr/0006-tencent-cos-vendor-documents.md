# ADR-0006: 厂商证照对象存储选用腾讯云 COS

- **Status:** accepted
- **Date:** 2026-08-14
- **Author:** cursor
- **Deciders:** /1-brainstorming 总体设计审批（用户选定预签名直传 + 后台写回路径规则）

---

## Context

厂商申请身份证 / 营业执照原先由 `taskAiProvider` 写入本机 `vendorDocsDir`。多实例无法共享本地盘；证照为高敏 PII，不宜散落在应用主机磁盘。OPT-20260810-016 已排除 MinIO/S3 兼容层。现网已提供桶 `ai-provider-1259712831`（上海 `ap-shanghai`）。

## Decision

We will store vendor KYC documents in Tencent Cloud COS and upload via **browser presigned PUT**:

1. 桶 `ai-provider-1259712831`、地域 `ap-shanghai`；私有读写 + 阻止公共访问 + SSE-COS AES256。
2. 应用用仓内 `sdk/tencent/cos-go-sdk-v5` 签发短时 PUT URL（绑定 object key、Content-Type、Content-Length）；出站 Client 直连（`Proxy = nil`）。
3. SecretId/SecretKey 只写 `conf/ai/ai-provider/config.local.yaml`（gitignore）；`config.yaml` 仅非密钥骨架。
4. 对象路径规则默认 `{keyPrefix}/{userId}/{kind}_{id}{ext}`；运营可改并写回 `vendor-docs-path.yaml` 片段后热加载。
5. 运营审阅经 taskAiProvider `GetObject` 反代，不向浏览器签发长期 GET。
6. `file_key` 存 COS object key；存量本地 key 在 COS miss 时回退读盘。

## Alternatives Considered

### Alternative 1: 继续本机磁盘

- **Pros:** 已实现，无外部依赖。
- **Cons:** 多实例不一致；主机磁盘存 PII。
- **Why rejected:** 不满足多实例与合规落盘。

### Alternative 2: MinIO / 通用 S3

- **Pros:** 本地可自建。
- **Cons:** 与已选腾讯云桶重复；多一套运维。
- **Why rejected:** OPT-20260810-016 已定 COS。

### Alternative 3: 服务端代传 multipart

- **Pros:** 无需 CORS；前端几乎不改。
- **Cons:** 证照字节经应用进程；带宽与超时压在 :8010。
- **Why rejected:** 用户选定浏览器预签名直传。

### Alternative 4: STS 临时密钥 + COS JS SDK

- **Pros:** 多分片更灵活。
- **Cons:** 浏览器持有临时密钥，策略面更大；单文件 ≤5MB 不需要分片。
- **Why rejected:** 预签名 PUT 权限更窄。

## Consequences

### Positive

- 多实例共享同一私有桶；静态加密由 COS 承担。
- 密钥不进前端、不进 git。

### Negative / Trade-offs

- 必须正确配置桶 CORS，否则浏览器 PUT 失败。
- 预签名 URL 若被日志打印会泄漏上传权（TTL 内）。
- 后台写 conf 片段要求进程对 `conf/ai/ai-provider/` 可写。

### Mitigations

- CORS Origin 仅公网 FE / provider 域；不开放 GET CORS。
- 日志只记 file_key / kind / user_id。
- pathRule 白名单占位符，拒绝 `..`。
- 单测用 fake COS + `backend=local`。

## References

- `docs/superpowers/specs/2026-08-14-vendor-docs-cos-presign-design.md`
- `docs/intents/vendor-docs-cos-presign.intent.md`
- OPT-20260810-016
