# Code Review: taskGateway（APISIX）

> 日期：2026-06-02（复审）  
> 计划：`docs/superpowers/plans/2026-06-02-taskgateway-apisix-plan.md`  
> 设计：`docs/superpowers/specs/2026-06-02-taskgateway-apisix-design.md`

## 结论

**通过（2026-06-02 Ship Increment）** — profile 路由、换绑 taskAuth 路径、冒烟脚本已落地；Docker E2E 待本机安装 `docker compose` 后执行 `smoke_taskgateway.sh`。

**待手测：** Docker Compose E2E（`run.sh start` + `curl -k :8443`）未在本环境执行。

---

## 对照计划

| 计划项 | 状态 | 说明 |
|--------|------|------|
| G1 骨架 + TLS + runAll | ✅ | `taskGateway/`、`conf/task-gateway/`、`runAll.yaml` 已登记 |
| G2 路由拆分 | ✅ | `auth_mode: app-jwt` on oauth/start；callback `none` |
| G3 forward-auth + Django 网关头 | ✅ | `gateway_forward_auth.go`、`CustomTokenAuthentication` |
| G4 CORS / trace / limit / log | ✅ | `request-id`、`file-logger`、`limit-req` on login |
| G5 前端 apiBaseUrl | ✅ | `conf/vue`、vite 条件跳过 proxy |
| G6 CI check | ✅ | `check_taskgateway_routes.sh` + pytest |
| E2E 网关启动 | ❌ 未测 | 环境无 docker compose；runAll health 可能因 TLS 自检失败 |

---

## 自动化验证

| 检查 | 结果 |
|------|------|
| `pytest tests/test_taskgateway_routes_codegen.py` | 通过 |
| `pytest tests/test_gateway_trust_headers_auth.py` | 通过 |
| `go test -run GatewayForwardAuth ./src` (taskAuth) | 通过 |
| `python scripts/ci/check_ddd_bdd_compliance.py` | 通过 |

---

## 阻断 / 重要问题

### ✅ 已修复：`GET profile` 路由（`/7-build` 2026-06-02）

曾误匹配 `taskauth-get-user`（`user_id=profile`）。已新增 `django-users-profile`（priority 845，`/api/accounts/users/profile/*` → django + `token`），并加 `test_profile_routes_to_django_not_taskauth`。

---

### ✅ 已修复（Redesign 2026-06-02）

| 项 | 处理 |
|----|------|
| OAuth start forward-auth | `routes.yaml` → `auth_mode: app-jwt`（无 forward-auth，JWT 在 gitOauth 内校验 `?token=`） |
| profile ORM | `_build_profile_payload` → `login_methods_resolver.get_identifier_for_method`；`test_profile_payload_login_methods_resolver.py` |
| Docker upstream | `TASK_GATEWAY_APISIX_IN_DOCKER=1` → `host.docker.internal`；`run.sh routes-apply` 默认；CI `--check` 同 env |
| TLS 私钥 | `taskGateway/certs/.gitignore` + `certs/README.md` |
| 客户端 `X-User-Id` | `auth_mode: token` 路由 `request-transformer` 剥离客户端网关头 |

---

### 🟠 待验证：Docker E2E

`apisix.yaml` 已生成 `host.docker.internal:800x` upstream。审查环境仅有 `docker` 无 `docker compose` / `docker-compose`，`run.sh start` 未跑通。

→ 手测：`bash taskGateway/run.sh start` 后 `curl -k https://172.20.10.3:8443/api/health/` 与 profile/OAuth start。

---

### ✅ 已修复：profile 换绑（Ship Increment）

`profile_replace_phone` 读 `get_identifier_for_method`、写 `upsert_phone_login_method`；taskAuth `phone-taken` + `phone-login-method` internal API。

---

### 🟡 建议：测试与模型漂移

| 缺口 | 说明 |
|------|------|
| 无 profile 路由 upstream 断言 | codegen 应保证 `profile` 不指向 `up-taskAuth` |
| 无 `auth_mode: app-jwt` 回归 | 应断言 oauth/start 无 `forward-auth` |
| 无 APISIX 集成测 | 仅 YAML 快照 |
| `gateway_forward_auth` | 无有效 token → 200 路径 |
| `taskGateway/domain/route.py` | 仍为 `public`/`protected`，与 `auth_mode` 四分法不同步 |

→ 下一步: **`/7-build-构建`**；领域 VO 同步可选 **`/6-plans-实施计划`** 记一笔。

---

## 做得好的地方

- 声明式 `routes.yaml` + `routes-to-apisix.py --check`，配置漂移可 CI 门禁。
- `deny-internal`、`task-auth`/`git-oauth` 分流与设计一致（除 start 鉴权）。
- Django 网关头 + `TASK_GATEWAY_TRUST_HEADERS` 测试环境默认 false，pytest 直连不受影响。
- 本地 provider `redirect_uri` 已切到 `https://…:8443`。
- `taskGateway/domain/` 轻量 VO，无基础设施泄漏；DDD 检查通过。

---

## 领域模型

`taskGateway/domain/route.py` 边界清晰，无 ORM/Kafka 导入。**无需**回到 `/5-ddd` 除非要把路由发布做成显式聚合服务（V2）。

---

## 审查签字

| 维度 | 判定 |
|------|------|
| 计划对齐 | ✅ G1–G6 + Redesign；plan 文档可补勾选 profile 路由项 |
| 安全 | ✅ token 路由剥离伪造网关头 |
| 可测试性 | ✅ 7 项单测 + DDD；❌ profile 路由未测；❌ 容器 E2E |
| 可运维性 | ⚠️ Docker compose 手测待做 |

**交付建议：** 先修 profile 路由 → `/7-build`；Compose 手测通过后 `/9-ship-交付`。
