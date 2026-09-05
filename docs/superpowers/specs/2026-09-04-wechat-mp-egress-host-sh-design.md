# 微信服务号 API 出站剥离至 Host sh

- **Date**: 2026-09-04 19:20
- **Author**: cursor
- **Status**: accepted (goal-mode auto)
- **Iteration**: wechat-mp-egress-host-sh
- **ADR**: [ADR-0059](../../adr/0059-wechat-mp-api-egress-host-sh.md)
- **TraceId**: `81612ca5-03e6-42a7-be52-b7b77420b2c5`

## 🔍 Trace 日志分析 (traceId: `81612ca5-03e6-42a7-be52-b7b77420b2c5`)

- **Grafana Trace Dashboard**: http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=81612ca5-03e6-42a7-be52-b7b77420b2c5
- **涉及服务**: task-gateway → task-auth (`route_id=taskauth-wechat-mp-follow-qr`, upstream `:8003`)
- **HTTP**: `POST /api/auth/wechat/mp/follow-qr/` → **503**
- **根因**: `wechat_mp_follow_qr_failed` — 微信返回 `errcode=40164 invalid ip 120.36.185.132 … not in whitelist`
- **对照探针**:
  - INFRA 出口调 `cgi-bin/token` → 40164（`120.36.185.132`）
  - Host sh（`1.117.67.121`）同样调 token → **也是 40164**（该 IP 尚未入白名单）

## 问题陈述

推荐页 `/profile/referral/` 关注闸门依赖 `POST /api/auth/wechat/mp/follow-qr/`（进程 **taskAuth**）。创码调用 `api.weixin.qq.com/cgi-bin/qrcode/create`，受公众平台 **IP 白名单**约束。当前 INFRA 机出口为动态家宽/电信 IPv4 `120.36.185.132`（不稳定），不适合作为长期白名单地址。

## 架构理解确认

根据 current 架构（v131 application-integration / v129 enterprise-landscape 等）：

- 应用层：taskFE → APISIX(taskGateway) → taskAuth 持有微信服务号票据与身份
- 技术层：INFRA 机跑业务栈；Host sh（`1.117.67.121`）跑区域 GitLab 等边缘/区域组件
- 上次交付版本：v131

本次在此基础上：**不迁 taskAuth 整进程**；仅剥离微信服务号 **出站 API** 到 Host sh。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 OK
- `codegraph query createWechatMPSceneQR` → `taskAuth/src/auth_wechat_mp_ticket.go`
- Callers：`issueOrReuseMPFollowTicket`；HTTP 经由 `wechatHTTPPostJSON` / `getWechatMPAccessToken` → `wechatHTTPGet`
- 影响面：所有 `api.weixin.qq.com/cgi-bin/*` 服务号调用（token / qrcode / user info / followers）

## 方案决策

| 方案 | 适合？ | 结论 |
|------|--------|------|
| A. 整进程 taskAuth 迁到 Host sh | ❌ | forward-auth 热路径、auth DB、OIDC、回调入站均绑 INFRA；迁机成本与耦合过高 |
| B. 仅把 INFRA 当前 IP 加入白名单 | ⚠️ 临时 | 可救急，但 `120.36.*` 家宽出口会变，复发率高 |
| C. **微信服务号 API 出站中继部署在 Host sh** | ✅ | sh 出口 `1.117.67.121` 为云主机固定公网；taskAuth 仍在 INFRA 持票/鉴权 |

**采用 C + 运维白名单**：代码侧落地 egress；运维须将 **`1.117.67.121`** 加入公众平台 IP 白名单（当前 sh 探针亦 40164，无白名单则中继仍失败）。

## 目标架构

```
taskFE → Gateway → taskAuth(:8003 INFRA)
                      │  (Idempotency-Key / 票表 / 会话)
                      ▼
              wechat-mp-egress(:8030 Host sh)
                      │  egress IP 1.117.67.121
                      ▼
              api.weixin.qq.com (cgi-bin/*)
```

- **入站**微信回调仍走公网 → Gateway → taskAuth（不变）
- **出站**仅 `api.weixin.qq.com` 且 host 允许列表强制校验
- 鉴权：`X-Internal-Secret`（conf-local），禁止匿名公网滥用

## 配置

- `conf/auth/task-auth/config.yaml`：`wechat.mpEgress.baseUrl`（空=直连，兼容旧行为）
- `conf-local/auth/task-auth/config.yaml`：`wechat.mpEgress.internalSecret` + 生产 `baseUrl: http://1.117.67.121:8030`
- `conf/infra/wechat-mp-egress/config.yaml`：listen host/port（`0.0.0.0:8030`）

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| 签发关注二维码 | （无新事件） | handleWeChatMPFollowQR | 既有票表 + 微信创码 | 仅改出站路径，意图已有；成功路径既有 `wechat_mp_follow_qr_issued` 日志 |
| 扫码绑定 | WechatMpSubscribed（既有） | processWechatMPFollowTicket | 既有 | 不变 |

## 🐍 Python 新增接口

not_applicable — 无 Python 新 endpoint。

## Value Stream Impact

- 影响流：`user-auth` / referral MP follow gate（既有 `referral_mp_follow_bind`）
- 字段：无新表字段；运行时出站路径变更
- 测试：taskAuth 单测覆盖 egress 路由与 allowlist；部署后人工/探针验证 token 非 40164

## 🏛️ 架构变更影响

- **迭代版本**: v132 🎯 target → ship 后 current
- **变更明细**:
  - 🟢 [NEW] `wechat-mp-egress`（Host sh）
  - 🟡 [MODIFIED] taskAuth 微信 cgi-bin 出站经 egress
- **伴生文件**:
  - `docs/architecture/v132-application-integration-20260904-1920-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
  - `docs/architecture/v132-enterprise-landscape-20260904-1920-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`

## 运维门禁（交付后二维码恢复的硬条件）

1. 公众平台 → 开发 → 基本配置 / IP 白名单 → 添加 **`1.117.67.121`**
2. 部署 wechat-mp-egress 于 Host sh :8030
3. conf-local 打开 `mpEgress.baseUrl` 并精准重启 task-auth
4. 探针：`POST /api/auth/wechat/mp/follow-qr/` 返回 200 + `qr_src`

## 权限

- 无新公网 API；egress 仅 internal secret
- follow-qr 仍为 token + Idempotency-Key（既有）
