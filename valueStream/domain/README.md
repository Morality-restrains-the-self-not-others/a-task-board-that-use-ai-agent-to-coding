# ValueStream Domain Model

## Bounded Contexts

- `ValueStreamConfigContext`: 管理价值流配置、业务域顺序、以及配置持久化契约。
- `ValueStreamExecutionContext`: 管理测试运行状态与步骤执行（已存在于 `src/` 实现）。

当前 DDD 建模聚焦 `ValueStreamConfigContext`，目标是为“业务域拖拽改序”提供纯领域层契约。

## Aggregate

- **Aggregate Root**: `ValueStreamCatalog`
  - Entities: `ValueStream`
  - Value Objects: `DomainName`, `DomainOrder`
  - Domain Event: `DomainOrderReordered`

## Repository Contract

- `ValueStreamRepository` 仅定义配置读取与顺序持久化契约，不包含任何 YAML/文件系统实现细节。

