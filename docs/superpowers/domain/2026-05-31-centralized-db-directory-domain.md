# Domain: 集中式 SQLite 目录（基础设施）

## Bounded Context

**Database Configuration** — 路径注册与解析，非业务域。

## Value Objects

- `MonorepoRoot` — 含 `db/registry.yaml` 的目录
- `DatabaseFile` — `key`, `relative_path`, `absolute_path`, `engine=sqlite3`

## Aggregates

- `DatabaseRegistry` — 映射 `key → DatabaseFile`；registry.yaml 为真源

## Repository Interfaces

无（纯配置文件，无 ORM）

## Domain Events

无
