# DDD Model: 任务详情 OAuth session userId 解析

> 输入: 价值流 Increment 1 + NFR 澄清（前端增量，无新后端聚合）

## 限界上下文

- **前端 Auth Session（presentation）**：解析当前登录用户 numeric id
- **任务协作 OAuth 引导（presentation）**：任务详情行级 OAuth 绑定 UI

## 前端领域单元（非 Django domain/）

| 单元 | 职责 |
|------|------|
| `SessionUserIdResolver` | `resolveAuthenticatedUserId(): Promise<string>` |
| `TaskDetailOAuthConnectionGateway` | 构建 user-scoped connection URL 并 fetch |

## 不变量

1. cookie 存在 → 不调用 profile
2. cookie 与 profile 均空 → 抛出/展示「缺少 userId」
3. 同一 inflight 窗口内 profile 至多一次

## 跳过说明

无新 `task2app/Saas_project/**/domain/` 文件；后端 OAuth 契约不变。
