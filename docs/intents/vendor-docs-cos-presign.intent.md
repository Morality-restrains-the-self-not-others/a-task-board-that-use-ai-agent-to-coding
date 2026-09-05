# 功能意图：厂商证照经 COS 预签名直传

## 背景与目标

厂商申请身份证 / 营业执照从本机 `vendorDocsDir` 改为腾讯云 COS 桶 `ai-provider-1259712831`（上海）。浏览器用短时预签名 PUT 直传；密钥只在 `config.local.yaml`；运营可改对象路径规则并写回 `conf/ai/ai-provider/vendor-docs-path.yaml`。

## 范围与边界

- **范围内**：taskAiProvider 预签名 / Head / Get；taskFE 申请表直传；AdminPortal 路径规则；COS 控制台私有+CORS+SSE。
- **范围外**：服务端代传生产路径；STS 联邦密钥进浏览器；公有读桶；新 Python 接口。

## 约束与风险

- 证照为 PII：桶私有、SSE-COS、日志不落影像与完整预签名 URL。
- 出站 COS Client 禁止环境 Proxy。
- 密钥禁止提交 git。
- AdminPortal 已超 500 行，路径规则 UI 必须抽独立组件。

## 参与者

- 申请人、运营（ai-provider staff）
- taskAiProvider、taskFE、Tencent COS
- taskAuth（SMS gate，不变）

## 成功路径

1. 申请人选文件 → `upload-url` → 浏览器 PUT COS → `upload-complete`（HeadObject）→ 得 file_key
2. 短信验证后提交申请（既有）
3. 运营经 staff API 流式看证照（GetObject，本地 key 可回退）
4. 运营改 pathRule → 写回 conf 片段 → 热加载 → 新上传用新规则

## 失败路径

- 非法类型/超 5MB / 预签名过期 / Head 无对象 / file_key 越权 → 4xx
- `backend=cos` 仍打旧 multipart upload → 410
- pathRule 含 `..` 或未知占位符 → 400，不写盘

## 验收标准

见设计文档 S1–S7；测试意图 T1–T12。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 申请人完成证照直传并经 Head 确认 | VendorDocumentUploaded | upload-complete | 无（审计日志） | — |
| 运营修改证照路径规则 | VendorDocPathRuleUpdated | admin-vendor-docs-storage PATCH | 无（审计）；热加载 | — |
| 提交/重提厂商申请 | — | — | — | 沿用 vendor-application-kyc-docs：无 MQ 订阅方 |
