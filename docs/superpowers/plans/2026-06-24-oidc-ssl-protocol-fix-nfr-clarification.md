# NFR 澄清: OIDC SSL Protocol Fix

> 输入:
> - 设计文档: `docs/specs/oidc-ssl-debug/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-24-oidc-ssl-protocol-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | L2 | GitLab 容器重建后 fix 需在 5 分钟内自动恢复 |
| 安全性 | L1 | 无变更 — 原有 HTTP 通信保持不变 |
| 可维护性 | L2 | 修复步骤文档化，通过脚本自动化注入 |
| 性能 | L0 | 不适用 — 无新增运行时路径 |
| 可伸缩性 | L0 | 不适用 — 单实例配置变更 |
| 数据一致性 | L0 | 不适用 — 无数据模型变更 |
| 容错机制 | L0 | 不适用 — 无新增故障模式 |
| 可观测性 | L1 | 基础 — GitLab 日志中 OIDC discovery 成功/失败可观测 |
| 合规与隐私 | L0 | 不适用 — 无数据收集变更 |

## 逐增量 NFR 分析

### Increment 1: SWD.url_builder HTTP Protocol Fix

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: GitLab 容器重建后，fix 在启动脚本执行后 5 分钟内恢复；SSO 登录可用率 ≥ 99.5%（开发环境）
- **说明**: 当前 fix 通过 `docker exec` 手动注入 initializer。容器重建后 fix 丢失，需脚本化自动恢复。

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 修复步骤完全脚本化；新开发者根据文档可在 10 分钟内理解并重现修复
- **说明**: 当前设计文档 + 诊断脚本已满足；Phase 2 需将 initializer 注入集成到 `gitService/run.sh` 或 `sync_omniauth_oidc.sh`

#### NFR 类别: 可观测性
- **等级**: L1 - 基础
- **量化目标**: GitLab production.log 中 OIDC discovery 失败/成功事件可查询
- **说明**: 已有日志输出 `(openid_connect) Authentication failure!` / `Request phase initiated.`，无需额外埋点

## 质量场景

### QS-01: GitLab 容器重建后 SSO 恢复

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 运维操作 (docker compose down && docker compose up) |
| 刺激 | GitLab 容器重建 |
| 制品 | GitLab Rails initializer (`zzz_fix_oidc_http.rb`) |
| 环境 | 正常部署 |
| 响应 | 启动脚本在 GitLab 就绪后自动注入 initializer 并重启 Rails |
| 响应度量 | `docker exec gitlab gitlab-rails runner "puts SWD.url_builder"` 输出 `URI::HTTP`；手动 SSO 登录成功 |

### QS-02: OIDC Discovery 可诊断性

| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L1 |
| 刺激源 | 用户点击 taskAuth SSO |
| 刺激 | OIDC discovery 请求 |
| 制品 | GitLab production.log |
| 环境 | 正常 |
| 响应 | 日志记录 discovery 成功/失败及关联 correlation_id |
| 响应度量 | `grep "openid_connect" application_json.log` 可检索到 Request phase / Authentication failure 事件 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 无 L2+ 数据/性能/安全 NFR 决策 | 无领域模型影响 | DDD 步骤可跳过的依据 — 纯基础设施修复，无新增领域概念 |

## 权衡与边界

### 取舍
- 选择 monkey-patch (`SWD.url_builder = URI::HTTP`) 而非升级 openid_connect gem —— 避免引入 gem 版本兼容性风险；代价是 GitLab 升级时需重新验证

### 明确不做什么
- 不在 V1 升级 openid_connect gem（需评估 gem 兼容性 + GitLab 回归测试）
- 不在 V1 修复 gateway TLS（APISIX standalone SSL 是独立 issue）
- 不添加 OIDC 请求的性能监控埋点（现有日志已满足可诊断性需求）

### 升级触发条件
- GitLab CE 升级到 20.x → 重新验证 `SWD.url_builder` 默认值和 initializer 兼容性
- openid_connect gem 发布修复版本 → 评估升级 gem 替代 monkey-patch
- 生产环境部署 → 可用性从 L2 升级到 L3（initializer 持久化到 docker-compose volume 挂载）

## 跳过声明

- **性能**: 跳过。修复不引入新的运行时代码路径，OIDC discovery 延迟不受影响。
- **可伸缩性**: 跳过。单实例配置变更，无水平扩展需求。
- **数据一致性**: 跳过。无数据库 schema 或事务边界变更。
- **容错机制**: 跳过。修复不改变 OIDC 调用链的容错行为；原有 HTTP 超时和重试逻辑不变。
- **安全性**: 跳过。通信协议不变（HTTP），无新增认证/授权逻辑。
- **合规与隐私**: 跳过。无数据收集或存储变更。
