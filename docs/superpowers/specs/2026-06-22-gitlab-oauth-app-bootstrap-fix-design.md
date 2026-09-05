# 设计文档：GitLab OAuth Application 启动期自愈创建

**日期:** 2026-06-22
**状态:** 待审批

---

## 1. 问题陈述

### 观察到的现象

```
[14:08:39] 同步 Doorkeeper Application uid=8d98496251234865578283bbf0e8d65872872b7e435979baa3c8441c13770b5a → scopes: read_repository api read_user
[14:09:35] GitLab 初始化完成，开始同步 OAuth scope。
[14:10:21] Doorkeeper::Application uid=8d98496251234865578283bbf0e8d65872872b7e435979baa3c8441c13770b5a not found
[14:10:21] 提示: OAuth scope 同步未完成，可稍后手动执行 gitService/scripts/sync_local_oauth_app_scopes.sh
```

git-service 启动后，OAuth scope 同步脚本找不到 Doorkeeper Application，导致 GitLab OAuth 授权链路处于破损状态——用户无法完成 GitLab OAuth 授权。

### 根因分析

**两阶段时序存在 bootstrap 不对称：**

```
[手动一次性操作]                           [自动化脚本]
在 GitLab Admin UI 创建 OAuth App        sync_local_oauth_app_scopes.sh
→ 记录 uid + secret 到 YAML             → 仅 UPDATE Doorkeeper Application
                                         → 不处理 "not found" 场景
```

1. Doorkeeper Application 的 `uid` (`8d984962...`) 是**前一个 GitLab 容器实例**的手动创建产物，记录在 `conf/auth/git-oauth/providers/http-localhost-8012.yaml` 的 `client_id` 字段
2. `sync_local_oauth_app_scopes.sh` 只有 UPDATE 逻辑——它假设 Application 已经存在
3. 当 GitLab 容器数据卷被清除（`docker compose down -v`、新环境部署、或磁盘清理），Doorkeeper Application 随数据库一起消失
4. 容器重启后数据卷仍存时，Application 还在，脚本正常工作——这是"偶发"的根因：**只在首次部署或数据卷清理后触发**

### 影响范围

- **用户可见影响：** GitLab OAuth 授权页报 `invalid scope`，或换 token 后 `GET /api/v4/user` 返回 403 → 前端显示 `profile_failed`
- **影响功能：** 自托管 GitLab 仓库的 OAuth 绑定、分支预览、容器层 AccessToken 拉取
- **影响面：** 所有使用 `gitlab-local` provider 的 OAuth 流程

---

## 2. 解决方案

### 核心策略：sync 脚本自愈化（Idempotent Bootstrap）

将 `sync_local_oauth_app_scopes.sh` 的 Ruby 逻辑从 "Update Only" 改为 "Find-or-Create + Update"：

```
┌─────────────────────────────────────────────────────┐
│  sync_local_oauth_app_scopes.sh                     │
│                                                     │
│  docker exec gitlab gitlab-rails runner:            │
│                                                     │
│  1. app = Doorkeeper::Application.find_by(uid: uid) │
│                                                     │
│  2a. if app exists → update scopes, redirect_uri    │
│  2b. if app NOT found → CREATE with:                │
│        - uid          = YAML client_id              │
│        - secret       = YAML client_secret          │
│        - name         = "gitOauth Local GitLab"     │
│        - redirect_uri = YAML redirect_uri           │
│        - scopes       = YAML scope                  │
│        - confidential = true                        │
│                                                     │
│  3. print OK: <name> scopes=<scopes>                │
└─────────────────────────────────────────────────────┘
```

### 为什么创建时显式设置 uid / secret

- `uid` 必须匹配 YAML 中的 `client_id`——这是 gitOauth Django 发起 OAuth authorize 请求时使用的 `client_id`
- `secret` 必须匹配 YAML 中的 `client_secret`——这是 gitOauth Django 在 `POST /oauth/token` 换票时使用的凭证
- Doorkeeper 的 `has_secure_token :uid` 和 `has_secure_token :secret` 在属性为空时自动生成；显式赋值后保存，Doorkeeper 会使用显式值

### 配置来源

同步脚本从 `conf/auth/git-oauth/providers/http-localhost-8012.yaml` 读取：

| YAML 字段 | 用途 | 写入 Doorkeeper 列 |
|-----------|------|---------------------|
| `target.client_id` | OAuth client_id | `uid` |
| `target.client_secret` | OAuth client_secret | `secret` |
| `target.scope` | 授权 scope | `scopes` |
| `target.redirect_uri` | 回调 URL | `redirect_uri` |

### 字段级变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `gitService/scripts/sync_local_oauth_app_scopes.sh` | 修改 | Ruby runner 脚本：`find_by` → `find_or_initialize_by` + 创建逻辑 |
| `conf/auth/git-oauth/providers/http-localhost-8012.yaml` | 不变 | 已包含所需全部字段 |

### 安全考量

- `client_secret` 在 YAML 中以明文存储（现状如此）——本次改动不改变 secret 的存储方式
- 创建操作在 GitLab 容器**内部**执行（`docker exec ... gitlab-rails runner`），不暴露额外攻击面
- `confidential: true` 确保 Application 为机密客户端（客户端可安全保存 secret）

---

## 3. 领域概念清单

> 为后续 `/5-ddd-领域设计驱动` 提供输入。

| 维度 | 概念 |
|------|------|
| **Bounded Context** | `git-service` — 自托管 GitLab 基础设施生命周期管理 |
| **Key Entities** | `Doorkeeper::Application`（GitLab 内建 OAuth 应用实体） |
| **Candidate Aggregates** | OAuth Application Aggregate（root: Doorkeeper::Application；属性: scopes, redirect_uri, secret） |
| **Domain Events** | `OAuthApplicationBootstrapped`（首次创建时）、`OAuthApplicationScopesSynced`（每次更新时） |

---

## 4. 价值流影响分析

### 受影响的现有流

查看 `value-stream.yaml`，与此变更直接相关的流：

| 价值流 | 影响类型 | 说明 |
|--------|----------|------|
| `gitlab-oauth-scope-failfast-governance` | 增强 | 步骤 `gitlab-scope-startup-failfast` 当前只覆盖 Django 启动期 scope 校验；本次修复增强 GitLab 侧的 Application 存在性保证 |
| `gitoauth-binding-state-persistence` | 间接增强 | Application 存在是 OAuth 回调落库的前提；修复后首次部署无需手动创建 |
| `project-detail-repo-oauth-row-action` | 间接增强 | 仓库行 OAuth 入口依赖 GitLab provider 可用 |

### 新价值流？

**不需要新增价值流。** 这是一个 bug 修复——现有 `gitlab-oauth-scope-failfast-governance` 流已定义 fail-fast 语义，本次修复是该流的实现完整性增强。

### 测试影响

| 测试文件 | 影响 | 说明 |
|----------|------|------|
| `tests/test_git_oauth_scope_validation.py` | 无需修改 | 只覆盖 Django 侧的 scope 校验 |
| 新增：`gitService/scripts/test_sync_oauth_app.sh` | 新增 | 集成测试：验证 Find-or-Create 逻辑在 Application 不存在时正确创建 |
| `../../gitOauth/api/tests.py` | 无需修改 | OAuth 流程测试依赖 git-service 运行环境 |

### 字段影响

无需新增 `value-stream.yaml` 字段。本次修改不引入新的数据库表或配置项。

---

## 5. 时序保证

### 修复前（当前行为）

```
GitLab 容器启动
  │
  ├── [OK] 数据卷存在，Application 存在 → UPDATE scopes ✓
  │
  └── [FAIL] 数据卷被清除，Application 不存在 → "not found" ✗
       └── 手动介入：GitLab Admin UI → 创建 Application → 复制 uid/secret 到 YAML
```

### 修复后

```
GitLab 容器启动
  │
  ├── [OK] Application 存在 → UPDATE scopes, redirect_uri ✓
  │
  └── [NEW] Application 不存在 → CREATE with YAML credentials ✓
       └── 无需手动介入，首次启动即自动创建
```

---

## 6. 回滚方案

若创建逻辑引入问题：
1. 恢复 `sync_local_oauth_app_scopes.sh` 到修改前版本（git revert）
2. 手动在 GitLab Admin UI 创建 Application 并更新 YAML

---

## 7. 实现摘要

1. **修改** `gitService/scripts/sync_local_oauth_app_scopes.sh`：Ruby runner 部分
   - 将 `app = Doorkeeper::Application.find_by(uid: uid)` 改为 `app = Doorkeeper::Application.find_or_initialize_by(uid: uid)`
   - 对 `new_record?` 的 Application 设置完整属性（name, secret, redirect_uri, scopes, confidential）
   - 对已存在的 Application 保持现有更新逻辑（同步 scopes 和 redirect_uri）
2. **可选增强**：在 `run.sh` 日志中区分 "Created" vs "Updated" 输出

---

## 8. 下一步

设计文档获批后，进入实现阶段。可选路径：
- **工作空间隔离** → `/2-worktrees-工作隔离`（推荐，为中大型变更创建独立 worktree）
- **价值流映射** → `/3-value-stream-价值流`（跳过隔离，直接进入价值流细化）
- **重新头脑风暴** → 调整设计方向
