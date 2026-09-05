# 微信支付必须调用仓内 wechatpay-go SDK（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 最后修改：2026-08-22
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 54 条
- Cursor：`.cursor/rules/wechatpay-go-sdk.mdc`（alwaysApply）
- ADR：`docs/adr/0032-wechatpay-go-sdk.md`
- 门禁：`db/scripts/ci/check_wechatpay_go_sdk.py`
- 自测：`db/scripts/ci/test_check_wechatpay_go_sdk.py`
- 技能：`.claude/skills/wechatpay-payment-integration/`（咨询/选型/排障）；**写代码时本条优先**

## 背景（为何是元规则）

微信支付 APIv3 的请求签名、应答验签、平台证书/公钥轮换、回调 AEAD 解密与敏感字段加解密，必须使用官方客户端。本仓库已 vendor 官方 Go SDK 于 `sdk/wechatpay-go`。若业务自行拼 HTTP 或引入社区 SDK，资金路径会与官方实现分叉，且仓内补丁无法生效。

## 核心原则

**处理微信支付（下单、查单、关单、退款、分账、转账、回调验签/解密、平台证书）时，必须调用仓内 `sdk/wechatpay-go` 中的对应 API。禁止第二套微信支付客户端。**

## 强制要求

### 1. 唯一模块与 replace

- Go import **只允许** `github.com/wechatpay-apiv3/wechatpay-go/...`
- 依赖该 module 的服务 `go.mod` **必须** 含：

```
replace github.com/wechatpay-apiv3/wechatpay-go => ../sdk/wechatpay-go
```

（路径相对该 `go.mod`，解析后须指向仓库根 `sdk/wechatpay-go`。）

- **禁止**只从模块代理拉包而不 replace；**禁止**再 vendor 一份到服务目录。

### 2. 优先类型化 service，其次 core.Client

| 能力 | 必须使用的包 |
|------|----------------|
| Native / JSAPI / H5 / APP 下单与查单 | `services/payments/native`（或 `jsapi` / `h5` / `app`） |
| 退款 | `services/refunddomestic` |
| 分账（订单/接收方/回退/商户比例） | `services/profitsharing` |
| 支付回调验签与解密 | `core/notify` |
| 初始化带自动证书的 Client | `core` + `core/option` |
| 商户私钥 / 平台证书工具 | `utils` |

- **有** `services/<产品>` 生成方法时，**禁止**再用 `core.Client.Post/Get` 拼同一条 `/v3/...` path。
- **没有**生成方法时，才允许 `core.Client` 的 Request/Get/Post；调用点须注释 `WechatPay-SDK-OK: no services/<pkg> for <path>`。
- 出站 Client 须直连（禁止环境 Proxy），见 [23_app_startup_no_env_proxy.md](./23_app_startup_no_env_proxy.md)。

### 3. 禁止

| 禁止 | 替代 |
|------|------|
| `github.com/go-pay/gopay`、`github.com/silenceper/wechat`、`github.com/objcoding/wxpay` 等社区 SDK | `sdk/wechatpay-go` |
| APIv2 XML + MD5/HMAC 自签 | APIv3 + 本 SDK |
| 手写 `Authorization: WECHATPAY2-SHA256-RSA2048` | `core.Client` / service |
| 裸 `http.Client` 直打 `api.mch.weixin.qq.com` / `consts.WechatPayAPIServer` | `core.Client` 或 `services/*` |
| Django/Flask 直调微信支付 APIv3 | 经 **taskBill**（Go 服务优先） |
| 用 conf / 本地百分比伪装微信查询结果 | 调 SDK 查询接口（见失败经验 114/115） |

### 4. 测试与 mock

- 单测可替换 service 方法或注入 `core.Client`（如 `wechatPrepayCall` 接缝）。
- `mode: mock` 只允许跳过真实微信网络，**不得**再实现一套签名协议。
- 禁止用真实商户密钥做自动化测试（见 [16_payment_kyc_compliance.md](./16_payment_kyc_compliance.md)）。

## 触发

- 新增/修改微信支付下单、退款、分账、转账、回调、查单
- `taskBill/src/wechat_*.go`、`conf/billing/wechatPay/`
- Agent 拟「先 curl 调通再封装」或引入其他支付 SDK
- 评审 `go.mod` 新增 `wechatpay` / `gopay` / `wxpay` 依赖

## 与既有规则的关系

| 规则 | 关系 |
|------|------|
| 支付与 KYC 合规（第 17 条 / 16_） | 本条是微信支付**客户端落点**；合规、验签、密钥、日志脱敏仍遵守 16_ |
| 新增接口优先落 Go（第 22 条） | 微信支付 API 所有权在 taskBill，禁止 Python 旁路 |
| 应用启动禁止环境 Proxy（第 25 条） | SDK Client 出站直连 |
| wechatpay-payment-integration Skill | 产品选型/官方文档/排障；**实现必须落本 SDK** |

## 验收

```bash
python3 db/scripts/ci/test_check_wechatpay_go_sdk.py
python3 db/scripts/ci/check_wechatpay_go_sdk.py
rg -n 'replace github.com/wechatpay-apiv3/wechatpay-go' taskBill/go.mod
```

## 变更日志

- 2026-08-22：1.0.0 初版；ADR-0032；仓内 `sdk/wechatpay-go` 为唯一 APIv3 客户端。
