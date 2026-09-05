# Domain: value-stream 配置治理（轻量）

> 工具层，非业务 bounded context。

## Value Objects

### FieldName

- **属性:** `service`, `table`, `column`（各 string，小写 snake/kebab）
- **不变量:** 恰好三段；`ParseFieldName` 正则校验
- **工厂:** 从 `providers[].budget_enabled` 等逻辑路径扁平化为 `providers_budget_enabled`

## Domain Service

### ConfigValidator

- **职责:** 对 YAML 配置与设计文档中的 `fields[].name` 执行 FieldName 不变量
- **实现:** `valueStream/src/fields.go` + 测试套件
