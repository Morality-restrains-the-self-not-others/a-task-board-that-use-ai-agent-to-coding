# DDD：AutoCloneNestedRepos 项目策略

- **日期**: 2026-08-12

## 限界上下文

- **Project（taskProjectService）**：拥有策略值对象属性 `AutoCloneNestedRepos`。
- **Credential / Container bootstrap**：消费策略；不拥有写权。

## 聚合

`Project` 聚合根字段：`auto_clone_nested_repos: bool`（默认 true）。

## 领域事件

| 业务意图 | 事件 | 例外 |
|----------|------|------|
| 设置是否自动克隆子仓 | — | 配置字段 CRUD，无跨服务最终一致需求；书面例外：不投递 MQ |

## 不变式

1. Discovery（nested-git-repos）独立于策略。
2. 策略 false ⇒ MergeNested 不得向该项目追加子仓 URL。
3. 策略缺失 ⇒ 视为 true。
