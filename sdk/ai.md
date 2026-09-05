# sdk/ 目录规则

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 维护者：Trae AI 团队

## 微信支付

- **SSOT 客户端**：`sdk/wechatpay-go`（官方 APIv3 Go SDK，以源码入库，非嵌套 git clone）。
- 业务代码 **不得** 修改该目录后当作「业务补丁长期分叉」；升级应同步 upstream，必要时再 replace。
- 调用约定见 `.ai/01_project_constraints/59_wechatpay_go_sdk.md` 与 ADR-0032。
- 业务服务 `go.mod` 使用 `replace github.com/wechatpay-apiv3/wechatpay-go => ../sdk/wechatpay-go`（路径按服务目录调整）。

## 其他厂商 SDK

- 阿里云 / 腾讯云等仓内 SDK 仍按 `.ai/03_technical_implementation/03_cloud_sdk_specifications.md` 与 `07_aliyun_sdk_usage.md`。
