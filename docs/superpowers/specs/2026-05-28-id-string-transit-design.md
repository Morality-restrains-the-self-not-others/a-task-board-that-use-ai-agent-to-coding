# 设计：Snowflake ID 传输层一律字符串

> **2026-06-01 修订**：与元规则 `.ai/03_technical_implementation/11_id_field_string_transit.md` 对齐——取消「服务内部 int64 流转」，改为**进程内亦 string，仅入库转 DB 类型**。

## 背景

JSON 数字超过 `2^53` 会丢精度，导致租户/账户 ID 错乱（如 `tenant_id=0`）。taskBill 与 billing_bridge 已部分修复，需统一规范。

## 原则

| 层级 | 规则 |
|------|------|
| 进程内（视图、领域、DTO、handler） | 所有 `id` / `*_id` 为 **string** |
| HTTP 请求体 / 响应体 | 所有 Snowflake 及外键 ID 字段为 **JSON 字符串** |
| URL 路径 / Header | 保持字符串（Django `str:tenant_id`；`X-Tenant-Id` 等） |
| Kafka / Redis / 子进程 env | **string** |
| 数据库**入库** | **唯一**允许转为 `int64` / `bigint` 的边界 |
| 数据库**出库** | 读出后立即 `str()` / `FormatInt`，向上层返回 string |
| 禁止 | JSON number 传递 Snowflake ID；进程内以 `int64` 跨层传递业务 ID |

## 范围（本迭代及后续）

- `taskBill/src`：入站解析、出站序列化；`parseIDField` 仅用于写库边界
- `task2app/Saas_project/billing_bridge`：client 与扣费调用方
- 增量推广至 taskEvents、taskAuth 等全部 monorepo 服务

## 价值流影响

- 计费域（taskBill / billing API）
- 任务帖创建扣费、服务器启动扣费内部 API
- 领域事件与跨进程 fan-out（taskEvents）

## 非目标

- 全 monorepo Django 模型 Serializer 一次性改造（后续增量）
- 修改 DB 列类型（仍为 bigint；仅应用层 string 边界）

## 权威规则

完整细则见：`.ai/03_technical_implementation/11_id_field_string_transit.md`
