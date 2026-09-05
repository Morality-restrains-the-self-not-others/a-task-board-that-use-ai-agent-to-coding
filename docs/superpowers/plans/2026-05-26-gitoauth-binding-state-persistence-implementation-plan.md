# 实施计划: gitOauth 绑定失败留痕与状态化凭据

> 来源：
> - 价值流：`docs/superpowers/plans/2026-05-26-gitoauth-bind-failure-persistence-value-stream.md`
> - NFR：`docs/superpowers/plans/2026-05-26-gitoauth-binding-state-persistence-nfr-clarification.md`
> - DDD：`gitOauth/api/domain/` 下的聚合、值对象、仓储接口、领域服务、领域事件

## 目标

- OAuth callback 先落库，再按 bind 结果更新状态（`pending/active/failed`）。
- bind 失败时保留记录并写入 `bind_error`，不再删除。
- `access-for-user` 仅消费 `active` 凭据。
- `summary-for-user` 提供 `bind_status/bind_error`，支持前端与运维诊断。

## 执行顺序约束（DDD 依赖）

1. 先以 `domain/` 契约为准（已完成）  
2. 再实现/调整基础设施与应用层（view/model wiring）  
3. 最后补齐回归测试与发布验证

---

## Task Checklist

### A. 领域契约冻结与一致性校验

- [x] **A1. 冻结聚合与值对象契约**
  - 路径：
    - `gitOauth/api/domain/entities/oauth_credential_binding.py`
    - `gitOauth/api/domain/value_objects/credential_bind_status.py`
    - `gitOauth/api/domain/value_objects/bind_error.py`
  - 验收：
    - 状态迁移仅允许 `pending -> active/failed`
    - `BindError` 不允许敏感字符串直出
  - 命令：
    - `cd gitOauth && python3 -m pytest -q`（若未启用 pytest，见 A4）

- [x] **A2. 冻结仓储接口与领域服务契约**
  - 路径：
    - `gitOauth/api/domain/repositories/oauth_credential_binding_repository.py`
    - `gitOauth/api/domain/services/oauth_credential_binding_lifecycle_service.py`
  - 验收：
    - 领域层不出现 ORM/HTTP 依赖
    - 服务方法覆盖 start/pending、success/active、failed/error 三种路径
  - 命令：
    - `cd gitOauth && rg "django\\.|requests|models\\.Model|ForeignKey" api/domain -g "*.py"`

- [x] **A3. 冻结领域事件契约**
  - 路径：
    - `gitOauth/api/domain/events/oauth_credential_bind_failed.py`
    - `gitOauth/api/domain/events/oauth_credential_bind_activated.py`
  - 验收：
    - 事件命名过去式
    - 失败事件携带安全原因字段

- [x] **A4. 领域合规硬门禁**
  - 验收命令：
    - `cd gitOauth && rg "django\\.|requests|boto3|sqlalchemy|models\\.Model|ForeignKey|OneToOneField|ManyToManyField" api/domain -g "*.py"`
    - 结果必须为空

### B. 基础设施与应用层接线（基于领域契约）

- [x] **B1. 凭据状态字段迁移与模型对齐**
  - 路径：
    - `gitOauth/api/migrations/0008_githubappusercredential_bind_status_and_error.py`
    - `gitOauth/api/models.py`
  - 验收：
    - 线上/本地迁移成功
    - 字段可读写：`bind_status`、`bind_error`
  - 命令：
    - `cd gitOauth && python3 manage.py migrate`

- [x] **B2. GitHub callback 状态化改造**
  - 路径：`gitOauth/api/github_browser_views.py`
  - 验收：
    - callback 落库初始 `pending`
    - bind 成功置 `active`
    - bind 失败置 `failed` 且写 `bind_error`
    - 不删除凭据行

- [x] **B3. GitLab callback 状态化改造**
  - 路径：`gitOauth/api/gitlab_browser_views.py`
  - 验收同 B2（GitLab provider_key 路径）

- [x] **B4. access-for-user 仅消费 active**
  - 路径：`gitOauth/api/github_internal_views.py`
  - 验收：
    - 查询条件含 `bind_status="active"`
    - failed 记录不返回 token

- [x] **B5. summary 扩展状态字段**
  - 路径：
    - `gitOauth/api/github_internal_views.py`
    - `gitOauth/api/swagger_serializers.py`
  - 验收：
    - 返回 `bind_status/bind_error`
    - 向后兼容旧字段

### C. 跨服务消费端行为对齐

- [x] **C1. task2app 分支预览鉴权前置保护**
  - 路径：`task2app/Saas_project/projects/views/project_views.py`
  - 验收：
    - 无 token 且无 cookie 时返回可操作错误，不误报 repo not found
    - 保留 `resolve_error` 便于排查

### D. 测试与回归矩阵

- [x] **D1. gitOauth API 回归**
  - 路径：`gitOauth/api/tests.py`
  - 覆盖：
    - GitHub bind 失败留痕
    - GitLab bind 失败留痕
    - summary 新字段契约
  - 命令：
    - `cd gitOauth && python3 manage.py test api.tests`

- [x] **D2. task2app 侧回归**
  - 路径：`task2app/Saas_project/tests/test_project_branches_gitlab_auth_guard.py`
  - 命令：
    - `cd task2app/Saas_project && python3 -m pytest tests/test_project_branches_gitlab_auth_guard.py tests/test_projects_git_utils.py`

- [x] **D3. 全量最小验收**
  - 命令：
    - `cd gitOauth && python3 manage.py test api.tests`
    - `cd ../task2app/Saas_project && python3 -m pytest tests/test_project_branches_gitlab_auth_guard.py tests/test_projects_git_utils.py`

### E. 发布与验证

- [ ] **E1. 迁移发布检查单**
  - 校验项：
    - `api_githubappusercredential` 新字段存在
    - 历史行默认状态为 `active`（无断流）

- [ ] **E2. 功能验收脚本（手工）**
  - 场景：
    - OAuth 成功 -> `active`
    - bind 失败 -> `failed + bind_error`
    - 再次成功绑定 -> `active + bind_error 清空`

- [ ] **E3. 监控与回滚预案**
  - 监控：
    - `bind_failed` 占比
    - `access-for-user` 404 增量
  - 回滚：
    - 仅回滚应用代码，不回滚已执行迁移（字段保留兼容）

---

## 完成定义（DoD）

- [x] DDD 契约保持纯领域（无基础设施导入）
- [x] callback 失败不删行，且可查询失败原因
- [x] active-only 消费规则生效
- [x] 两侧测试全部通过
- [ ] 迁移与手工验收通过

