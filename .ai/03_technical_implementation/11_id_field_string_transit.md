# ID 字段字符串传输规范

## 基本信息

- 版本：1.0.0
- 创建日期：2026-06-01
- 最后修改：2026-06-01
- 维护者：Trae AI 团队

## 背景与问题摘要

本 monorepo 大量主键与外键使用 **Snowflake ID**（`bigint`）。JSON 数字超过 `2^53-1` 会在 JavaScript 等运行时丢失精度；Python/Go 若在进程内过早转为 `int`/`int64` 并在多层传递，也会导致跨语言、跨进程边界时不一致。

**统一原则**：ID 仅在**入库**（写入数据库）时转换为 DB 原生类型；在**进程运行**与**进程间交互**全链路保持 **string**。

## 适用范围

- **字段**：名为 `id` 或 `*_id` 的业务标识（含 Snowflake 主键、外键 ID、动态外键双字段中的 ID 段）
- **服务**：`task2app/`、`taskBill/`、`taskEvents/`、`taskAuth/`、`go_relayToTrae/`、`trae-agent/` 等 monorepo 内全部进程
- **交互**：HTTP 请求/响应、Kafka/Redis Stream 事件、内部 REST、gRPC、环境变量、日志字段、Header（如 `X-Tenant-Id`）

## 核心规则（必须遵守）

### 1. 分层边界

| 层级 | ID 形态 | 说明 |
|------|---------|------|
| 进程内（视图、领域服务、DTO、handler、中间变量） | **string** | 禁止在业务层用 `int`/`int64` 持有并向下传递 Snowflake ID |
| 进程间（HTTP、消息队列、子进程 env、日志） | **string** | JSON 中 ID 必须为 `"123..."`，禁止 number |
| 数据库**入库**（INSERT/UPDATE、ORM `.create()`/`.save()`） | **DB 原生类型** | 唯一允许 `int`/`int64`/`bigint` 转换的边界 |
| 数据库**出库**（SELECT、ORM `.get()`/`.filter()`） | 读出后立即转 **string** | Repository/DAO 向上层返回前完成 `str()` / `FormatInt` |

### 2. 转换时机

- **允许转换**：执行写库语句或 ORM 持久化的**紧邻前一步**（如 `parse_id_for_db(s: str) -> int` 仅用于 SQL 绑定参数）
- **禁止转换**：为「方便比较/拼接/日志」在领域层提前 `int()`；为「统一类型」在 handler 入口 parse 后在进程内以整数流转

### 3. 序列化

- **出站**（API、事件、回调）：所有 ID 字段序列化为十进制字符串
- **入站**：接受 string；若收到 JSON number 应拒绝或显式报错（防 float 精度丢失），不得静默 `float64 → int64`
- **非 ID 数值**（金额、计数、`points_delta` 等）：仍按 [API 规范与实现](./02_api_specifications.md) 中数字字符串规则处理；本规范仅约束 ID 类字段

### 4. 动态外键双字段

与 [风格指南 - 动态外键双字段](./06_style_guide.md) 配合：模型标记字段 + **ID 字段**在业务层与事件中均为 string；入库时对 ID 字段做类型转换。

## 语言实现要点

### Python（Django / 侧车）

```python
# 领域层、序列化器、事件 payload：str
tenant_id: str

# 仅 Repository 写库前
def save_order(tenant_id: str, ...) -> None:
    Order.objects.create(tenant_id=int(tenant_id), ...)

# 读库后向上返回
def get_tenant_id(pk: int) -> str:
    return str(pk)
```

- Serializer：`CharField` 或显式 `to_representation` 输出 `str(instance.id)`
- URL 路径参数：保持 `str:tenant_id`，不在 view 入口 `int(tenant_id)` 后传给 domain

### Go

```go
// DTO / handler / 事件 struct
type ChargeRequest struct {
    TenantID string `json:"tenant_id"`
}

// 仅 database/sql 或 ORM 绑定参数处
id, err := strconv.ParseInt(req.TenantID, 10, 64)
```

- 参考实现：`taskBill/src/ids.go` 中 `formatID`（出站）、`stringField`（进程内持有）；`parseIDField` 仅用于**入站解析后立即写库**或校验，解析结果不得存入跨层 DTO

### TypeScript / 前端

- 所有 ID 类型声明为 `string`，禁止 `number` 存储 Snowflake ID

## 禁止事项

- 请求体、事件体、Redis/Kafka payload 中用 JSON **number** 传递 Snowflake ID
- 在多个模块间传递 `int64`/`int` 形式的业务 ID
- Django ORM `filter(id=some_int)` 时，`some_int` 来自未校验的字符串强转且可能非数字（应先用 string 校验，再在查询边界转换；见 `parse_cloud_platform_authorization_pk` 类守卫）

## 与相关规则的关系

- [API 规范与实现](./02_api_specifications.md)：HTTP 层数字字符串规则；ID 字段以本文件为**更具体**约束
- [外键禁止与业务层关联](./06_style_guide.md)：关联 ID 以普通字段存储，业务层 string 维护
- 历史设计 [id-string-transit-design](../../../docs/superpowers/specs/2026-05-28-id-string-transit-design.md) 已按本规范修订（取消「服务内部 int64 流转」）

## 变更日志

- 2026-06-01：版本 1.0.0 - 初版：明确 ID 仅在入库转 DB 类型，进程内与进程间一律 string
