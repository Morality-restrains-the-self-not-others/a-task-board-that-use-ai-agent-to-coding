# 微信登录 openid/unionid 跨应用身份设计（brainstorming v64 候选）

- **迭代**: wechat-login-openid-unionid
- **作者**: claude
- **设计日期**: 2026-08-05
- **基于架构**: v63 ✅ current（RBAC 综合设计，taskAuth :8003）
- **设计文档**: 本文（`docs/superpowers/specs/2026-08-05-wechat-login-openid-unionid-design.md`）
- **相关代码**: `taskAuth/src/auth_wechat.go`、`conf/auth/task-auth/config.yaml`、`dataMigrate/taskAuth/`

---

## 1. 背景与问题定义

**场景**：同一微信用户可能通过不同微信应用登录本平台：

| 登录形态 | 微信应用 | openid | unionid |
|---------|---------|--------|---------|
| PC 网页扫码 | 开放平台**网站应用**（qrconnect, `snsapi_login`） | `openid_web` | `U`（同一开放平台账号下） |
| 微信内置浏览器 | **公众号网页授权**（`snsapi_userinfo`） | `openid_oa` | `U`（同上） |
| （未来）小程序 | **小程序**（`code2session`） | `openid_mp` | `U`（同上） |

- 同一开放平台账号下的不同应用，openid **互不相同且无跨应用意义**；unionid **相同且唯一标识该微信用户**。
- 已确认：网页应用与公众号/小程序应用绑定在**同一微信开放平台账号**下（unionid 机制可用）。

**目标**：同一微信用户无论从哪个应用登录，都必须映射到**同一个 taskAuth 账号**；已分裂的账号有可追踪的修复路径。

---

## 2. 当前设计分析（taskAuth/src/auth_wechat.go）

### 2.1 现状流程

1. `GET /api/auth/wechat/login/` → 跳转 `qrconnect`（单一应用配置：单 AppID/Secret/RedirectURI，仅网站扫码）
2. 回调 `code` → 换 `access_token`（可能含 unionid）→ `sns/userinfo`（unionid + openid）
3. `findOrCreateWeChatUser(unionID, openID, ...)`：
   - `identifier = unionID ?: openID`（unionid 优先，缺省回退 openid）
   - 查 `auth_login_method`（`identifier=? AND method_type='wechat'`）→ 命中即返回该用户
   - 未命中 → 新建用户，插入 unionid 行 + （若不同）openid 行
4. 发 token → 重定向 `/auth/login/?wechat_token=...`

### 2.2 缺陷清单

| # | 缺陷 | 后果 |
|---|------|------|
| **G1** | 仅支持单一微信应用（单 AppID + 仅 `qrconnect` 扫码）；无 `snsapi_userinfo`（微信内置浏览器公众号授权）入口 | 「微信浏览器内部登录」当前不存在；未来接入需重写配置模型 |
| **G2** | unionid 缺失时回退 openid → 账户分裂 | 微信仅在用户已关注绑定公众号或已授权过开放平台账号下任一应用时返回 unionid；用户首次经某应用登录可能拿不到 unionid → 同人两账号 |
| **G3** | openid 以裸值存 `auth_login_method.identifier`，无 app 维度 | openid 只在同一应用内有意义；无法跨应用匹配；不同应用不同用户的 openid 值理论上可碰撞 → 匹配语义不严谨 |
| **G4** | 无分裂账户合并路径 | U1 以 openid_A 建号（无 unionid）；unionid 可得后建 U2；后续登录命中 U2，U1 成孤儿。身份漂移 |
| **G5** | 回调无 app 来源标识（state 仅随机串） | 多应用共用回调时无法区分 code 属于哪个 appid |

### 2.3 正确的部分（保留）

- **unionid 优先**是跨应用身份合并的正确地基 — 保留。
- 存量 `auth_login_method` wechat 行（unionid 主行）作为兼容层 — 保留。

---

## 3. 目标设计

### 3.1 身份模型：新增 `wechat_identity` 表（taskAuth DB）

openid 是**应用维度**的身份标识，不适合塞进单列的 `auth_login_method.identifier`（那是登录凭据表，email/phone/wechat 共用一个 namespace）。新增专用身份表：

```sql
-- dataMigrate/taskAuth/029_wechat_identity.sql
CREATE TABLE IF NOT EXISTS wechat_identity (
  id           BIGINT PRIMARY KEY,
  user_id      BIGINT       NOT NULL,
  app_key      VARCHAR(64)  NOT NULL,           -- 应用逻辑名: 'web' / 'inapp' / 'miniapp'
  app_id       VARCHAR(64)  NOT NULL DEFAULT '',-- 微信 AppID (wx...)
  openid       VARCHAR(128) NOT NULL DEFAULT '',-- 应用维度 openid（可为空，存量 unionid 行无法回填）
  unionid      VARCHAR(128) NOT NULL DEFAULT '',-- 跨应用唯一标识
  nickname     VARCHAR(255) NOT NULL DEFAULT '',
  avatar_url   VARCHAR(512) NOT NULL DEFAULT '',
  created_at   DATETIME(6)  NOT NULL,
  updated_at   DATETIME(6)  NOT NULL,
  UNIQUE KEY uk_app_openid (app_key, openid),   -- MySQL UNIQUE 允许多 NULL，空 openid 行不冲突
  KEY idx_unionid (unionid),
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**职责划分**：

| 表 | 职责 |
|----|------|
| `auth_login_method` (method_type='wechat') | 兼容层：主标识行（identifier=unionid；存量无 unionid 用户为 openid 裸值行）。现有查询（`findLoginMethodByIdentifier`、OIDC bridge、登录方式列表）不感知变更 |
| `wechat_identity` | 权威身份映射：`(app_key, openid) → user_id` 与 `unionid → user_id`；记录昵称/头像 |

### 3.2 登录匹配顺序（`findOrCreateWeChatUser` 重构）

```
登录回调拿到 (app_key, openid, unionid?, nickname, avatar)

1. unionid 非空:
   a. 查 wechat_identity by unionid → 命中: 登录该 user；upsert (app_key, openid) → 该 user
   b. 未命中 → 查 (app_key, openid) → 命中且其 user 的 unionid 为空:
       该 openid 用户就是当前 unionid 用户 → 绑定转移: (app_key, openid) 改绑到新 unionid 解析的账号
       （见 3.4 合并策略; 冲突时发 WechatIdentityConflict）
   c. 均未命中 → 新建 user + wechat_identity(app_key, openid, unionid) + auth_login_method(unionid)
2. unionid 为空（首次授权场景）:
   a. 查 (app_key, openid) → 命中: 登录该 user（同一应用内重登，不分裂）
   b. 未命中 → 新建 user + wechat_identity(app_key, openid, unionid='')
        + auth_login_method(openid 裸值，与现状一致，兼容层)
```

要点：
- **unionid 是身份真源**；openid 绑定只做"应用维度别名"，其归属随 unionid 迁移。
- 同一应用内无 unionid 重登走 (app_key, openid) → 不再分裂。
- 新建用户时**不再**插入 openid 裸值行（3.1 已说明）；旧逻辑的 openid 行由迁移处理。

### 3.3 多应用配置模型（conf/auth/task-auth/config.yaml）

```yaml
wechat:
  apps:
    web:                     # 开放平台网站应用 — PC 扫码 (qrconnect, snsapi_login)
      appId: wx625802b55b33608f
      appSecret: "..."
      redirectUri: ${scheme}://${subdomains.base}/api/auth/wechat/web/callback/
      type: qr
    inapp:                   # 公众号网页授权 — 微信内置浏览器 (snsapi_userinfo)
      appId: wx...
      appSecret: "..."
      redirectUri: ${scheme}://${subdomains.base}/api/auth/wechat/inapp/callback/
      type: web_oauth
```

- **向后兼容**：旧顶层 `wechat.appId/appSecret/redirectUri` 自动映射为 `apps.web`；未配置 apps 时仅 web 可用（等价现状）。
- **回调路由按 app 独立 path**（微信侧 redirect_uri 需与申请一致，独立 path 天然区分 app，无需解析 code 归属）：
  - `GET /api/auth/wechat/web/callback/`（兼容旧 `/api/auth/wechat/callback/`，保留别名）
  - `GET /api/auth/wechat/inapp/callback/`
- **state 编码 app_key**：`state = base64url(app_key) + "." + random`；消费时解析 app_key 决定用哪个 AppID 换 token（防混淆）。
- **登录入口**：`GET /api/auth/wechat/login/?app=web|inapp`（默认 web，兼容现状无参调用）。
- 网关：APISIX 已转发 `/api/auth/*` → taskAuth，**无需路由变更**。

### 3.4 分裂防护与合并策略

| 场景 | 处理 |
|------|------|
| A. unionid 可得，命中既有 unionid 行 | 直接登录；upsert 当前 (app_key, openid) 别名 → 同一用户（跨应用自然合并） |
| B. unionid 可得，未命中 unionid，但 (app_key, openid) 已归属某用户且该用户无 unionid | 该 openid 用户即当前 unionid 用户 → **绑定转移**：(app_key, openid) 别名改绑到 unionid 解析出的账号；auth_login_method openid 裸行同步改绑；发 `WechatIdentityLinked` 事件供审计 |
| C. unionid 可得，(app_key, openid) 已归属另一用户且该用户已有**不同** unionid | 外部异常（(app, openid)↔unionid 本应双射）→ **拒绝该 openid 登录 + 发 `WechatIdentityConflict`**，人工介入 |
| D. unionid 缺失 | 仅走 (app_key, openid) 匹配（3.2 步骤 2），新建时插 `wechat_identity(openid, unionid='')`，等 unionid 可得后由场景 B 收敛 |

> ⚠️ **不做用户数据自动合并**：taskAuth 账号关联 taskTenant 的 company/member 等业务数据，自动合并用户实体超出本次范围且危险。场景 B 只做**登录方式归属转移**（身份别名收敛到真源账号），业务数据合并作为后续迭代（锚点：手机号绑定），本次仅事件告警 + 审计日志。

### 3.5 前端（taskFE）

- 登录页保留微信扫码按钮（默认 `app=web`）。
- UA 检测 `MicroMessenger`（微信内置浏览器）→ 显示"微信内一键登录"，走 `app=inapp`（公众号授权）。
- 微信内登录态判断（`wechat_token` 回调）逻辑不变。

### 3.6 数据迁移（dataMigrate/taskAuth/）

`029_wechat_identity.sql`（DDL）+ 迁移逻辑（Go，幂等，data_migrate_log 追踪）：

1. 建表 `wechat_identity`（3.1）。
2. 存量 `auth_login_method` 扫描 `method_type='wechat'`：
   - **unionid 行 + openid 行并存**（同 user_id）→ `wechat_identity(user, 'web', appid, openid, unionid)`；openid 行保留兼容（或按 3.4 逻辑后续收敛）。
   - **仅 unionid 行**（openid 当时缺失/相等）→ `wechat_identity(user, 'web', appid, openid='', unionid)`。
   - **仅 openid 行**（unionid 缺失时建号）→ `wechat_identity(user, 'web', appid, openid, unionid='')`。
3. 存量 openid 裸行无法判断 app → 全部归入 `web`（当前生产仅此一个应用，安全）。

### 3.7 事件与意图对照（设计硬门禁）

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 用户微信登录成功（任意应用） | USER_LOGGED_IN（payload 增 `provider_app`，向后兼容） | handleWeChatCallback | taskEvents SSE | — |
| 微信新用户注册 | USER_CREATED | findOrCreateWeChatUser | taskEvents SSE | — |
| 跨应用身份别名收敛（场景 B 绑定转移） | WechatIdentityLinked 🆕 | taskAuth 合并逻辑 | 审计日志消费者（可选） | — |
| 身份冲突（场景 C / 绑定 C1） | WechatIdentityConflict 🆕 | taskAuth 冲突检测 | 运维告警 | — |
| 用户绑定微信到既有账号 | WechatBound 🆕 | 绑定回调成功分支 | 审计日志消费者（可选） | — |
| 用户解绑微信 | WechatUnbound 🆕 | 解绑接口 | 审计日志消费者（可选） | — |
| openid 别名 upsert（纯内部元数据） | — | — | — | 无跨边界副作用，taskAuth 内部状态 |

**无对应事件**项：openid 别名 upsert 为 taskAuth 内部身份元数据，无跨服务副作用 — 按门禁规则注明例外。

### 3.8 微信绑定（手机号/邮箱登录 → 绑定微信）🆕

**现状缺口**：taskAuth **无任何绑定接口** — `findOrCreateWeChatUser` 是「登录或创建」。手机号用户点「微信登录」会**静默新建账号**（同人两账号），这是账户分裂最常见的触发路径。

**新增绑定能力（Go taskAuth :8003）**：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/auth/wechat/bind/?app=web\|inapp` | **登录态**发起微信授权；state 编码 `app_key + bind 标记 + user_id`（微信回调无法带自定义头，绑定上下文只能走 state） |
| GET | `/api/auth/wechat/bind/callback/` | 绑定回调：unionid 未归属 → 绑到当前 user；已归属当前 user → 幂等成功；**已归属他号 → 409 冲突** |
| DELETE | `/api/auth/wechat/unbind/` | 解绑（可选）：须校验账号剩余 ≥1 种登录方式，避免裸号 |

**绑定流程**：登录态 → 微信授权 → 解析 (app_key, openid, unionid) → `wechat_identity` 绑到当前 user_id + `auth_login_method` wechat 行 → 完成。此后微信登录经 unionid 命中该账号。

> 匹配顺序修订（3.2）：登录/绑定回调统一入口，**首步检查 state 是否带 `bind` 标记** — 绑定分支优先于「新建用户」，杜绝「绑定意图被静默变成新建账号」。

**绑定冲突矩阵**：

| 冲突 | 场景 | 处理 |
|------|------|------|
| C1 微信已属他号 | unionid 已归属另一 user | **409 拒绝** + 提示「该微信已绑定其他账号，是否直接用该微信登录？」；不自动合并 |
| C2 绑定幂等 | 该微信已在当前账号 | 成功（upsert 幂等） |
| C3 遗留孤儿号 | 绑定前该微信曾以新号登录过（分裂已发生） | 别名收敛 + `WechatIdentityConflict` 告警；业务数据合并留后续迭代 |
| C4 手机号被占 | — | 微信绑定不要求手机号验证；换绑手机号属另一功能，不在本设计 |

### 3.9 身份绑定全场景矩阵（含反向绑定与冲突组合）

| # | 场景 | 现状 | 冲突点 | 设计处理 |
|---|------|------|--------|---------|
| S1 | 手机号/邮箱登录 → 绑定微信 | ❌ 无绑定接口；点微信登录会**静默新建账号**（最常见分裂路径） | C1 微信已属他号 | 3.8 绑定流程；冲突 409 + 引导 |
| **S2** | **微信登录 → 绑定手机号（反向）** | ⚠️ 手机号 upsert 仅 **internal API**（`/api/internal/users/id/{id}/phone-login-method/`，需 internal secret），无公网自助流程；走 `phone_register`（注册）会**再次分裂**（手机号未命中 → 新建 U3） | 手机号已被他号占用（`errPhoneTaken → 409` 已有） | **登录态 + SMS 验证码 + upsert**（复用 phone-taken 409）；禁止走注册/登录入口绑手机号 — 语义必须是「绑定」非「注册」 |
| S3 | 一个账号绑多个微信 | — | 无（技术上允许） | 允许；`wechat_identity` 不做 user 维度唯一约束；C1 检查自然阻止同一微信绑两账号 |
| S4 | 一个手机号绑多个微信账号 | internal upsert 已 409 | phone-taken | 409 + 提示换绑/解绑；手机号作为**跨账号合并锚点**（验证码确认同一人）— 即 G4 合并策略的锚点来源 |
| S5 | 解绑后重绑 | — | 无（幂等） | 绑定 C2 幂等分支覆盖；解绑保留 ≥1 登录方式 |
| S6 | 邮箱登录 → 绑定微信 | 同 S1 | 同 S1 | 共用 3.8 绑定流程（与手机号无差别） |

**S2 补充设计**（微信账号补手机号，行业标准要求）：
- 入口：登录态 + SMS 验证码 → `wechat_identity.user` 对应账号 upsert phone 行
- 现状 internal API 复用（已含 phone-taken 409），本次仅需确认公网入口语义（或由 Django 内部调用方控制 — 决策点）
- **禁止路径**：手机号验证码「登录/注册」入口对已登录微信用户 → 未命中 → 新建账号（U3 分裂）

### 3.10 剩余生命周期场景（完整性扫描）

| # | 场景 | 分析 | 设计处理 |
|---|------|------|---------|
| S7 | 微信登录命中已有账号（换设备/旧浏览器） | 无冲突 — 正常登录路径 | unionid 命中即登录（3.2 步骤 1a） |
| S8 | 换绑手机号（旧→新） | 与微信身份无关；phone 行 upsert 覆盖 | 复用 internal upsert + phone-taken 409；本期不改，仅确认微信行不受影响（绑定关系独立） |
| S9 | 账号注销后微信身份复用 | 注销/归档（`is_archived`）后 wechat_identity 保留 → 新登录会命中归档账号被拒（`userIsActive` 已处理） | 注销需级联解绑 wechat_identity（本期标注为注销流程配套，不新增接口） |
| S10 | 企业微信（WeCom）登录 | 企业微信是**另一体系**（corpid + 企业内部 unionid 语义不同），与开放平台 unionid **不互通** | 本期**明确排除**；接入时按独立 provider 建模（`method_type='wecom'`），不复用 wechat_identity |
| S11 | OIDC（gitService SSO）用户绑定微信 | OIDC 是平台自签身份（issuer=网关），与微信身份无冲突；绑定后 wechat_identity 指向同一 user | 共用 3.8 绑定流程，无特殊处理 |
| S12 | 微信登录是否强制补手机号 | 行业常见（小程序强制）；当前微信登录直接建完整用户 | **产品决策点**：若强制，走 S2 绑定语义（登录态+验证码+upsert），**禁止**注册入口；默认本期不强制 |

### 3.11 API 清单

**均为 Go（taskAuth :8003）** — 🐍 Python 门禁**不触发**（零 Python endpoint）：

| 方法 | 路径 | 说明 | 兼容 |
|------|------|------|------|
| GET | /api/auth/wechat/login/ | 微信登录入口，`?app=web\|inapp`，默认 web | ✅ 无参行为与现状一致 |
| GET | /api/auth/wechat/web/callback/ | 网站应用回调（qrconnect） | ✅ 新增；旧 /api/auth/wechat/callback/ 保留别名 |
| GET | /api/auth/wechat/inapp/callback/ | 公众号网页授权回调 | 🆕 |
| GET | /api/auth/wechat/bind/ | 登录态发起微信绑定授权 | 🆕 |
| GET | /api/auth/wechat/bind/callback/ | 绑定回调（含 409 冲突分支） | 🆕 |
| DELETE | /api/auth/wechat/unbind/ | 解绑（校验剩余登录方式） | 🆕 |
| (taskBill) | POST /api/billing/wechat/notify/ | 回调改造：pending 持久化 + 凭据记录 + 幂等 + PAYMENT_SUCCEEDED（已有 path，改造逻辑；**无身份核对**） | 🟡 |
| (内部) | wechat_identity 读写 | taskAuth 内部查询/迁移 | 非 HTTP 接口 |

---

## 4. 支付回调可靠性 + 支付凭据记录（登录/支付解耦）

> 修订（2026-08-05）：移除「回调身份核对映射 userId」— **登录账号与支付账号不强绑定**。
> 用户可能手机号登录 + 微信支付（微信未绑定）、微信登录 + 支付宝支付（后续渠道）等 —
> 支付渠道身份（openid/unionid）与平台登录身份是**正交**的。

### 4.1 现状核实（taskBill :8004）

| 维度 | 现状 | 评价 |
|------|------|------|
| 回调入口 | `POST /api/billing/wechat/notify/`（Go） | ✅ |
| 验签/解密 | wechatpay-go `ParseNotifyRequest`（平台证书验签 + AES 解密） | ✅ 回调已是支付成功唯一标志（不依赖前端跳转） |
| 成功门禁 | `TradeState == SUCCESS` 才入账 | ✅ |
| **pending 存储** | **内存 map** `wechatPending[out_trade_no]` — taskBill 重启即丢失 | ❌ 重启后回调查不到 → 已付款不入账（持续 FAIL → 微信重试） |
| **回调幂等** | `markOrderPaid` 对 `status != pending` 直接 error → 重复回调返回 FAIL → 微信重试风暴 | ❌ |
| **订单归属** | `billing_resource_order`（tenant_id, order_number, status, total_yuan_cents）— **无 user_id** | ❌ 无法回查"哪个用户下单" |
| **支付凭据** | 回调里的 AppID / Payer.OpenID / unionid 被忽略 | ❌ 无法审计"谁付的款" |
| **成功事件** | `markOrderPaid` 仅佣金分账 goroutine + 会员累计同步，**无 Kafka 事件** | ❌ 违反业务意图→MQ 事件门禁 |

### 4.2 目标设计（解耦模型）

**语义**：订单归属 = **下单时登录会话的 user_id**（写入订单）；回调只确认「钱付了」并幂等入账；
回调里的支付方式/appid/openid/unionid **记录为支付凭据**（审计/对账），**不参与任何用户判定**。

1. **`billing_payment_pending` 表**（持久化替代内存 map）：out_trade_no PK、user_id（下单会话）、tenant_id、order_id、amount_fen、status(pending/paid/refunded)、created_at、paid_at
2. **`billing_resource_order` 加列**：`user_id`（下单会话写入）、`pay_method`、`pay_app_key`、`pay_openid`、`pay_unionid`（**回调时记录**，仅审计，不判定）
3. **回调处理链**（taskBill 改造）：

```
验签+解密 → TradeState==SUCCESS → 查 billing_payment_pending(out_trade_no)
  → 金额核对 → 记录支付凭据（pay_method/app_key/openid/unionid，审计）
  → 幂等 markOrderPaid（已 paid → 返回 SUCCESS）→ 发 PAYMENT_SUCCEEDED 事件
```

4. **幂等**：markOrderPaid 增加已 paid → 返回 SUCCESS 分支（重复回调不产生副作用）
5. **不引入**：wechat_identity → userId 的回调解析、身份比对、拒付逻辑 — 支付渠道身份与登录身份解耦

### 4.3 解耦后的边界（明确不做的）

| 场景 | 行为 |
|------|------|
| 手机号登录 + 微信支付（微信未绑定账号） | ✅ 正常入账（订单 user_id = 手机号账号） |
| 微信登录 + 支付宝支付（后续新增渠道） | ✅ 正常入账（订单 user_id = 微信账号） |
| 回调 openid 与订单 user_id 对应微信不一致 | ✅ 仅记录凭据，**不拒付、不告警**（对账由财务审计依据凭据字段人工处理） |
| 同一微信替他人代付 | ✅ 允许（凭据记录 openid，可追溯） |

### 4.4 事件对照（新增行）

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 订单支付成功（回调入账） | PAYMENT_SUCCEEDED 🆕（payload：order_id/user_id/pay_method/pay_openid/金额） | markOrderPaid | 审计/对账/SSE（异步） | — |

## 5. Value Stream 影响

读取 `conf/value-stream.yaml`：受影响流为 **`login`**（fields: `task-auth.auth_login_method.identifier` 等）与注册流（`auth_login_method.is_verified`）。

- 新增 `task-auth.wechat_identity.*` 字段（app_key/openid/unionid → user_id 映射）。
- `auth_login_method.identifier` 的 wechat 语义不变（兼容层），无需改现有 test_file（`accounts/view_test/UserViewSet_login_test.py` 不涉微信）。
- 新增建议流步骤：`wechat-multi-app-login`（qr + inapp 双入口、跨应用合并断言）— 完整切片留给 `/4-value-stream`。

---

## 6. 风险与决策点

| 决策点 | 选项 | 本设计选择 |
|--------|------|-----------|
| 身份存储 | 新表 wechat_identity vs auth_login_method 加 app_id 列 | **新表**（openid 是应用维度身份，非登录凭据；避免污染凭据 namespace） |
| 分裂账户合并 | 自动合并用户数据 vs 仅身份别名收敛 + 告警 | **仅收敛 + 告警**（业务数据合并留后续，锚点手机号绑定） |
| 微信内浏览器入口 | 本期实现 vs 仅加固身份模型 | **本期实现 inapp 入口**（用户已确认场景覆盖目标） |
| 冲突（场景 C） | 拒绝登录 vs 允许登录 | **拒绝 + 告警**（双射异常应暴露而非静默） |
| 手机号账号绑定微信 | 登录态绑定流程 vs 仅提示用微信登录 | **新增绑定流程**（手机号→微信是最常见分裂路径，必须提供显式绑定；绑定冲突 C1 拒绝 + 引导） |
| 微信账号绑定手机号（S2） | 本期一并设计 vs 仅确认现状（internal 已支持） | **本期设计**：登录态 + SMS 验证码 + upsert；公网入口语义需与 Django 调用方确认（决策点）；**禁止**走注册/登录入口绑手机号 |
| 解绑能力 | 本期提供 vs 后续 | 本期提供 DELETE unbind（校验剩余登录方式） |
| 登录/支付绑定关系 | 强绑定（回调身份核对）vs 解耦 | **解耦**（修订 2026-08-05）：订单归属由下单会话决定；回调身份仅作凭据记录，不判定不拒付；避免误伤「手机号登录+微信支付」等正常场景 |
| 支付回调可靠性 | 本期实施 vs 仅设计 | **本期实施**：pending 持久化 + 幂等 + PAYMENT_SUCCEEDED（不依赖身份核对，独立交付） |

## 7. 🏛️ 架构变更影响（v64 🎯 target）

- **迭代版本**: v64 🎯 target
- **迭代名称**: wechat-login-openid-unionid
- **变更视图**: `application-integration`（仅此视图；enterprise-landscape 不涉新服务/新层）
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — 多应用微信 OAuth（配置模型 apps[]、web/inapp 回调、state 带 app_key）+ wechat_identity 表 + 匹配/合并/绑定逻辑
  - 🟡 [MODIFIED] taskFE — 登录页微信入口（UA 检测 → 扫码 / 微信内一键登录）
  - 🟡 [MODIFIED] taskBill — 支付回调可靠性改造（持久化 pending + 凭据记录 + 幂等 + PAYMENT_SUCCEEDED 事件；**无身份核对**）
  - 🟢 [NEW] `wechat_identity` DataObject（taskAuth DB）
  - 🟢 [NEW] `billing_payment_pending` DataObject（taskBill DB）；`billing_resource_order` 加 user_id（下单会话）+ pay_method/pay_openid/pay_unionid（回调凭据记录）
  - 🟢 [NEW] 事件 `WechatIdentityLinked` / `WechatIdentityConflict` / `WechatBound` / `WechatUnbound` / `PAYMENT_SUCCEEDED`（→ Kafka）
  - 🟡 [MODIFIED] USER_LOGGED_IN — payload 增 provider_app（向后兼容）
  - 网关/其他服务：无变更（APISIX `/api/auth/*` 已转发 taskAuth）
- **新增文件**（每个视图三类伴生格式）:
  - 🆕 `docs/architecture/v64-application-integration-20260805-1753-claude.puml`
  - 🆕 `docs/architecture/v64-application-integration-20260805-1753-claude.archimate`（含 Plateau v63→v64 架构变迁视图 + sourceConnection，Archi CLI 加载通过）
  - 🆕 `docs/architecture/v64-application-integration-20260805-1753-claude.mermaid.md`
- **已有文件（未修改）**: v63 current 文件（`v63-application-integration-20260805-0238-claude.*`）

---

## 8. 交付物清单

| 交付物 | 路径 |
|--------|------|
| 设计文档 | `docs/superpowers/specs/2026-08-05-wechat-login-openid-unionid-design.md` |
| 意图文件 | `docs/intents/backend/wechat_login_openid_unionid.intent.md` |
| 架构 .puml | `docs/architecture/v64-application-integration-<ts>-claude.puml` |
| 架构 .archimate | `docs/architecture/v64-application-integration-<ts>-claude.archimate` |
| 架构 .mermaid.md | `docs/architecture/v64-application-integration-<ts>-claude.mermaid.md` |
| 版本历史 | `docs/architecture/VERSION_HISTORY.md`（v64 target 条目） |
| 迁移 SQL | `dataMigrate/taskAuth/029_wechat_identity.sql`（实施阶段） |
