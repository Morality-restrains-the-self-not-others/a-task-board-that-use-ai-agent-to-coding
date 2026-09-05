---
name: intent-test
description: Run the complete system intent verification suite. Discovers all test-intent definitions across backend, frontend, container, and platform layers, executes the corresponding test suites, and generates a pass/fail/skip regression report. Use when user says "test intents", "verify system intents", "run intent tests", "check if system still meets its design goals", "regression test all services", "intent verification", "run all tests against design docs", or after major refactors/migrations to confirm system integrity.
---

# Intent Test — 系统意图回归验证

## 概述

本技能运行**完整的系统意图测试套件**，验证系统经过变迁后是否仍然符合设计文档中记录的所有测试意图。

**系统现有**: ~95 个意图定义 (`docs/intents/`), ~40 个 `.testIntent` 伴随文档, ~1200 个测试文件 (Go + JS + Python).

## 执行流程

### Step 1: 扫描意图定义 (Discovery)

```bash
# 统计意图覆盖情况
echo "=== 意图定义统计 ==="
echo "意图定义文件: $(find docs/intents -name '*.intent.md' | wc -l)"
echo "测试意图文件: $(find docs/intents -name '*.test-intent.md' -o -name '*.test.intent.md' | wc -l)"
echo "伴随 .testIntent: $(find . -name '*.testIntent' -not -path '*/node_modules/*' -not -path '*/.git/*' | wc -l)"
echo "Go 测试文件: $(find . -name '*_test.go' -not -path '*/vendor/*' | wc -l)"
echo "JS 测试文件: $(find . -name '*.test.js' -o -name '*.spec.ts' -not -path '*/node_modules/*' | wc -l)"
```

查看详细目录: `cat docs/intents/INDEX.md`

### Step 2: 运行自动化测试 (Execution)

根据用户需求选择执行层级：

#### 层级 1 — 快速检查 (dry-run, 无需启动服务)
```bash
bash scripts/run-all-intent-tests.sh --dry-run
```
仅列出所有测试项，不执行。适合快速了解测试范围。

#### 层级 2 — 编译时检查 (无需启动服务)
```bash
bash scripts/run-all-intent-tests.sh
```
执行: 基础设施检查 + Go 编译检查 + JS 单元测试 + 静态分析 + 意图扫描。

#### 层级 3 — 完整测试 (需运行服务)
在服务已启动的情况下，追加 Playwright E2E 测试：
```bash
# 确保服务运行中
curl -s --connect-timeout 2 http://127.0.0.1:4000 >/dev/null && echo "Server UP" || echo "Server DOWN"

# 运行完整测试
bash scripts/run-all-intent-tests.sh

# 单独运行某个 Playwright 测试
bash e2e-tests/oauth-redirect-e2e.sh
bash e2e-tests/people-manage-permission-isolation-e2e.sh
```

#### 层级 4 — JSON 报告 (CI 集成)
```bash
bash scripts/run-all-intent-tests.sh --format json
cat tmp/intent-test-results/intent-test-report-*.json
```

#### 按分类运行
```bash
bash scripts/run-all-intent-tests.sh --category backend    # 仅后端
bash scripts/run-all-intent-tests.sh --category frontend   # 仅前端
bash scripts/run-all-intent-tests.sh --category platform   # 仅平台
```

### Step 3: 手动验证检查 (Manual Verification)

对于无法自动化的测试意图（见下方表格），执行手动检查：

| 意图 ID | 检查方法 | 验证标准 |
|---------|---------|---------|
| B-001 | `ss -tlnp \| grep -E '800[0-9]\|879[0-9]\|4000'` | 无 `127.0.0.1:PORT` |
| B-003 | `grep 'use_proxy' conf/runAll.yaml` | `use_proxy: false` |
| C-001 | 创建任务 + 检查 clone token | SSH/HTTPS clone 可用 |
| M-001 | 微信支付充值流程 | 支付→回调→余额更新 |
| F-* | Playwright 测试 (需 server) | `cd taskFE && npx playwright test` |

### Step 4: 分析报告 (Analysis)

1. **统计**: 通过/失败/跳过数量及通过率
2. **失败分析**: 区分"新回归"和"已知遗留问题"
3. **缺口识别**: 缺失测试意图的服务或未覆盖的意图
4. **建议**: 需要在 `.learnings/OPTIMIZATION_TODOS.md` 中记录的改进项

### Step 5: 生成覆盖率矩阵

```bash
# 快速查看哪些服务缺少 .testIntent 文件
for dir in taskAuth taskBill taskCloudService taskContainerGateway \
    taskCredentialService taskEvents taskGitOauth taskProjectService \
    taskReferral taskTaskService taskTenantService taskAiProvider \
    taskAIComment taskAIEndPoint taskAgentSupport taskGateway taskSSE; do
  go_tests=$(find "$dir" -name '*_test.go' 2>/dev/null | wc -l)
  intents=$(find "$dir" -name '*.testIntent' 2>/dev/null | wc -l)
  echo "$dir: $go_tests Go tests, $intents .testIntent files"
done
```

## 意图分类与测试映射

完整的意图→测试映射参见: [docs/intents/INDEX.md](../../../docs/intents/INDEX.md)

### 快速参考

| 分类 | 意图数 | 可自动测试 | 需手动验证 |
|------|--------|-----------|-----------|
| 后端 (backend/) | ~50 | ✅ Go 单元测试 | B-001 (端口绑定) |
| 前端 (frontend/) | ~40 | ✅ Playwright (需 server) | — |
| 容器 (container/) | 6 | ✅ Go + 集成测试 | C-001 (端到端 clone) |
| 平台 (platform/) | 4 | 部分 | P-001 (邀请码流程) |
| 业务 (root) | 5 | 部分 | M-001 (微信支付) |

## 已知缺口 (执行时需关注)

以下意图**缺少测试定义**，运行时表现为 "SKIP — no test definition":

1. `feature_params_proxy_rewrite_go` — 需补充 .test-intent.md
2. `cloud_domain_tables_task_cloud_migration` — 需补充测试定义
3. `ecs_orphan_double_start_guard` — 需补充测试定义
4. `budget_console_user_api_cloud` — 需补充测试定义
5. `budget_record_usage_batch_thin_removed` — 需补充测试定义
6. `saas_budget_ledger_http_cutover` — 需补充测试定义
7. `tenant_budget_permission_cloud` — 需补充测试定义
8. `top_deliverable_queued_auto_run_schedule` — 需补充测试定义
9. `rewrite_sub_token_python_removed` — 需补充测试定义
10. `task_events_saas_http_cutover` — 需补充测试定义
11. `task_cloud_saas_sqlite_http_cutover` — 需补充测试定义

以下服务**零测试覆盖**:
- `taskGateway` — 0 Go 测试, 0 .testIntent (关键路由层!)
- `taskSSE` — 0 Go 测试, 少量 JS 测试

## 规则约束

- **DRY-RUN 优先**: 首次运行建议 `--dry-run` 了解范围
- **不阻断 CI**: Phase 1-4 可在 CI 无服务环境下运行；Phase 5-6 需要运行环境
- **渐进式修复**: 发现缺口应写入 `.learnings/OPTIMIZATION_TODOS.md`，按优先级逐步补充
- **禁止删除既有测试**: 修改旧测试需审批（`.ai/05_testing_quality/02_test_management_rules.md`）
- **新增意图必须配测试**: 新架构决策 → adr/ + docs/intents/ 两个文件

## 典型使用场景

### 场景 1: 大规模重构后验证
```
/intent-test
→ 运行全部自动化测试 → 报告通过率 → 列出需手动验证的项目
```

### 场景 2: PR 前快速检查
```
/intent-test --category backend --dry-run
→ 确认涉及的意图都有测试覆盖
```

### 场景 3: 发布前完整回归
```
/intent-test (完整运行, 含 Playwright)
→ 100% 通过或记录所有已知失败
```

### 场景 4: 新服务上线前
```
/intent-test
→ 检查是否在 INDEX.md 注册 → 检查是否有意图文档 → 检查是否有测试
```
