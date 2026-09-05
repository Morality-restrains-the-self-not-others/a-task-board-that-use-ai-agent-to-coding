# 厂商证照腾讯云 COS 预签名直传 — 设计文档

- **日期**: 2026-08-14
- **作者**: cursor
- **迭代**: vendor-docs-cos-presign-v79
- **前置**: `2026-08-10-vendor-application-kyc-docs-design.md`（本地 `vendorDocsDir` 已交付）
- **OPT**: OPT-20260810-016（INFRA 阻塞解除：桶 `ai-provider-1259712831` 已提供）
- **状态**: accepted（2026-08-14 总体设计审批通过）
- **ADR**: 落地时新增 ADR-0006（第三方对象存储选型：腾讯云 COS）

## 1. 问题背景

厂商申请身份证 / 营业执照现由 `taskAiProvider` 写入本机 `vendorDocsDir`。多实例无法共享本地盘，证照为高敏 PII。对象存储已定为腾讯云 COS，桶地址：

`https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com`

产品选择：**浏览器预签名直传**（不经应用进程中转文件体），密钥填本机 `config.local.yaml`，**管理员后台可改对象路径规则并写回 conf**。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 申请人浏览器 PUT 到 COS，应用只签发短时预签名 URL | FE 不再走 multipart 落盘；单测 + 联调 |
| S2 | 桶私有；对象 SSE-COS AES256；密钥不入库、不进前端 | 控制台清单 + `config.local.yaml` gitignore |
| S3 | 提交申请时 `HeadObject` 校验 file_key 属当前用户且对象存在 | 缺对象 / 越权 → 400 |
| S4 | 运营仍经 staff API 看证照（服务端 GetObject 流式），不暴露永久公网 URL | AdminPortal 下载 200 |
| S5 | 路径规则可在管理后台修改，写回本服务 conf 片段并热加载 | PATCH 后新上传立即用新规则 |
| S6 | 出站 COS Client 直连，禁止环境 Proxy | `Transport.Proxy = nil` |
| S7 | 存量本地 `file_key` 仍可被 staff 读取 | 解析器：COS miss → 回退本地盘 |

## 3. 方案决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 浏览器 **Presigned PUT**（非 STS 临时密钥、非服务端代传） | 用户选定；≤5MB 单次 PUT 足够；密钥永不进浏览器 |
| D2 | 预签名绑定 **精确 object key + Content-Type + Content-Length** | 防改 key / 改类型 / 超大文件 |
| D3 | 新增 `POST .../upload-url/` + `POST .../upload-complete/`；保留旧 multipart 仅作 `backend=local` 测试回退 | 生产走 COS；单测不依赖真桶 |
| D4 | 密钥只写 `conf/ai/ai-provider/config.local.yaml` | 已 gitignore；`ReadAppConfig` 深合并 |
| D5 | 路径规则写 `conf/ai/ai-provider/vendor-docs-path.yaml`（无密钥），后台 PATCH 原子写回 | 满足「写回 conf」且不碰 `config.yaml` / 密钥文件 |
| D6 | 运营下载仍走 Go 反代 GetObject | 避免预签名 GET 泄漏到浏览器历史 / 日志 |
| D7 | 接口落现有 Go `taskAiProvider` | 证照表 owner 已是 ai-provider；无新 Python 接口 |
| D8 | 架构 bump v79 | 新增外部 Technology（COS）与浏览器→COS 数据流 |

## 4. 腾讯云控制台必须设置（本问核心）

桶名解析：`ai-provider-1259712831` = 自定义名 `ai-provider` + APPID `1259712831`；地域 **上海** `ap-shanghai`。

### 4.1 访问与加密（强制）

| 项 | 设置 | 禁止 |
|----|------|------|
| 访问权限 | **私有读写** | 公有读、公有读写 |
| 阻止公共访问 | **开启** | 关闭后误配 ACL |
| 默认加密 | **SSE-COS AES256** | 不加密 |
| 版本控制 | 建议开启（误删可回滚） | — |
| 静态网站 / CDN 源站 | **关闭** | 把证照当静态站 |

### 4.2 CORS（浏览器 PUT 必需）

在桶「安全管理 → 跨域访问 CORS」增加规则（Origin 用 `conf/base.yaml` 的公网域，勿写 LAN IP）：

| 字段 | 值 |
|------|-----|
| AllowedOrigin | `https://www.daydaymoney.com`、`https://daydaymoney.com`、`https://provider.daydaymoney.com`（`isDev` 另加本机 FE origin，写 `config.local.yaml` 的 `vendorDocs.cos.corsAllowedOrigins`） |
| AllowedMethod | `PUT`, `HEAD`, `OPTIONS` |
| AllowedHeader | `*`（至少 `Content-Type`, `Content-Length`, `x-cos-server-side-encryption`, `x-cos-acl`） |
| ExposeHeader | `ETag`, `x-cos-request-id` |
| MaxAgeSeconds | `600` |

不开放 `GET` CORS：运营看图走本站 API，浏览器不应直读 COS。

### 4.3 CAM 子账号（已有密钥时核对）

策略只授本桶前缀，示例：

```json
{
  "version": "2.0",
  "statement": [{
    "effect": "allow",
    "action": [
      "name/cos:PutObject",
      "name/cos:GetObject",
      "name/cos:HeadObject"
    ],
    "resource": [
      "qcs::cos:ap-shanghai:uid/1259712831:ai-provider-1259712831/vendor-docs/*"
    ]
  }]
}
```

禁止：`DeleteBucket`、全桶 `GetObject` 无前缀、把 Secret 提交进 git。

### 4.4 本机密钥落点

```yaml
# conf/ai/ai-provider/config.local.yaml  （勿提交）
vendorDocs:
  backend: cos
  cos:
    secretId: "AKID..."
    secretKey: "..."
```

`config.yaml` 只放非密钥骨架（bucket / region / 默认 pathRule）。

## 5. 配置 SSOT

`conf/ai/ai-provider/config.yaml` 新增（密钥空串）：

```yaml
vendorDocs:
  backend: local   # 生产/联调改为 cos；单测保持 local
  localDir: ""     # 空则沿用代码默认 taskAiProvider/data/vendor-docs
  cos:
    bucket: ai-provider-1259712831
    region: ap-shanghai
    keyPrefix: vendor-docs
    pathRule: "{keyPrefix}/{userId}/{kind}_{id}{ext}"
    sse: AES256
    presignTTLSeconds: 300
    secretId: ""
    secretKey: ""
    corsAllowedOrigins: []  # 控制台 CORS 的文档对照；运行时不靠此开桶
```

`conf/ai/ai-provider/vendor-docs-path.yaml`（后台可写片段）：

```yaml
keyPrefix: vendor-docs
pathRule: "{keyPrefix}/{userId}/{kind}_{id}{ext}"
```

加载顺序：`config.yaml` → `config.local.yaml` → `vendor-docs-path.yaml`（后写覆盖 pathRule/keyPrefix）。`LoadConfig` 用 `confload.ReadAppConfig` + `ReadAppFragment`，不再只手解析一张 yaml。

**pathRule 白名单占位符**：`{keyPrefix}` `{userId}` `{kind}` `{id}` `{ext}` `{yyyy}` `{mm}` `{dd}`。拒绝 `..`、绝对路径、空规则、未声明占位符。

后台写回：原子写临时文件 + rename；只改上述两键；写后进程内热加载。不写 `config.yaml`，避免把密钥或人工注释冲掉。

## 6. API

均落 **Go `taskAiProvider`**。无新增 Python 接口。

### 6.1 `POST /api/ai-provider/vendor-application/upload-url/`

鉴权同现上传（`X-User-Id` + 真实邮箱）。

```json
{ "kind": "id_card", "filename": "id.png", "content_type": "image/png", "size": 12345 }
```

校验 kind / 扩展名 / size（1…5MiB）/ content_type 与扩展名一致。按 pathRule 生成 `file_key`，`GetPresignedURL(PUT, 300s)`，签名头含 `Content-Type`、`Content-Length`、`x-cos-server-side-encryption=AES256`。

```json
{
  "file_key": "vendor-docs/42/id_card_123.png",
  "upload_url": "https://ai-provider-1259712831.cos.ap-shanghai.myqcloud.com/...",
  "method": "PUT",
  "headers": {
    "Content-Type": "image/png",
    "Content-Length": "12345",
    "x-cos-server-side-encryption": "AES256"
  },
  "expires_in": 300
}
```

日志只记 `file_key` / kind / user_id，**禁止**打印完整预签名 URL。

### 6.2 `POST /api/ai-provider/vendor-application/upload-complete/`

```json
{ "kind": "id_card", "file_key": "vendor-docs/42/id_card_123.png" }
```

`HeadObject`：存在、大小匹配、Content-Type 匹配、key 属当前 userId。通过后 200。

### 6.3 既有 `POST .../vendor-application/`

`ResolveVendorDocument` 改为：COS `HeadObject`；若 COS 无对象且本地盘有旧 key，则兼容通过。

### 6.4 既有 `GET .../admin-vendors/{id}/documents/{kind}/`

staff：COS `GetObject` 流式；miss 则读本地。`Content-Disposition: inline`。

### 6.5 `GET|PATCH /api/ai-provider/admin-vendor-docs-storage/`

staff / 平台运营（同 marketplace-settings）。GET 返回当前 `backend/bucket/region/keyPrefix/pathRule`（无密钥）。PATCH `{keyPrefix, pathRule}` → 校验 → 写 `vendor-docs-path.yaml` → 热加载。

### 6.6 旧 `POST .../upload/`（multipart）

仅 `vendorDocs.backend=local` 可用；`backend=cos` 返回 410，引导走 upload-url。

## 7. 前端

- `VendorApplicationForm.vue`：选文件 → upload-url → `fetch(upload_url, {method:'PUT', headers, body:file})` → upload-complete → 保存 file_key。
- `AdminPortal.vue` 已超 500 行：**必须**抽 `AdminVendorDocsStorage.vue` 承载路径规则表单，禁止继续堆进 Portal。
- 报错节点继续带 `data-traceId`。

## 8. 领域概念（供 /6-ddd）

| 类型 | 概念 |
|------|------|
| Bounded Context | AI Provider Marketplace / Vendor Onboarding |
| Entity | Vendor、VendorDocument（file_key 标识） |
| Aggregate | Vendor（证照 key 为值对象） |
| 外部系统 | Tencent COS（Technology Service） |

## 9. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 申请人完成证照直传并经 Head 确认 | VendorDocumentUploaded | upload-complete handler | 无（审计日志）；预留运营统计 | — |
| 提交/重提厂商申请 | — | — | — | 沿用既有例外：状态闭环在 ai-provider，无 MQ 订阅方 |
| 运营修改证照路径规则 | VendorDocPathRuleUpdated | admin-vendor-docs-storage PATCH | 无（审计）；热加载本进程 | — |

`VendorDocumentUploaded` / `VendorDocPathRuleUpdated` payload **不含**文件字节、证件号、完整预签名 URL；只含 user_id / vendor_id / kind / file_key / pathRule。

## 10. 价值流影响

`conf/value-stream.yaml` 无独立「厂商申请证照」stream。影响：

- 既有镜像市场 / 厂商入驻步骤：上传从「本机 multipart」改为「预签名 PUT + complete」。
- 字段：`ai-provider.ai_provider_vendor.id_card_file_key` / `business_license_file_key` 语义从「相对本地路径」变为「COS object key（或历史本地 key）」。
- 测试：`vendor_docs_test.go`、`vendor_status_db_test.go`、`VendorApplicationForm.test.js`、意图 T1 改为 upload-url + complete；新增 T11 路径规则校验、T12 COS miss 回退本地。
- `/4-value-stream` 再切片；本步不改 YAML。

## 11. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：108 nodes / 937 edges / 17 files；语言 javascript, typescript, python, bash；last_updated 2026-08-12；branch main |
| 关键发现 | 图未索引 Go `taskAiProvider`；证照写盘在 `infrastructure/vendor_docs.go`，上传入口 `handleVendorApplicationUpload`，FE `VendorApplicationForm.vue` |
| 决策影响 | 爆炸半径限 ai-provider + taskFE 申请表 + AdminPortal；不改 taskAuth SMS gate |
| skip 理由 | Go 符号不在当前图内；按源码检索补全 |

## 12. Trace 日志

本次无 `data-traceId` / 运行时报错。`skipped_no_traceid`。

## 13. 安全（STRIDE 摘要）

| 威胁 | 缓解 |
|------|------|
| Spoofing | 预签名仅登录申请人可领；key 含 userId |
| Tampering | 签名绑定 key / Content-Type / Content-Length |
| Info disclosure | 桶私有；日志无影像/无签名 URL；运营走反代 |
| DoS | 5MiB + TTL 300s |
| Elevation | pathRule 白名单；staff 才能改规则 / 下载 |

测试禁止真实证件。联调用虚构图。

## 14. 🐍 Python 新增接口

不触发。全部 Go。

## 15. 🏛️ 架构变更影响

- **迭代版本**: v79 🎯 target
- **迭代名称**: vendor-docs-cos-presign-v79
- **作者**: cursor
- **设计日期**: 2026-08-14 13:33
- **新增文件**（每个视图四类伴生）：
  - 🆕 `docs/architecture/v79-enterprise-landscape-20260814-1333-cursor.puml`
  - 🆕 `docs/architecture/v79-application-integration-20260814-1333-cursor.puml`
  - 🆕 `docs/architecture/v79-enterprise-landscape-20260814-1333-cursor.diff.archimate`（增量：v13→v79 COS 数据流 + Plateau/Gap/WP）
  - 🆕 `docs/architecture/v79-application-integration-20260814-1333-cursor.diff.archimate`（增量：v78→v79）
  - 🆕 `docs/architecture/v79-enterprise-landscape-20260814-1333-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v79-application-integration-20260814-1333-cursor.full.archimate`（全量拓扑）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v75-application-integration-20260811-2135-cursor.puml` (current)
  - `docs/architecture/v13-enterprise-landscape-20260806-0008-claude.puml` (current)
  - v76–v78 仍为积压 target
- **变更明细**: 🟢 COS 桶 / path 片段 / 🟡 aiProvider + taskFE + file_key

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v78/v13 → Gap（本地盘不可共享）→ WP → Plateau v79；变更数据流 FE→COS PUT、aiProvider→COS sign/head/get |
| **`.full.archimate`** | 变迁后拓扑（网关 / FE / aiProvider / COS / file_key / path 片段） |

## 17. 权限影响分析

见 `2026-08-14-vendor-docs-cos-presign-permission-analysis.md`。绿灯：复用 requireVendorApplicant / requireStaff；file_key 属主强制；无新租户 region。

## 16. 实现落点（批准后）

- `conf/ai/ai-provider/config.yaml` 骨架
- `taskAiProvider/infrastructure/vendor_docs.go` → `VendorDocStore` 接口（local / cos）
- `sdk/tencent/cos-go-sdk-v5` + `tracelog.DirectClient` 作为 Transport
- `taskAiProvider/go.mod` replace 指向仓内 SDK
- ADR-0006
- 意图文档同步本文件第 9 节
