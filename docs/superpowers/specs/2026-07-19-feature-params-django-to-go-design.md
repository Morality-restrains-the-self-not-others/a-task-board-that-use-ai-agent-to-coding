# Feature-Params 公网接口 Django→Go 迁移 — 设计

- **日期**: 2026-07-19
- **状态**: 已采用（goal-mode 自动采用）
- **落点**: 扩展 `taskCloudService`（已有 resolve/env/internal store 客户端）

## 1. 成功标准

1. 公网 CRUD 经 taskGateway 直达 taskCloudService：
   - `GET|POST /api/tenant/{id}/feature-params/`
   - `GET|POST /api/tenant/{id}/workspace/{wid}/feature-params/`
   - `GET|POST /api/personal/feature-params-configs/`
   - `GET|PUT|DELETE /api/personal/feature-params-configs/{id}/`
2. 保留访问门禁：`X-Feature-Params-Access-Context`、`?view=summary`、审计字段语义
3. Django 原公网视图对直连返回 **410**（经网关不再命中）
4. 表仍由 saas-backend 经 **internal-only** upsert/list 提供（过渡期）；Cloud 为公网 API owner
5. Go 单测覆盖门禁 + 脱敏；Django 测试改为 410 / internal upsert

## 2. 方案（选定）

| 层 | 职责 |
|----|------|
| taskGateway | 高优路由 → taskCloudService |
| taskCloudService | 鉴权、成员/workspace 校验、access gate、摘要脱敏、审计、编排 serialize |
| Django internal | 表 CRUD（tenant/workspace/personal upsert + list + audit write） |
| Django public | 410 Gone stub |

**不新建服务**：与 resolve 同限界上下文。  
**暂不迁表**：符合现有「Cloud HTTP→saas store」模式；表迁 Cloud 列为 OPT。

## 3. Python 例外

仅扩展 **已登记** 的 `api/internal/feature-params/` store（approved_python_exceptions），不新增公网 Django 路由。

## 4. 架构图

本次为路由切流 + 公网 handler 落 Go；组件拓扑不变（Cloud 已存在）。**不新增** ArchiMate 视图；更新 `api_route_ownership.yaml` / `api-route-to-owner.md`。

## 5. 事件

无新 MQ：CRUD 同步 HTTP；审计落库例外同前。
