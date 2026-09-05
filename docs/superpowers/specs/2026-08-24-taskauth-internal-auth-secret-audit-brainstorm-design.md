# 内部鉴权与密钥治理问题细化清单 — 头脑风暴设计文档

- **日期**: 2026-08-24 15:48
- **作者**: claude
- **触发**: 对「另有：internalSecret: "" 跳过内部鉴权、OIDC 签名私钥被 git 跟踪 3 份、gsoidc-dev-secret-do-not-use-in-prod 类占位密钥。」的问题细化罗列
- **性质**: 安全/配置治理评审（问题枚举 + 修复方向）；不新增接口、不新增服务

---

## 1. 问题全景（TL;DR）

| # | 问题族 | 严重性 | 影响面 | 状态 |
|---|--------|--------|--------|------|
| P1 | `internalSecret: ""` fail-open — 内部鉴权形同虚设 | 🔴 高 | 9 处配置 × 8+ 服务；taskAuth 全部内部端点 | 实锤（源码 + 配置） |
| P2 | OIDC 签名私钥被 git 跟踪 4 份，且 4 把密钥互不相同 | 🔴 高 | taskAuth + taskProjectService 子仓 git 历史 | 实锤（git ls-files + 指纹） |
| P3 | `*-do-not-use-in-prod` / `dev-secret` 占位密钥进入生产配置 | 🟠 中高 | GitLab OIDC client secret ×4、网关 internalSecret、relayToTrae secret | 实锤 |
| P4 | 内部鉴权 header 名跨服务不一致（`X-Internal-Secret` / `X-TaskAuth-Internal-Secret` / `X-TaskBill-Internal-Secret`） | 🟠 中 | 服务间调用链 | 实锤 |
| P5 | `signingKeyPath: ""` 默认路径与 git 跟踪文件重合，密钥无法轮换 | 🟡 中 | OIDC token 签发 | 实锤 |

---

## 2. 🔴 P1 — `internalSecret: ""` 跳过内部鉴权（fail-open）

### 2.1 机制

[taskAuth/src/auth_users.go:10-15](taskAuth/src/auth_users.go#L10-L15)：

```go
func requireInternalSecret(r *http.Request) bool {
	if cfg.InternalSecret == "" {
		return true              // ← 空配置 = 直接放行，不做任何鉴权
	}
	return r.Header.Get("X-TaskAuth-Internal-Secret") == cfg.InternalSecret
}
```

**设计为 fail-open**：`InternalSecret` 未配置时所有内部接口无鉴权放行，本意是降低本地开发门槛，但生产配置同步沿用空值后即成为漏洞。

### 2.2 空值配置清单（9 处）

| # | 文件 | 行 | 所属服务 |
|---|------|----|----------|
| 1 | conf/auth/task-auth/config.yaml | 9 | taskAuth（顶层 InternalSecret） |
| 2 | conf/taskTaskService/config.yaml | 35 | taskTaskService |
| 3 | conf/taskTenantService/config.yaml | 22 | taskTenantService |
| 4 | conf/taskCloudService/config.yaml | 35 | taskCloudService |
| 5 | conf/taskProjectService/config.yaml | 23 | taskProjectService |
| 6 | conf/taskAIComment/config.yaml | 24 | taskAIComment |
| 7 | conf/ai/task-ai-endpoint/config.yaml | 14 | taskAIEndPoint |
| 8 | conf/gateway/task-container-gateway/config.yaml | 29 | 网关→taskCloudService |
| 9 | conf/gateway/task-container-gateway/config.yaml | 33 | 网关→taskAuth |

（另 conf/events/domain-events/config.yaml、conf/billing/task-bill/config.yaml 等引用该键但为上游值或已配值；全仓 `internalSecret` 键共 26 处。）

配置注释明确自述了 fail-open 语义：
```yaml
  # empty = Cloud requireInternalSecret allows when InternalSecret unset
  internalSecret: ""
```

### 2.3 暴露的受保护端点（taskAuth 侧）

`requireInternalSecret` 保护的全部内部端点，空配置下均可匿名访问：

| 文件 | 端点/功能 | 数据敏感度 |
|------|-----------|-----------|
| auth_profile_internal.go（5 处） | 用户资料内部读写 | 🔴 用户 PII |
| auth_access_token.go（5 处） | 访问令牌签发/吊销 | 🔴 会话凭证 |
| auth_company_member.go | 公司成员查询（2 处） | 🔴 组织数据 |
| account_deletion_handlers.go | 账号注销（2 处） | 🔴 账号操作 |
| account_deletion_blockers_client.go | 注销阻塞检查 | 🟠 |
| auth_wechat_linked_account.go | 微信绑定查询 | 🟠 用户 PII |

**后果**：内网（或经网关透传的）任意请求可直接调用用户资料、令牌、成员、注销等内部 API，无任何凭据校验；`X-TaskAuth-Internal-Secret` header 成为摆设。

### 2.4 修复方向

1. **配置层（短期，本次可做）**：9 处空值全部替换为真实随机 secret（如 `openssl rand -hex 32`），由 `conf/` 统一管理（注意现有 `.env*` 敏感文件门禁）；网关两处同步
2. **代码层（建议）**：`requireInternalSecret` 改 fail-closed — 空配置时**拒绝**放行（或至少拒绝非 localhost/非开发模式请求）；对 `127.0.0.1` 来源可保留例外但须显式标注
3. 与「网关 tenant 兜底直连路由」等既有内部链路核对，避免改 fail-closed 后误伤合法调用（见记忆 [[gateway-tenant-catchall-direct-routes]]）

---

## 3. 🔴 P2 — OIDC 签名私钥被 git 跟踪 4 份，且互不相同

### 3.1 git 跟踪实锤

taskAuth 子仓 `git ls-files` 命中 3 份；taskProjectService 子仓 1 份：

| # | 路径 | 跟踪提交 | 内容 |
|---|------|---------|------|
| 1 | taskAuth/db/task-auth/oidc_signing_key.pem | 0d85033 | **生产默认加载路径**（见 P5） |
| 2 | taskAuth/oidc_signing_key.pem | a70792e | 另一把密钥 |
| 3 | taskAuth/src/db/task-auth/oidc_signing_key.pem | 3e4bafd | 第三把密钥 |
| 4 | taskProjectService/db/task-auth/oidc_signing_key.pem | ? | 第四把密钥 |

### 3.2 密钥内容差异

- 4 份均为 **1675 字节、无 passphrase 的未加密 RSA 私钥**
- md5 各不相同；`openssl rsa -pubout` 导出的公钥指纹也各不相同（`18ff…` / `723a…` / `09b2…`）→ **4 把不同的私钥**

### 3.3 风险展开

| 风险 | 说明 |
|------|------|
| **token 伪造** | OIDC signing key 用于签发 ID token / access token。私钥进了 git → 每个 clone 者、CI 缓存、镜像层都有该私钥 → 可离线伪造任意 OIDC token |
| **git 历史不可清除** | 即使删除文件，3 个提交中的 blob 永久保留（改写历史需 force-push + 全 clone 失效，成本高） |
| **环境间签名不一致** | 4 把不同的 key：若不同环境各自加载不同路径（或曾经用过），A 环境签发的 token 在 B 环境验签失败 → 隐晦的 SSO 故障源 |
| **密钥轮换困难** | `kid` 由公钥派生（`keyIDFromPublic`），key 更换后旧 token 全部失效，无 JWKS 多密钥并轨机制 |

### 3.4 修复方向

1. **立即**：确认生产实际加载的是哪一把（`signingKeyPath` 为空 → 默认 `db/task-auth/oidc_signing_key.pem`）；若生产在用 → 生成**新密钥**替换 + 全站 token 失效（用户重登），旧 key 从 git 历史评估清除
2. **从 git 移除**：`git rm --cached` 3+1 份 + `.gitignore` 拒绝 `*.pem`（或至少 `oidc_signing_key*.pem`）+ 评估 `git filter-repo` 清历史（须与团队确认，影响所有 clone）
3. **密钥入安全通道**：私钥仅存部署机/密钥管理，配置 `signingKeyPath` 指向容器外挂载卷；配置留空自动生成的逻辑仅保留给本地开发（且生成路径必须在 `.gitignore` 内）
4. 检查 taskProjectService 为何携带该文件（疑似 db 同步脚本 `drop_shared_auth_tables.sh` 同批误复制），修正同步链

---

## 4. 🟠 P3 — `*-do-not-use-in-prod` / `dev-secret` 类占位密钥进入配置

### 4.1 全量清单

| # | 占位值 | 位置 | 用途 | 风险 |
|---|--------|------|------|------|
| 1 | `gsoidc-dev-secret-do-not-use-in-prod` | conf/auth/task-auth/config.yaml:53 | OIDC bootstrap client 1 secret | 公开已知值 → SSO 登录可伪造 |
| 2 | `gsoidc-dev-secret-do-not-use-in-prod` | conf/auth/task-auth/config.yaml:57 | OIDC bootstrap client 2 secret | 同上 |
| 3 | `gsoidc-dev-secret-do-not-use-in-prod` | gitService/docker-compose.yml:26 | `GITLAB_OIDC_CLIENT_SECRET` 环境变量**默认值** | 未覆盖 env 即用占位 |
| 4 | `gsoidc-dev-secret-do-not-use-in-prod` | conf/infra/git-service-tencent-sh-1/docker-compose.yml | 生产编排 GitLab OIDC secret | 🔴 生产可能直接生效 |
| 5 | `task-container-gateway-local-dev-secret-do-not-use-in-prod` | conf/gateway/task-container-gateway/config.yaml:6 | 网关自身 internalSecret | 与 P1 叠加：有值但公开已知 |
| 6 | `dev-secret` | conf/gateway/task-container-gateway/config.yaml:12 | relayToTrae loopback secret | 低（仅 loopback，但仍是已知值） |
| 7 | `gsoidc-dev-secret-do-not-use-in-prod` | gitService/docs/tencent-sh-1.env.example | 示例文档 | 可接受（example），但建议改成 `<FILL_ME>` |

### 4.2 风险展开

- 占位值**公开在 git 仓库**（且该仓库还跟踪着签名私钥），等于「安全门锁配了公开已知的钥匙」
- GitLab OIDC client secret 是 **SSO 登录**的凭据：攻击者可持占位值以任意 client 身份完成 OIDC 授权流程 → 账号接管面
- `docker-compose.yml` 默认值形态最危险：任何部署环境忘记 export `GITLAB_OIDC_CLIENT_SECRET` 即静默采用占位值，无告警

### 4.3 修复方向

1. 环境变量形态的占位默认值 → 改为**必填校验**（启动时检测到 `do-not-use-in-prod` 前缀/已知占位值即拒绝启动，fail-fast）
2. 生产编排（conf/infra/git-service-tencent-sh-1/）确认实际注入值，替换为真实 secret
3. taskAuth bootstrap client secrets → 真实随机值，与 GitLab 侧 client 注册一致
4. `.env.example` 占位改 `<FILL_ME>`，避免被误抄
5. 可加 pre-commit 门禁：禁止 `*-do-not-use-in-prod`、`dev-secret`、`secret-do-not-use` 进入非 example 配置文件（与现有 `.env*` 敏感门禁同族）

---

## 5. 🟠 P4 — 内部鉴权 header 名跨服务不一致

同一鉴权体系存在 3 种 header 名：

| Header | 使用方 |
|--------|--------|
| `X-Internal-Secret` | account_deletion_blockers_client.go（对 taskAuth/其他服务）、auth_company_member.go |
| `X-TaskAuth-Internal-Secret` | taskAuth `requireInternalSecret` 校验 |
| `X-TaskBill-Internal-Secret` | 对 taskBill 调用（account_deletion_blockers_client.go:30） |

**风险**：服务间调用若 header 名对不上，即便配了 secret 也会被拒（或反过来 — 某服务校验别的名字导致实际未校验）。配置空值 + header 混乱叠加后，鉴权链完全失效。

**修复方向**：统一为单一规范 header（建议 `X-Internal-Secret` + 服务后缀放值内如 `svc:secret`，或全链路统一 `X-Internal-Secret` 同名），列出服务间调用矩阵核对；配合 P1 的 fail-closed 一起做回归测试（现有 `bill_internal_secret_fragment_test.go` 覆盖 taskBill 方向，需扩展到 taskAuth 方向）。

---

## 6. 🟡 P5 — `signingKeyPath: ""` 默认路径与 git 跟踪文件重合

- `conf/auth/task-auth/config.yaml:39`：`signingKeyPath: ""` # 留空自动生成临时 RSA 密钥
- [taskAuth/src/jwt.go:40-42](taskAuth/src/jwt.go#L40-L42)：空路径 → `defaultSigningKeyPath()` = `"db/task-auth/oidc_signing_key.pem"`
- 该路径文件**已存在于 git 跟踪**（P2 #1）→ 生产启动时 `loadPEMKey` 直接命中 git 跟踪的私钥，**自动生成逻辑永远不触发**，注释描述与实际行为不符
- 结果：生产 OIDC 签名私钥 = 仓库里那把公开私钥；「临时密钥」机制形同虚设

**修复方向**：`signingKeyPath` 显式指向部署卷内路径（非工作区相对路径），或空值语义改为「仅开发模式允许生成，生产模式强制要求显式路径 + 文件存在校验」。

---

## 7. 🕸️ Code Review Graph 分析

- **状态**: `partial` — `graph.db` 存在且 status OK，但图建于 meta 仓根（108 nodes / 17 files，branch main @1e01e76，head 已前进至 73dac86），taskAuth / taskProjectService 等子仓符号**未入图**
- 本次问题定位依赖直接源码读取（auth_users.go / jwt.go / config.go / 配置清单），已逐条实锤
- 子仓（taskAuth 等）如需图分析，须在子仓内重建索引；本任务不需要

---

## 8. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| — | — | — | — | **纯配置/安全治理变更，无新业务意图、无服务端状态变更**；密钥轮换与内部鉴权收紧属基础设施横切面，不产生领域事件。若后续实施「密钥轮换 → 全站 token 失效通知」，可考虑 `OidcSigningKeyRotated` 事件（届时补录） |

---

## 9. 架构变更影响

- **当前架构**: v107 🎯 target（enterprise-landscape + application-integration 两视图 current 各 1 份）
- **本次**: **不创建新版本架构文件** — 问题枚举阶段无组件/数据流变更；修复方案（fail-closed 改造、密钥出 git、配置实值化）落地时再按 v108 出 `security-architecture` 相关视图（如新增 `Motivation_Constraint: 内部鉴权 fail-closed`、`Technology_Artifact: OIDC 私钥迁移至卷挂载`）
- 修复后若需在架构图标注安全策略变更 → 按 skill 规则生成 .puml + .diff.archimate + .full.archimate + .mermaid.md 四件套
- **Python 接口审批**: 不触发（本次零新接口；`requireInternalSecret` 逻辑改造属已有接口行为变更）

---

## 10. 修复优先级建议

| 批次 | 内容 | 依赖 |
|------|------|------|
| **P0（本次即可）** | 9 处 `internalSecret: ""` 填真实值；占位默认值 fail-fast 校验 | 无 |
| **P1（1-2 天）** | 新生成 OIDC 签名密钥并切换 + 全站 token 失效重登；`git rm --cached` + .gitignore | P0 |
| **P2（评估后）** | fail-closed 改造 + header 统一 + 回归测试扩展；生产编排真实 secret 核对 | P0/P1 |
| **P3（规划）** | git filter-repo 清历史（需团队 + 全 clone 失效沟通）；`signingKeyPath` 显式化 | P1 |

---

## 11. 待确认问题（需用户拍板）

1. 生产当前 OIDC 签名私钥是否为 `taskAuth/db/task-auth/oidc_signing_key.pem`（提交 0d85033 跟踪的那把）？轮换后的全站 token 失效窗口可接受？
2. `internalSecret` 改 fail-closed 后，本地开发（无 secret）是否需要保留「仅 loopback 放行」例外？
3. git filter-repo 清历史是否执行（影响所有 clone 者）？还是仅 `git rm --cached` + 新密钥轮换（旧私钥即作废）即视为已缓解？
