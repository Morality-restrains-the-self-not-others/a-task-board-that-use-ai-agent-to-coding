# DDD — ImageMarket 恢复厂商申请入口

无新有界上下文。复用 taskAiProvider **厂商门户**上下文。

| 概念 | 说明 |
|------|------|
| Bounded context | 厂商门户（taskAiProvider）；主站身份（taskAuth）只提供邮箱/短信 |
| Aggregate | Vendor（根：`ai_provider_vendor`，键 saas_user_id / email） |
| VO | 申请材料（公司名、联系人、双证 file_key、E.164 手机） |
| Domain service | `ApplyVendorApplication`（已有） |
| Port | HTTP 已有；本迭代不新增端口 |
| Event | `VendorApplicationSubmitted`（handler 已 `logInfo event=`）；无 MQ 订阅方（既有例外） |

端口-适配器：不改基础设施。UI 从 provider SPA 适配器换到 taskFE 适配器，调用同一 Go API。
