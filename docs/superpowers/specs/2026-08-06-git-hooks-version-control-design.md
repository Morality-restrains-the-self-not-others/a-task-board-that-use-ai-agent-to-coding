# Git Hooks 入库与可跟踪设计 — 统一 hooksPath + 模板 SSOT

- **日期**: 2026-08-06
- **迭代**: git-hooks-version-control
- **作者**: claude
- **状态**: 设计完成，待审批

## 1. 背景与问题

**用户诉求**：各仓库 `.git/hooks/` 下的脚本没有提交到仓库中，导致自动执行无法被跟踪。

`.git/hooks/` 是本地目录（不属于工作树），**天然不入库**。当前钩子部署采用「复制模式」（把钩子文件拷贝进 `.git/hooks/`），由此产生三类不可跟踪性：

| 不可跟踪点 | 表现 |
|-----------|------|
| **内容变更无历史** | 钩子脚本改动不产生 git 提交，无 review、无回滚、无 blame |
| **部署状态不可审计** | 无法回答「哪个仓库装了哪个版本的哪个钩子」；新克隆/新开发机钩子缺失或持有过期版本 |
| **多套机制漂移** | 同质部署脚本 4+ 套并存，且已有双副本漂移实锤 |

## 2. 现状勘察（2026-08-06 实测）

### 2.1 四套并存机制

| 仓库 | 跟踪源 | 激活方式 | 状态 |
|------|--------|---------|------|
| 主仓 ram-work | `.githooks/` ✅（commit-msg/pre-commit/pre-push） | `core.hooksPath=.githooks` ✅ | ✅ 已是目标模式；但 `.git/hooks/` 仍持有同步副本（`install_root_hooks.sh`），双份并存 |
| taskFE / taskReferral | `scripts/hooks/` ✅（含 install.sh，复制到 `.git/hooks/`） | 复制模式，**未设 hooksPath** | ⚠️ 副本不入库；当前与源一致（cmp 验证），但无机制保证 |
| docs | `.githooks/pre-commit` ✅（2026-08-06 ce6ee17 入库） | `install-arch-hooks.sh` 复制，**未设 hooksPath** | ⚠️ **已入库但未激活** — 现状不一致实锤 |
| 其余 ~34 仓 | 无 | — | 无任何门禁 |

### 2.2 部署脚本分散（4+ 套）

| 脚本 | 位置 | 职责 |
|------|------|------|
| `install_root_hooks.sh` | runAll/scripts/ | 主仓 `.githooks/*` → `.git/hooks/*` 副本同步（OPT-20260805-003） |
| `commit_with_submodules.py` | runAll/scripts/ | `--deploy-hooks` / `--install-root-hooks` / `--check-hooks` 批量入口 |
| `deploy_repo_random_precommit.sh` | scripts/ **与 db/scripts/** 双副本 | 按语言模板分发随机单测 pre-commit 到各仓 |
| `install.sh` | taskFE/taskReferral scripts/hooks/（模板分发产物） | 子仓本地复制安装 |
| `install-arch-hooks.sh` | docs/architecture/scripts/ | docs 专属复制安装 |

**漂移实锤**：`scripts/deploy_repo_random_precommit.sh` 与 `db/scripts/deploy_repo_random_precommit.sh` 内容**已不一致**（diff：模板路径解析、仓库清单 `task2app`/`dataMigrate` 差异）。

### 2.3 钩子门禁体系（内容不变，本次只改承载机制）

- **pre-commit** — 随机单测门禁（`.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md`），模板 SSOT 于 `scripts/hooks/templates/`（6 语言变体 + `lib/random_test_runner.sh` 共享抽测库）
- **commit-msg** — bug-fix 回归单测门禁（`.ai/01_project_constraints/41_bug_fix_unit_test_required.md`），适用范围「根仓提交 + 各子仓提交」
- **pre-push**（主仓）— 子仓库未推送阻断（`.ai/01_project_constraints/32_submodule_commit_order.md`）
- **docs pre-commit** — 架构视图自动归档（`archive-old-views.sh`）

## 3. 根因分析

1. **复制模式**：钩子被复制进 `.git/hooks/`（本地、不入库）→ 内容与部署状态均不可跟踪
2. **hooksPath 未推广**：主仓已用 `core.hooksPath=.githooks`（Git 原生、零复制），但从未推广到子仓；docs 有源未激活
3. **多约定并存**：`.githooks/` vs `scripts/hooks/` vs `scripts/hooks/templates/` 三个目录层级语义重叠
4. **无版本戳/台账**：没有任何机制记录钩子版本，无法校验过期

## 4. 目标与非目标

### 目标

1. **钩子源码入库**：每个 git 仓库的钩子真源 = 仓库内被跟踪的 `.githooks/` 目录
2. **Git 原生激活**：`git config core.hooksPath .githooks` — 零复制、零漂移；`.git/hooks/` 退役副本
3. **模板 SSOT 单一化**：共享钩子模板只存在于主仓 `scripts/hooks/templates/`
4. **部署/校验统一入口**：一个分发器 + 一个校验器，基于 `HOOK_VERSION` 版本戳可审计
5. **一键环境引导**：新克隆/新开发机一条命令全仓激活

### 非目标

- 不改变钩子业务逻辑（随机单测 30% 比例、bug-fix 门禁判定规则等保持）
- 不做运行时钩子执行日志系统（如把每次 hook 执行写入审计库）— 本次只保证「钩子本身入库 + 版本可查」；执行时可打印版本戳便于 grep
- 不迁移 `lib/random_test_runner.sh` 到 shareLib（现状已在主仓 SSOT，本次不动）
- 不新增业务 HTTP 接口、不改任何服务运行时行为

## 5. 方案设计

### 5.1 统一目录约定（每个 git 仓库）

```
<repo>/                                # 主仓 + 38 子仓，同一约定
├── .githooks/                         # ✅ 入库真源（git 跟踪）
│   ├── install.sh                     # 幂等激活引导
│   ├── pre-commit                     # 本仓 pre-commit（模板渲染产物或自研）
│   ├── commit-msg                     # bug-fix 门禁（按 SSOT 策略分发）
│   ├── pre-push                       # 主仓专属
│   └── HOOK_VERSION                   # 版本戳清单（渲染时生成）
└── .git/hooks/                        # 退役：仅保留 *.sample，不再放置业务钩子
```

- `git config core.hooksPath .githooks` 激活（仓库级 local config）
- `install.sh` 幂等：检测 `core.hooksPath` 已指向 `.githooks` 则跳过；同时校验版本戳

### 5.2 激活机制

```bash
# 单仓（任意仓库根）
bash .githooks/install.sh          # = git config core.hooksPath .githooks（幂等）

# 全仓一键（主仓根，新增统一入口）
bash runAll/scripts/install-hooks-all.sh   # 遍历主仓 + 全部子仓幂等激活
```

- `install-hooks-all.sh` 被 `commit_with_submodules.py --deploy-hooks` 调用
- 新开发者 README 更新：「克隆 → `bash runAll/scripts/install-hooks-all.sh`」

### 5.3 模板 SSOT 与渲染分发（唯一分发器）

**SSOT**（保持现状、单一）：主仓 `scripts/hooks/templates/` — 6 语言 pre-commit 模板 + `lib/random_test_runner.sh` + `install.sh` 模板 + `deploy` 清单。

**分发器**（升级现有 `deploy_repo_random_precommit.sh` 为唯一实现，消除 db/scripts 副本）：

1. **pick_template** — 沿用现有语言探测逻辑（按仓内 `*_test.go` / `test_*.py` / `*.test.js` 探测，KEEP/SKIP 特例表）
2. **渲染** — 模板插入版本戳头 + 仓名参数，输出到**子仓 `.githooks/`（入库位）**，不再写 `.git/hooks/`
3. **ensure_lib** — `lib/random_test_runner.sh` 同步到各子仓 `scripts/hooks/lib/`（保持现有引用，或一并收敛到 `.githooks/lib/`，见风险 R2）
4. **激活** — 子仓内执行 `bash .githooks/install.sh`（写 hooksPath）
5. **提交提示** — 输出「子仓 `.githooks/` 有变更，请提交」（`commit_with_submodules.py --apply` 自动提交，符合 32 号规则子仓优先）

**db/scripts 副本处理**：删除 `db/scripts/deploy_repo_random_precommit.sh`（同步更新 `commit_with_submodules.py` 的引用路径），消除漂移源。

### 5.4 版本戳与校验（可审计）

- 每个渲染产物文件头携带 `HOOK_VERSION=<semver>`（如 `# HOOK_VERSION 1.3.0`）；`HOOK_VERSION` 由分发器统一递增
- `.githooks/HOOK_VERSION` 清单记录各钩子版本
- **校验**（`commit_with_submodules.py --check-hooks` 升级）：

| 检查项 | 通过条件 |
|--------|---------|
| 激活 | `git config core.hooksPath` == `.githooks` |
| 入库 | `.githooks/pre-commit` 存在且被跟踪（`git ls-files`） |
| 版本 | `.githooks` 内版本戳 == SSOT 期望版本 |
| 无残留 | `.git/hooks/` 无非 sample 业务钩子 |

- 不通过 → `--deploy-hooks` 自动修复；CI（如存在 hooks 校验 job）同标准
- **执行可跟踪（本次落地的最小形态）**：钩子执行时打印 `[hook:pre-commit v1.3.0]`，配合 CI/日志可 grep 确认「自动执行」的钩子版本

### 5.5 迁移计划（分期）

| 阶段 | 内容 | 验收 |
|------|------|------|
| **P0 主仓** | 已有 hooksPath；删除 `.git/hooks/` 业务副本；`install_root_hooks.sh` 改为「hooksPath 校验 + 提示」（不再复制）；`install-hooks-all.sh` 新建 | 主仓 `.git/hooks/` 无业务钩子，hooksPath 生效 |
| **P1 docs** | `install-arch-hooks.sh` 改为写 hooksPath（.githooks 已入库，只差激活） | `git -C docs config core.hooksPath` == `.githooks` |
| **P2 taskFE/taskReferral** | `scripts/hooks/` 内容迁移为 `.githooks/`（源已入库，换安装位）+ hooksPath；旧 `scripts/hooks/install.sh` 退役 | 子仓 `.githooks/` 入库 + 激活 |
| **P3 全量子仓** | 分发器渲染全量子仓 `.githooks/` + 激活 + 子仓提交（钩子入库）；`commit_with_submodules.py --check-hooks` 升级；删除 db/scripts 副本 | `--check-hooks` 全绿；38 仓钩子全部入库 |
| **P4 文档/CI** | `.ai/32_submodule_commit_order.md` 更新 hooksPath 说明；新开发者 README；CI hooks 校验 job | 文档与实现一致 |

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|--------|--------------|---------|
| — | — | — | — | **无对应事件**：本设计为纯开发工具链/CI 配置变更，不改变任何系统事实（无服务状态变更）、不触发跨边界副作用、无用户可感知行为；仅影响开发者本地 git 行为。符合「纯前端无服务端状态变更」例外类别 |

## 7. 🐍 Python 新增接口清单与 Go 替代评估

**未触发**：本设计不新增任何 HTTP 接口（分发器为 bash/python 脚本，非服务端点），不进入 Python 接口专项审批门。

## 8. Value Stream 影响

**无影响**：`conf/value-stream.yaml` 各 stream（用户与认证 / 组织与成员 / 云平台与资源 / 项目与工作空间 / 任务协作等）均以 `<service>.<table>.<field>` 为字段维度，本设计零服务/表/字段变更，不新增 stream、不改变任何 stream 状态。按「纯开发工具链变更」跳过价值流映射。

## 9. 🏛️ 架构变更影响

- **迭代版本**: v13 🎯 target
- **迭代名称**: Git Hooks 入库与统一管理（git-hooks-version-control）
- **作者**: claude
- **设计日期**: 2026-08-06
- **变更视图**: `enterprise-landscape`（Technology 层新增开发工具链元素 + Implementation 层变迁视图）
- **新增文件**（三类伴生格式，缺一不可）:
  - 🆕 `docs/architecture/v13-enterprise-landscape-<ts>-claude.puml`
  - 🆕 `docs/architecture/v13-enterprise-landscape-<ts>-claude.archimate`（含 Plateau v12→v13 架构变迁视图 + sourceConnection 连线）
  - 🆕 `docs/architecture/v13-enterprise-landscape-<ts>-claude.mermaid.md`
- **已有文件（未修改）**: `v12-enterprise-landscape-*.puml` (current)、`v63-application-integration-*.puml` (current) 及其伴生文件
- **变更明细 (vs v12)**:
  - 🟢 [NEW] `Git Hooks 模板 SSOT`（Technology_Artifact — scripts/hooks/templates/，含 6 语言 pre-commit 模板 + random_test_runner.sh）
  - 🟢 [NEW] `Hooks 部署/校验器`（Technology_Artifact — deploy + install-hooks-all + check-hooks，HOOK_VERSION 版本戳）
  - 🟢 [NEW] `.githooks/ 入库真源`（Technology_Artifact — 38 仓统一目录，core.hooksPath 激活）
  - 🔴 [DEPRECATED] `.git/hooks/ 复制模式`（副本同步 install_root_hooks.sh / install-arch-hooks.sh / db 副本）

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v12** | Current 基线 — `.git/hooks/` 复制模式 + 4 套部署脚本并存 |
| **Plateau v13** | Target — 全仓 `.githooks/` 入库 + hooksPath 激活 + 单一分发器 |
| **Gap** | 钩子不入库 → 内容/部署状态/版本不可跟踪；多套机制漂移 |
| **WorkPackage** | WP-git-hooks-vc — P0 主仓 → P1 docs → P2 FE/Referral → P3 全量子仓 → P4 文档/CI |
| **视图** | `架构变迁 v12→v13 — Git Hooks 入库`（可导入 Archi 打开；含 sourceConnection 连线） |

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| R1: hooksPath 为本地配置，新克隆环境仍要执行安装脚本 | 统一 `install-hooks-all.sh` 一键入口 + README/CI 引导；hooksPath 不入库是 Git 设计事实，接受并文档化 |
| R2: 模板渲染产物入库（.githooks 内容含 lib 引用路径） | 保持 `scripts/hooks/lib/` 现状引用兼容（KEEP/SKIP 特例仓不动）；若收敛到 `.githooks/lib/` 需同步更新 6 模板 source 路径，列入 P3 |
| R3: 38 仓批量渲染/提交耗时长、产生大量小提交 | 幂等增量（版本戳比对跳过未变仓）；`commit_with_submodules.py --apply` 串行处理；允许分批执行 |
| R4: 特例仓（task2app 已删除、dataMigrate 远程仓等）在清单内 | 分发器按现状 KEEP/SKIP 表处理；清单以 `.gitmodules` 为 SSOT 动态生成，消除硬编码漂移（顺带修复 db 副本差异） |
| R5: `.git/hooks/` 副本删除时若 hooksPath 未设置会瞬间失去门禁 | 迁移顺序严格「先设 hooksPath + 验证，再删副本」；P0/P2 阶段独立验收 |

## 11. 验收清单

- [ ] 主仓：`git config core.hooksPath` == `.githooks`；`.git/hooks/` 无非 sample 业务钩子；commit 触发门禁正常
- [ ] docs：hooksPath 已设置，pre-commit 归档钩子生效
- [ ] taskFE / taskReferral：`.githooks/` 入库且激活，`scripts/hooks/install.sh` 退役
- [ ] 全量子仓：`--check-hooks` 全绿（激活 ✅ / 入库 ✅ / 版本 ✅ / 无残留 ✅）
- [ ] db/scripts 副本已删除，`commit_with_submodules.py` 引用更新
- [ ] 新克隆演练：`git clone + install-hooks-all.sh` 后门禁生效
- [ ] 架构文件 v13 三类伴生 + VERSION_HISTORY 条目 + Archi 加载验证通过
