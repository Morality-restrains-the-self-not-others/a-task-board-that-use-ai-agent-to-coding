# 设计文档：gitOauth port_config 字段重命名

**日期：** 2026-05-27  
**状态：** 已批准（无旧字段兼容）

## 变更

| 旧字段 | 新字段 | 语义 |
|--------|--------|------|
| `target.allowedHost` | `target.website` | Git 托管站 origin |
| `service.base` | `service.allowedHost` | gitOauth 服务对外 URL |

归一化运行时字段：`allowedHost` → `website`；`service_base` 仍从 `service.allowedHost` 读取。

## 范围

- `port_config.json`、task2app/gitOauth 归一化、OAuth 路由、daydaymoney nginx、测试与集成文档

## 非目标

- 不改 `django.gitoauth`、`django/vue.allowedHost` 等其他段
- 不改 DDD 值对象 `GitOauthServiceBase` 命名
