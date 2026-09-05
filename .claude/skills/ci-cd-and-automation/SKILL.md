---
name: ci-cd-and-automation
description: CI/CD 自动化流水线规范。适用于设置或修改构建/部署流水线、配置自动化质量门禁、调试 CI 失败。覆盖多服务 Docker 构建、动态质量门禁、灰度发布策略。
source: adapted from addyosmani/agent-skills
---

# CI/CD 与自动化

## 概述

自动化质量门禁，使每个变更在到达生产环境前都通过测试、lint、类型检查、构建和安全审计。CI/CD 是每个其他技能的执行机制 — 它捕获人类和 agent 遗漏的东西，并且对每个变更都一致执行。

**左移（Shift Left）：** 尽可能早地捕获问题。lint 阶段捕获的 bug 成本是分钟级；同一个 bug 在生产环境捕获成本是小时级。

**更快就是更安全：** 更小的批次和更频繁的发版降低风险，而非增加。

## 项目质量门禁流水线

```
PR/MR 提交
    │
    ▼
┌──────────────┐
│  LINT         │  golangci-lint / eslint / ruff
│  ↓ pass       │
│  TYPE CHECK   │  go vet / tsc --noEmit / mypy
│  ↓ pass       │
│  UNIT TESTS   │  go test ./... / pytest / vitest
│  ↓ pass       │
│  BUILD        │  go build / npm run build / docker build
│  ↓ pass       │
│  INTEGRATION  │  API/DB 测试 (含 Docker 服务)
│  ↓ pass       │
│  E2E          │  Playwright (关键路径)
│  ↓ pass       │
│  SECURITY     │  govulncheck / npm audit / trivy scan
│  ↓ pass       │
│  BUNDLE SIZE  │  (仅前端) bundlesize check
└──────────────┘
    │
    ▼
  Ready for review
```

**没有门禁可被跳过。** Lint 失败就修 lint — 不要禁用规则。测试失败就修代码 — 不要跳过测试。

## 多服务 Docker CI 模式

项目的 20+ 微服务使用 Docker 部署。CI 中使用 service containers：

```yaml
# 集成测试：需要 PostgreSQL + Redis + Kafka
integration:
  services:
    postgres:
      image: postgres:16
    redis:
      image: redis:7
  steps:
    - run: go test ./... -tags=integration
```

## CI 失败反馈给 Agent

CI + AI agent 的力量在于反馈循环：

```
CI 失败
    │
    ▼
复制失败输出 → 喂给 agent:
"CI 流水线失败，错误如下: [paste]
在本地修复并验证后再推送。"

Agent 修复 → 推送 → CI 再跑
```

## 部署策略

### 预览部署

每个 PR 获得预览部署用于手动测试。

### Feature Flags

Feature flag 将**部署**与**发布**解耦：

```
1. DEPLOY with flag OFF      → 代码在生产环境中但不活跃
2. ENABLE for team/beta      → 生产环境内部测试
3. GRADUAL ROLLOUT           → 5% → 25% → 50% → 100%
4. MONITOR at each stage     → 观察错误率、性能、用户反馈
5. CLEAN UP                  → 全量后两周内移除 flag 和死代码
```

### 灰度发布

| 指标 | 推进（绿） | 暂缓调查（黄） | 回滚（红） |
|------|-----------|---------------|----------|
| 错误率 | 基线 ±10% 内 | 高于基线 10-100% | >2x 基线 |
| P95 延迟 | 基线 ±20% 内 | 高于基线 20-50% | >50% 高于基线 |
| 客户端 JS 错误 | 无新错误类型 | 新错误 <0.1% 会话 | 新错误 >0.1% 会话 |

### 回滚计划

每次部署前必须有回滚方案：
- **触发条件**: 错误率 >2x 基线、P95 延迟 >50%、数据完整性异常
- **回滚步骤**: feature flag 关闭 (<1 分钟) / 代码回滚 (<5 分钟) / DB 迁移回滚 (<15 分钟)
- **通知对象**: 团队 channel

## CI 优化

当流水线超过 10 分钟时：

```
慢 CI？
├── 缓存依赖（Go: actions/cache go modules）
├── 并行运行任务（lint, test, build 分成独立并行 job）
├── 只跑已变更的（路径过滤器跳过无关 job）
├── 矩阵构建（分片测试跨多 runner）
└── 更大 runner
```

## 环境管理

```
.env.example        → 已提交 (开发者模板)
.env                → NOT 提交 (本地开发)
CI secrets          → GitHub Secrets / Vault
生产 secrets        → 部署平台 / Vault
```

CI 永远不应有生产 secrets。CI 测试使用独立 secrets。

## 自动化之外

- **Dependabot / Renovate** — 依赖自动更新
- **Branch protection** — main 分支禁止 force-push、需 CI 通过、需 review 批准
- **PR checks** — ≥1 审批 + CI 全绿 才能合并

## 红旗

- 项目中无 CI 流水线
- CI 失败被忽略或静默
- 禁用测试使流水线变绿
- 生产部署无 staging 验证
- 无回滚机制
- Secrets 存储在代码或 CI 配置文件中

## 验证

- [ ] 所有质量门禁存在（lint, types, tests, build, security audit）
- [ ] 流水线在每次 PR 和 push 到 main 时运行
- [ ] 失败阻止合并（branch protection 已配置）
- [ ] Secrets 存储在 secrets manager，不在代码中
- [ ] 部署有回滚机制
- [ ] 测试套件流水线 <10 分钟
