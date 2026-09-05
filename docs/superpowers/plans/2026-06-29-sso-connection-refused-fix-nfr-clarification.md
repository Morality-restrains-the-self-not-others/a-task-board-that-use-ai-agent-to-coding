# NFR 澄清: SSO Bridge Connectivity Fix

> 输入:
> - 设计文档: `docs/specs/sso-connection-refused-fix/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-29-sso-connection-refused-fix-value-stream.md`

## 跳过声明

此变更为 **Bug 修复**（配置绑定地址 + 路径修正 + ALLOWED_HOSTS 开发模式放宽），符合 NFR 跳过条件：

- **配置变更**：`conf/ai/ai-provider/config.yaml` host 改为 0.0.0.0 + allowedExtendHosts
- **路径修正**：`port_config_loader.py` / `run.sh` 配置路径对齐
- **无新增端点**：不引入新 API、新数据流或外部依赖
- **无新业务逻辑**：不改变 SSO 桥接的认证/授权/换票机制

## NFR 概览表

| 类别 | 等级 | 说明 |
|------|------|------|
| 性能 | L0 | 不适用 — 无新增代码路径 |
| 可伸缩性 | L0 | 不适用 |
| 可用性 | L2 | 修复后 ai-provider 在外部 IP 上可达（恢复预期可用性） |
| 安全性 | L0 | 不适用 — SSO bridge JWT 机制不变 |
| 数据一致性 | L0 | 不适用 |
| 可观测性 | L1 | E2E 测试已覆盖 SSO 跳转健康检查 |

## 质量场景

### QS-01: SSO 跳转可达性

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 管理员浏览器 |
| 刺激 | 点击「镜像市场管理（SSO）」链接 |
| 制品 | ai-provider :8010/admin SPA |
| 环境 | 正常 |
| 响应 | 浏览器成功连接（非 ERR_CONNECTION_REFUSED），SPA 加载 |
| 响应度量 | Playwright E2E 测试中 finalUrl 不含 `chrome-error`，直接访问返回 HTTP 200 |

## 领域模型影响

无 — 配置级别修复，不改变领域模型。

## 权衡与边界

### 取舍
- 开发模式 `ALLOWED_HOSTS = ["*"]` 放宽安全约束以换取本地开发便利性

### 明确不做什么
- 不引入 gateway 代理 8010（长期架构改进，非本次范围）
- 不修改 SSO bridge JWT 签发/换票逻辑
- 不添加新的认证机制

### 升级触发条件
- 生产部署时：将 `allowedExtendHosts: ['*']` 替换为具体域名白名单
- gateway 代理化时：需重新评估 NFR（尤其是安全性和可用性等级）
