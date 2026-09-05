# DDD 领域建模笔记 — TASK_FEATURE_PARAMS_SCOPE

- **日期**: 2026-07-09

## 限界上下文

`FeatureParams`（既有）— 多层级功能参数解析与容器 env 契约。

## 值对象 / 常量

| 名称 | 说明 |
|------|------|
| `FeatureParamsSource` | 既有：`company` / `workspace` / `personal` |
| `TASK_FEATURE_PARAMS_SCOPE` | 新增系统 env 键常量，取值 = `FeatureParamsSource.value` |

## 领域服务变更

| 服务 | 变更 |
|------|------|
| `FeatureParamsEnvSerializer` | `serialize` / `serialize_from_fields` 增加可选 `scope`；非空时在用户变量之后写入系统键 |
| `FeatureParamsApplicationService` | `resolve_and_snapshot` 将 `meta.source.value` 传入 serializer |

## 不新增

实体、仓储接口、领域事件、聚合根。
