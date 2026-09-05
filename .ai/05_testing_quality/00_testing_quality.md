# 测试与质量保障

## 基本信息
- 版本：2.5.3
- 创建日期：2026-01-27
- 最后修改：2026-08-05
- 维护者：Trae AI 团队

## 规则分类

### 核心规则
&gt; 影响测试质量和可靠性的关键规则，必须严格遵守

#### 测试域元索引（根目录）
- 快速索引与 Playwright 凭证环境变量表：项目根目录 [`测试.ai.md`](../../测试.ai.md)（含动态页 Playwright / Agent 执行速记，细则见 `01_core_testing_rules.md`）

#### 提交时随机单元测试与遗留债 10% 修复（元规则）
- 每次提交须随机抽测单元测试；遗留失败同次至少修复发现集合的 10%（ceil，至少 1 个）。详见 [提交时随机单元测试与遗留债 10% 修复](../01_project_constraints/28_commit_random_unit_test_debt_fix.md)；门禁 `db/scripts/ci/run_commit_random_unit_tests.py`

#### Bug 修复必须携带对应回归单元测试（元规则）
- 凡 fix 类提交，暂存区必须包含与被修源码对应的回归单元测试，缺失阻断提交（例外：纯文档/配置/脚本、仅测试提交、`no-test:` 显式豁免）。详见 [Bug 修复必须携带对应回归单元测试](../01_project_constraints/41_bug_fix_unit_test_required.md)；门禁 `db/scripts/ci/check_bug_fix_unit_tests.py`；自测 `db/scripts/ci/test_check_bug_fix_unit_tests.py`

#### 核心测试规则
- 详细内容请参考：[核心测试规则](./01_core_testing_rules.md)

#### 测试管理规则
- 详细内容请参考：[测试管理规则](./02_test_management_rules.md)

#### 测试时使用内存实现
- 详细内容请参考：[测试时使用内存实现（依赖注入规范）](./03_in_memory_services_for_testing.md)

#### BDD 开发流程规范
- 详细内容请参考：[BDD 开发流程规范](./04_bdd_development_workflow.md)

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 &gt; 最佳实践 &gt; 风格指南
  2. 文件级规则 &gt; 目录级规则 &gt; 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- 2026-08-05：版本 2.5.3 - 索引增补「Bug 修复必须携带对应回归单元测试」元规则指针（fix 提交必须携带对应回归单测，缺失阻断）
- 2026-07-19：版本 2.5.2 - 索引增补「提交时随机单元测试与遗留债 10% 修复」元规则指针
- 2026-05-11：版本 2.5.1 - `测试.ai.md` 与 `01_core_testing_rules.md` 增补 Playwright 动态页与智能体执行要点；本索引「测试域元索引」条增加速记指针
- 2026-05-08：版本 2.5.0 - BDD 升为强制级并接入 CI/pre-commit（见 `04_bdd_development_workflow.md`、`scripts/ci/README_DDD_BDD_COMPLIANCE.md`）
- 2026-04-29：版本 2.4.4 - Playwright 环境变量：根目录 `测试.ai.md` 速查；`02_test_management_rules.md` 增补 GitHub 专用变量说明
- 2026-04-21：版本 2.4.3 - 新增测试短信号码目标约定（`01_core_testing_rules.md`、`02_test_management_rules.md`）
- 2026-04-21：版本 2.4.2 - 核心测试规则新增 Playwright 经 9222 端口 CDP 连接非沙盒 Chrome 的约定（见 `01_core_testing_rules.md`）
- 2026-04-11：版本 2.4.1 - 补充测试意图伴随文档模板路径，统一 `.testIntent` 的自然语言编写结构
- 2026-04-11：版本 2.4.0 - 新增测试意图伴随文档规则，要求测试文件配套 `${testFileName}.testIntent` 自然语言描述并同步维护
- 2026-03-20：版本 2.3.0 - 新增 BDD 开发流程规范，要求功能撰写先写测例再写业务职能
- 2026-03-19：版本 2.2.0 - 新增测试时使用内存实现（依赖注入）规范
- 2026-03-16：版本 2.1.0 - 重构为索引文件，规则内容拆分到子文件
- 2026-02-19：版本 2.2.0 - 移除 Puppeteer 相关配置
