# NFR Clarification: 厂商证照 COS 预签名直传

> 价值流：`docs/superpowers/plans/2026-08-14-vendor-docs-cos-presign-value-stream.md`  
> 默认等级：L2；安全按证照 PII 升 L3。

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 可伸缩性 | 动作 |
|------|---------|------|----------|------|
| POST `/api/ai-provider/vendor-application/upload-url/` | 无路径 ID（身份来自 `X-User-Id`） | 无分片 ID | **L0** | 厂商申请低频；升级触发：日上传 >1 万或需按租户拆桶时在路径加 `tenantId` 并按租户前缀分桶 |
| POST `.../upload-complete/` | 无；file_key 含 userId | userId 适合作对象前缀，不是租户分片键 | L0 | key 必须含 userId；不按 tenant 分片 |
| POST `.../vendor-application/` | 无 | L0 | 同上 |
| GET `.../admin-vendors/{id}/documents/{kind}/` | vendor id | 不适配（运营全局审，低频） | L0 | 保持 staff 全局；升级触发：多区域运营分片时按 vendor.saas_user_id 路由 |
| GET\|PATCH `.../admin-vendor-docs-storage/` | 无 | 运维配置 | L0 | 单进程热加载即可 |
| COS object key `{keyPrefix}/{userId}/...` | userId | 高基数、与读写对齐、稳定 | L1 对象前缀 | **合适的对象前缀键**；与租户库分片无关 |
| 事件 `VendorDocumentUploaded` key | user_id | 审计/无消费者 | L0 | payload 带 user_id；无跨分片查询 |
| FE `/` 镜像市场表单 | 无 tenant 段 | L0 | 主站已有会话 |

**Hard Gate：通过。** 可伸缩性进入类别表，定级 L0（对象前缀 L1）。

## 类别定级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 / 隐私 | L3 | PII 影像；私有桶；无密钥入仓/日志 |
| 完整性 | L2 | Head 校验大小/类型/属主 |
| 可用性 | L2 | COS 不可用 → 4xx/502；本地回退仅存量 |
| 性能 | L2 | 预签名 <200ms；PUT ≤5MiB |
| 可伸缩性 | L0/L1 | API L0；对象前缀按 userId L1 |
| 可观测性 | L2 | event 名稳定；禁签名 URL |

## 质量场景

1. **越权 key**：申请人 complete 他人前缀 → 400，无对象泄露。  
2. **预签名过期**：>300s PUT → COS 403；须重新 upload-url。  
3. **密钥不落日志**：单测扫描 log 不含 Secret/query。  
4. **COS 故障**：Head 失败 → 申请 400/502，不写脏 vendor 行。

## 领域模型影响

- `VendorDocStore` 端口：Put 不经领域（浏览器直传）；领域只认 `Head`/`Get`/`PresignPut`。  
- `VendorDocument` 值对象：file_key + kind + userId。  
- 路径规则是配置值对象，不是聚合。

## 权衡

- 不做 STS、不做多桶、不做 CDN。  
- 不做跨区域复制（升级触发：合规要求异地）。
