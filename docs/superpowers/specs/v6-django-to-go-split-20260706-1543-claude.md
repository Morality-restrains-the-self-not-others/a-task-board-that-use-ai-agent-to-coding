# 设计文档：Django → Go 微服务拆分 — 项目/任务/阿里云

- **版本**: v6 🎯 target
- **作者**: claude
- **日期**: 2026-07-06 15:43
- **迭代**: Django 项目/任务/阿里云接口 → 3 个 Go 服务

---

## 1. 背景与动机

当前 Django `saas-backend` (:8001) 承载项目(Project)、任务(Task/Todo)、阿里云(Cloud)三大核心业务模块，代码量约 7000+ 行 views + 1100 行 cloud provider。随着业务增长，Django 单体形成瓶颈：

- **性能**: Python 同步模型在高并发云 API 调用场景下吞吐受限
- **部署独立性**: 云接口变更频繁，每次部署需重启整个 Django
- **团队协作**: 多个团队在同一 Django 代码库中修改易冲突

之前已成功拆分 Auth (:8003)、Billing (:8009)、ContainerGateway (:8014)、Credential (:8015)、AIEndPoint (:8013) 到 Go。本次完成剩余核心模块拆分。

---

## 2. 当前架构

### 2.1 架构版本演进

| 版本 | 状态 | 内容 |
|------|------|------|
| v1 | ✅ current | 初始基线：39 服务组件，18 领域事件消费者 |
| v2 | 🎯 target | taskAIEndPoint + taskCredentialService |
| v3 | 🎯 target | claude-agent Go 重写 |
| v4 | 🎯 target | task2app 接口 Go 拆分 (ContainerGateway/Relay) |
| v5 | ✅ shipped | relay lifecycle 事件消费者 |
| **v6** | **🎯 target** | **项目/任务/阿里云 Go 拆分** |

### 2.2 拆分范围

| 模块 | Django 代码量 | 迁移目标 |
|------|-------------|---------|
| 项目管理 | ~2000行 (15 views) | **taskProjectService** (Go :8016) |
| 任务管理 | ~1500行 (6 views) | **taskTaskService** (Go :8017) |
| 阿里云接口 | ~3500行 (25 views) + ~1100行 (13 provider files) | **taskCloudService** (Go :8018) |
| LLM 预算 | ~500行 | **保留在 taskAIEndPoint** (不迁移) |

---

## 3. 领域概念清单

### 3.1 Bounded Context 划分

```
APISIX Gateway → /api/tenant/<id>/... 路由分发
    ├── taskProjectService (:8016) — Project, Workspace, Access, Deliverable
    ├── taskTaskService (:8017)    — Task (Todo), Comment, AI Comment, FeatureParams
    └── taskCloudService (:8018)   — Aliyun Auth/OAuth, Instance, VPC, Image, Region
```

### 3.2 核心实体

**Project Context** (taskProjectService):
- `Project` (聚合根): id, name, description, company_id, installed_image_id, tags, server_run_template
- `Workspace` (聚合根): id, name, company_id, deliverable_system, task_archive_tier
- `WorkspaceAccess`: workspace_id, user_id, permission_level

**Task Context** (taskTaskService):
- `Todo/Task` (聚合根): id, title, status, workspace_id, project_id, assignee_id
- `Comment`: task_id, user_id, content
- `AITaskComment`: task_id, ai_model, instruct_id
- `TaskFeatureParams` / `TaskFeatureParamsSnapshot`

**Cloud Context** (taskCloudService):
- `CloudPlatformAuthorization` (聚合根): id, company_id, platform, access_key, region
- `CloudServerConfig` (聚合根): id, task_id, instance_id, region, public_ip
- `OAuthToken`: authorization_id, access_token, refresh_token

### 3.3 跨服务依赖

| 调用方 | 被调用方 | 目的 |
|--------|---------|------|
| taskTaskService | taskProjectService | 验证 workspace 存在性 |
| taskCloudService | taskProjectService | 获取 workspace 云平台默认配置 |
| taskCloudService | taskTaskService | 实例创建成功/失败回调更新 task 状态 |
| 所有新服务 | taskAuth (:8003) | JWT 验证 + 租户身份 |
| 所有新服务 | Django (internal) | Company 存在性校验 |

---

## 4. 技术规范

### 4.1 数据库

每个 Go 服务独立 PostgreSQL database：
- `task_project_db` — 项目/工作空间/权限
- `task_task_db` — 任务/评论/功能参数
- `task_cloud_db` — 云平台授权/实例/网络/镜像

跨库查询改为 HTTP API 调用，禁止直连其他服务数据库。

### 4.2 项目结构（对齐现有 taskAuth 模式）

```
taskProjectService/
├── src/
│   ├── main.go, config.go, db.go
│   ├── handlers.go, models.go
│   ├── django_client.go, auth_client.go
│   └── events.go
├── go.mod, go.sum, run.sh, build.sh
```

### 4.3 端口分配

| 服务 | 端口 |
|------|------|
| taskProjectService | 8016 |
| taskTaskService | 8017 |
| taskCloudService | 8018 |

---

## 5. 实施计划

### Phase 0 — 基础设施
- 创建 Go 服务骨架 + PostgreSQL databases + APISIX routes + Prometheus/Loki

### Phase 1 — taskProjectService
- Project/Workspace CRUD + WorkspaceAccess + DeliverableSystem + 数据迁移 + 路由切换

### Phase 2 — taskTaskService
- Task CRUD + Comment + AITaskComment + FeatureParams + 数据迁移 + 路由切换

### Phase 3 — taskCloudService
- Aliyun Provider 移植 + OAuth + Instance/Network/Image/Region + 数据迁移 + 路由切换

### Phase 4 — Django 清理
- 删除 Django views/provider 代码，保留 models 作为兼容层

---

## 6. 架构变更影响

### 新增 (NEW v6)
- `taskProjectService` (Go :8016), `taskTaskService` (Go :8017), `taskCloudService` (Go :8018)
- 3 个 PostgreSQL database
- APISIX 路由规则（3 组）

### 修改 (MODIFIED v6)
- Django `saas-backend` — 移除 projects/cloud views，保留 internal API
- APISIX Gateway — 新增 upstream + route

### 废弃 (DEPRECATED v6)
- Django `projects/views/` (15 文件), `cloud/views/` (25 文件), `cloud/providers/aliyun/` (13 文件)

### 架构交付物

| 文件 | 说明 |
|------|------|
| `v6-enterprise-landscape-20260706-1543-claude.puml` | 企业架构全景图 — Business/Application/Technology 三层 |
| `v6-application-integration-20260706-1543-claude.puml` | 应用组件架构 — 服务拓扑与数据流 |
| `v6-*-*.archimate` | ArchiMate 标准交换格式 (含架构变迁视图) |
| `v6-*-*.mermaid.md` | Mermaid 渲染格式 (GitHub/GitLab 内嵌) |

---

## 7. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Django 模型隐式依赖 (ForeignKey/JSON) | Phase 0 全面审计 |
| 阿里云 SDK Python→Go 移植遗漏 | 逐 API 对照测试，保留 Python 参考 |
| 数据库迁移数据丢失 | 双写验证 → canary 读 → full cutover |
| APISIX 路由切换中断 | shadow traffic → canary → 全量切换 |

## 8. 验证

1. 每个 handler table-driven test
2. Go 服务间 HTTP 集成测试
3. Playwright E2E: 创建项目→创建任务→启动云实例
4. 对比测试：Django vs Go 响应一致性
5. 性能测试：Go ≤ Django 50% 延迟
