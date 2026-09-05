# 设计：系统管理员按微信关联账号查询订单

- **日期**: 2026-08-23
- **入口**: `/system-admin/order-records/` 查询卡片 Tab
- **架构变更**: 无（不新增服务/聚合/消息；不更新 `docs/architecture/`）

## 背景

管理端已能按交易单号跨租户精确查单。客服常拿到的是微信昵称、绑定手机号或支付 openid，而不是订单号。需要在同一查询卡片增加 Tab，用微信关联账号查单。

## 方案（已采纳）

1. **UI**：查询区 Tab「交易单号 | 微信关联账号」。微信 Tab 文案说明支持昵称 / openid / unionid / 已绑定手机号、邮箱、用户名；精确匹配；无需先选租户。
2. **taskAuth** 内部只读：`GET /api/internal/users/wechat-linked-account/?q=` → `user_ids`。
3. **taskBill** 管理列表增加 `wechat_account`：编排 taskAuth + 本库 `user_id` / `pay_openid` / `pay_unionid` 等值过滤。
4. **互斥**：`order_number` 与 `wechat_account` 不得同时出现。
5. **安全**：接口仅平台员工；内部解析需内部密钥；响应不含 openid；日志不写查询原文。
6. **索引**：`wechat_identity.nickname` 前缀索引；订单 `user_id` / `pay_openid` / `pay_unionid` 等值索引。
7. **事件**：只读，不投递 MQ（书面例外）。

## 非目标

- 模糊昵称、公开查单、退款 Tab 复用、租户端同参数。

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| GET `/api/system-admin/orders/?wechat_account=` | 网关已验证 + `IsPlatformStaff` | 跨租户 |
| GET `/api/internal/users/wechat-linked-account/` | `X-TaskAuth-Internal-Secret` | 内部 |

## 路径分片键（NFR）

| 路径 | 分片 ID | 等级 | 说明 |
|------|---------|------|------|
| `GET /api/system-admin/orders/?wechat_account=` | 无租户 ID | L0 | 超管全局查单，QPS 低；升级触发：管理查单 QPS>20 或全表扫描告警 |
| `GET /api/internal/users/wechat-linked-account/` | 无 | L0 | 等值查找 + 人数上限 50；升级触发：单次扫描>10ms p95 |

## 幂等性

两条路径均为只读 L0：重复 GET 无副作用。前端查询按钮同步门闩防连点；不需要 Idempotency-Key。

## 领域

无新聚合。查询服务编排 Identity（taskAuth）与 Order（taskBill）。不引入领域事件。
