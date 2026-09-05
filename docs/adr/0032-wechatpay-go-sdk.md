# ADR-0032: 微信支付必须调用仓内 sdk/wechatpay-go

- **Status:** accepted
- **Date:** 2026-08-22
- **Author:** cursor
- **Deciders:** /goal 调整项目规则（微信支付走官方 Go SDK）

---

## Context

本仓库已将微信支付 APIv3 官方 Go 客户端落在 `sdk/wechatpay-go`（upstream `github.com/wechatpay-apiv3/wechatpay-go`）。`taskBill` 的 Native 下单、退款、回调验签已 import 该 module，但 `go.mod` 仍从模块代理拉 v0.2.21，未 `replace` 到仓内副本；分账等多处仍用 `core.Client` 拼 path，甚至在 client 未初始化时走未签名裸 HTTP。

若不把「仓内 SDK 为唯一客户端」写成架构约束：

- Agent 会再引入 `go-pay` / 手写 `WECHATPAY2-SHA256-RSA2048` / APIv2 XML；
- 签名、平台证书轮换、回调 AEAD 解密与官方实现分叉，资金路径无法审计；
- 仓内 SDK 补丁无法生效，线上与仓库源码漂移。

## Decision

We will treat **`sdk/wechatpay-go`** as the **only** WeChat Pay APIv3 client for this monorepo:

1. 业务 Go 代码只依赖 `github.com/wechatpay-apiv3/wechatpay-go`；对应服务 `go.mod` **必须** `replace` 到仓内 `sdk/wechatpay-go`。
2. 有生成接口时必须调用 `services/<产品>`（如 `payments/native`、`refunddomestic`、`profitsharing`）；回调走 `core/notify`。
3. **仅当** `services/` 没有对应生成方法时，才允许 `core.Client` 的 Get/Post/Request，并在调用点注释缺失的 service 名。
4. 禁止第三方微信支付 SDK、APIv2 XML、以及绕过 `core.Client` 直打 `api.mch.weixin.qq.com` 的裸 HTTP。
5. 微信支付出站仍须直连（元规则：应用启动禁止环境 Proxy）。

## Alternatives Considered

### Alternative 1: 继续从模块代理拉取 wechatpay-go

- **Pros:** 少一条 replace；与 pkg.go.dev 版本号一致。
- **Cons:** 仓内 `sdk/wechatpay-go` 形同摆设；本地补丁与审查面无法锁定。
- **Why rejected:** 用户明确要求调用 `sdk/wechatpay-go`。

### Alternative 2: 业务层手写 APIv3 HTTP + 自签

- **Pros:** 少一个依赖目录。
- **Cons:** 证书轮换、回调解密、敏感字段加密极易出错；与官方 SDK 重复造轮。
- **Why rejected:** 支付路径不允许自研密码学协议。

### Alternative 3: 引入 go-pay 等社区 SDK

- **Pros:** 部分产品封装更「全」。
- **Cons:** 非微信支付官方维护；与仓内已 vendor 的官方 SDK 双栈。
- **Why rejected:** 官方客户端已在仓内，双栈增加合规与签名风险。

## Consequences

### Positive

- Native / 退款 / 分账 / 回调共用同一套签名与证书逻辑。
- Agent 有明确落点：先查 `sdk/wechatpay-go/services/`，再写业务适配。

### Negative / Trade-offs

- 升级官方 SDK 须同步更新 `sdk/wechatpay-go` 与各服务 replace，不能只改版本号。
- 存量分账 `wechatV3Post` 尚未全部换成 `services/profitsharing`（跟踪 OPT）。

### Mitigations

- CI 检查 `go.mod` replace 与禁止第三方 import。
- 新微信支付代码走 `services/*`；无生成接口才用 `core.Client`。

## References

- `.ai/01_project_constraints/59_wechatpay_go_sdk.md`
- `.ai/01_project_constraints/16_payment_kyc_compliance.md`
- `sdk/wechatpay-go/README.md`
- [wechatpay-go](https://github.com/wechatpay-apiv3/wechatpay-go)
