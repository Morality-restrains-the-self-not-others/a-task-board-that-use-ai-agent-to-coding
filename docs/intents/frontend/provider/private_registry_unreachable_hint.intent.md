# 意图：解析镜像架构时提示云厂商内网 Registry 不可达

## 背景与目标

厂商在 `https://provider.daydaymoney.com/` 添加镜像版本时，失焦会调用 `POST /api/vendor/container-images/resolve-target-architectures/` 拉取 manifest。若镜像地址是云厂商 VPC/内网域名（如阿里云 `registry-vpc.cn-qingdao.aliyuncs.com`），平台公网节点无法触及，原先会等待约 20 秒后把原始 `context deadline exceeded` 显示在弹层 `p.err` 上，厂商不知道要改公网地址。

目标：识别云厂商内网/VPC Registry，立即失败并在同一错误节点展示「本平台无法触及、请改用公网地址」的提示（阿里云 ACR 给出 `registry-vpc.` → `registry.` 改写示例）。

## 范围与边界

- 范围内：taskAiProvider 解析架构 API 与厂商门户 `p.err`（经 `api.js` 富化）；taskCloudService 租户侧 `resolve-target-architectures` 同类 fail-fast。
- 范围外：不代理/不打通 VPC 专线；不自动改写用户填入的 URL；不拆分 `VendorPortal.vue`（超 500 行存量）。

## 约束与风险

- 检测只覆盖云厂商 VPC 主机名标签（`registry-vpc` / `vpc` / `vpce` / `internal` / `inner` 等）与 RFC1918，不把 httptest 使用的 loopback 判为内网。
- 错误展示须继续携带 `data-traceId`。
- 禁止出站走环境 Proxy（既有 registry client `Proxy = nil`）。

## 验收标准

1. 填入 `registry-vpc.cn-qingdao.aliyuncs.com/...` 时，接口在 2 秒内返回 400，`detail` 含「无法触及」及公网改写示例。
2. 弹层 `p.err` 展示该提示（后端 detail + 前端 `enrichResolveArchitectureError` 双保险）。
3. 公网地址（`registry.cn-qingdao.aliyuncs.com`、`docker.io`）不被此规则拒绝。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 解析镜像架构（含内网地址拒绝） | — | — | — | — | 只读探测 + 输入校验，无领域状态变更 |

## 实施计划

1. Go：解析前 `RejectPrivateRegistry`；超时错误附提示。
2. 前端：`privateRegistryHint.js` 在 `api.js` 对 resolve 失败富化文案。
3. 单测覆盖阿里云 VPC 样例与公网负例。

## 变更记录

- 2026-08-18：trace `9558a39b-d7d8-4523-92d5-118ed60018e0` 显示 20s 超时打 VPC ACR；增加提示并 fail-fast。
