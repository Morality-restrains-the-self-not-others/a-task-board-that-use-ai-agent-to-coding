# 设计：taskGateway 交付收尾（Ship Increment）

> 日期：2026-06-02  
> 父设计：`docs/superpowers/specs/2026-06-02-taskgateway-apisix-design.md`（已实现 G1–G6 + Redesign）  
> 触发：`/8-review` 有条件通过 + `/0-auto-flow`  
> 状态：**已实现（2026-06-02 /0-auto-flow）**

## 1. 目标

在 **不重做网关骨架** 的前提下，关闭审查遗留项，使 runAll 全栈可声明「经 `https://172.20.10.3:8443` 完成登录、profile、OAuth start」。

## 2. 范围（In）

| # | 项 | 说明 |
|---|-----|------|
| S1 | profile 换绑读路径 | `profile_replace_phone` 用 `login_methods_resolver` 判断当前手机号，禁止 `LoginMethod.get_active_phone_binding_for_user` |
| S2 | profile 换绑写路径 | `TASKAUTH_ENABLED` 时经 taskAuth internal API 写入 phone login_method；测试环境可保留 ORM fallback |
| S3 | 网关冒烟脚本 | `scripts/ci/smoke_taskgateway.sh`：`routes --check` + 可选 `curl -k` health（无 Docker 时 skip） |
| S4 | 领域 VO 同步 | `taskGateway/domain/route.py`：`AuthMode` 对齐 `deny|none|app-jwt|token` |
| S5 | 计划/审查文档 | 更新 plan 勾选、review 结论、value-stream 若需 |
| S6 | 交付 | 验证通过后 `gh pr create`（若仓库可 git） |

## 3. 非目标（Out）

- APISIX 集群 HA / 生产 Ingress
- 全量 profile 子资源迁 taskAuth
- 替换 `taskContainerGateway` 实现
- 在网关终止 JWT 登录（仍 forward-auth）

## 4. 价值流影响

| 流 | 影响 |
|----|------|
| `user-auth` | profile GET/POST 经网关 → django；换绑写 taskAuth DB |
| `task-gateway` | 新增 smoke 步骤；`django-users-profile` 路由已存在 |
| `git-oauth` | 无变更（start `app-jwt` 已落地） |

## 5. 领域概念（轻量清单）

| 上下文 | 概念 |
|--------|------|
| taskGateway | `GatewayRoute.auth_mode`、`UpstreamRef` |
| accounts (saas) | Profile 聚合（读 resolver、写 delegate） |
| taskAuth | `PhoneLoginMethod` 绑定（internal upsert，对齐 username 模式） |

## 6. 技术方案

### 6.1 profile_replace_phone（S1 + S2）

**读：** `phone_plain = get_identifier_for_method(user_id, 'phone')`；无绑定 → 400（与现文案一致）。

**写：**

```text
if taskauth_enabled():
    forward_to_taskauth PATCH /api/internal/users/{id}/phone-login-method/
else:
    LoginMethod.bind_verified_phone_for_user(...)  # pytest / 未迁移环境
```

taskAuth 新增 `handleUpsertPhoneLoginMethod`（事务内 void 旧 phone 行 + upsert，与 saas `bind_verified_phone_for_user` 语义一致）。

**校验：** 保留 `VerificationCodeService`、`_phone_active_bound_to_other_user`（后者在 taskauth 模式下改查 taskAuth 或 internal lookup）。

### 6.2 冒烟（S3）

```bash
bash scripts/ci/check_taskgateway_routes.sh
# 若 docker compose 可用：
cd taskGateway && bash run.sh start
curl -sk https://172.20.10.3:8443/api/health/ | grep -q ok
```

### 6.3 领域 VO（S4）

`AuthMode = Literal["deny", "none", "app-jwt", "token"]`；`GatewayRoute.auth` 重命名为 `auth_mode`（或保留别名属性）。

## 7. NFR（auto L2，auth L3）

| 属性 | 级别 | 场景 |
|------|------|------|
| 安全 | L3 | 换绑必须验证码；internal phone API 仅 `X-TaskAuth-Internal-Secret` |
| 可用性 | L2 | taskAuth 不可用时 profile 换绑 503（与 identity 一致） |
| 可测试性 | L2 | pytest mock resolver + taskAuth handler 单测 |

## 8. 验收标准

- [ ] `pytest`：profile 换绑读路径不触 saas `accounts_login_method` ORM（`TASKAUTH_ENABLED` mock）
- [ ] taskAuth：`TestUpsertPhoneLoginMethod` 绿
- [ ] `check_taskgateway_routes.sh` 绿
- [ ] smoke 脚本：无 Docker 时 exit 0（skip）；有 Docker 时 health 200
- [ ] 审查文档标记 Ship increment 完成

## 9. 风险

| 风险 | 缓解 |
|------|------|
| `_phone_active_bound_to_other_user` 仍查 saas | taskauth 模式改 internal lookup-by-phone |
| 无 git 仓库 | Step 9 输出变更清单 + 手動 init 说明 |

## 10. 变更记录

- 2026-06-02：Ship increment 初稿（/0-auto-flow Step 1）
