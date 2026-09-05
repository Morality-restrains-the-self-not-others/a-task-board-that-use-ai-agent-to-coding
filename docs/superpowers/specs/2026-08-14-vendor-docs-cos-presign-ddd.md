# DDD：厂商证照 COS

## Bounded Context

AI Provider Marketplace / Vendor Onboarding

## 值对象

- `VendorDocument`：UserID + Kind + FileKey + ContentType + Size
- `PathRule`：KeyPrefix + Pattern；白名单占位符

## 聚合

- `Vendor`（已有）：file_key 为值，不把 COS 对象纳入聚合

## 端口

- `VendorDocStore`：PresignPut / Head / Open / SaveLocal
- `EventBus`：Publish(name, payload)

## 领域事件

| 事件 | 触发 | payload（无 PII 影像） |
|------|------|------------------------|
| VendorDocumentUploaded | Head 确认成功 | user_id, kind, file_key |
| VendorDocPathRuleUpdated | 路径规则写回 | staff_id, key_prefix, path_rule |

## 适配器

- `LocalVendorDocStore`（测试/回退）
- `COSVendorDocStore`（sdk/tencent/cos-go-sdk-v5 + DirectClient）
- `LogEventBus`（结构化日志；无 Kafka 配置时即投递实现）
