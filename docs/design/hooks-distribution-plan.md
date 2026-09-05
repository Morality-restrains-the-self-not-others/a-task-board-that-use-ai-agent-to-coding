# Git Hooks 模板分发到各服务目录

> 头脑风暴产物 — 设计文档
> 日期: 2026-06-05
> 状态: 待审批

---

## 现状

```
scripts/hooks/                      ← 集中模板源（本次要清理）
├── README.md
├── pre-commit-go                   ← 7 个 Go 仓库的源模板
├── pre-commit-django               ← gitOauth 的源模板
├── pre-commit-jest                 ← DaydaymoneyGrafana 的源模板
├── pre-commit-node                 ← taskSSE 的源模板
├── pre-commit-python-scripts       ← AiMonitor 的源模板
└── pre-commit-noop                 ← gitService + mock_run_container 的源模板

各服务已安装:
go_relayToTrae/scripts/hooks/pre-commit     (.git/hooks/pre-commit 已链接)
go_run_container/scripts/hooks/pre-commit   (.git/hooks/pre-commit 已链接)
runAll/scripts/hooks/pre-commit             (.git/hooks/pre-commit 已链接)
valueStream/scripts/hooks/pre-commit        (.git/hooks/pre-commit 已链接)
taskAuth/scripts/hooks/pre-commit           (.git/hooks/pre-commit 已链接)
taskBill/scripts/hooks/pre-commit           (.git/hooks/pre-commit 已链接)
taskEvents/scripts/hooks/pre-commit         (.git/hooks/pre-commit 已链接)
gitOauth/scripts/hooks/pre-commit           (.git/hooks/pre-commit 已链接)
DaydaymoneyGrafana/scripts/hooks/pre-commit       (.git/hooks/pre-commit 已链接)
taskSSE/scripts/hooks/pre-commit            (.git/hooks/pre-commit 已链接)
AiMonitor/scripts/hooks/pre-commit          (.git/hooks/pre-commit 已链接)
gitService/scripts/hooks/pre-commit         (.git/hooks/pre-commit 已链接)
mock_run_container/scripts/hooks/pre-commit (.git/hooks/pre-commit 已链接)
```

**关键事实**：所有 13 个仓库**已经**有本地 `scripts/hooks/pre-commit`。`scripts/hooks/` 中的文件只是当初 `cp` 后留下的源副本，已不再是任何仓库的运行时依赖。

---

## 设计方案

### 原则

**每个服务的 Git Hook 归该服务所有。** 模板源分散到各服务目录下，每个服务自行维护自己的 pre-commit 逻辑。

### 操作

#### Step 1：验证各服务 hooks 与源模板一致

对比每个仓库的 `scripts/hooks/pre-commit` 与 `scripts/hooks/pre-commit-{type}`，确认它们是最新的。

#### Step 2：删除集中模板源

删除 `scripts/hooks/pre-commit-{go,django,jest,node,python-scripts,noop}`（6 个文件）。

#### Step 3：迁移 README

`scripts/hooks/README.md` → `docs/reference/git-hooks.md`，内容更新为：
- 列出每个 hook 类型的**参考实现**所在仓库
- 安装说明指向各仓库的本地文件

#### Step 4：指定参考实现仓库

| Hook 类型 | 参考实现位置 | 使用者 |
|-----------|-------------|--------|
| Go | `runAll/scripts/hooks/pre-commit` | go_relayToTrae, go_run_container, runAll, valueStream, taskAuth, taskBill, taskEvents |
| Django | `gitOauth/scripts/hooks/pre-commit` | gitOauth |
| Jest | `DaydaymoneyGrafana/scripts/hooks/pre-commit` | DaydaymoneyGrafana |
| Node | `taskSSE/scripts/hooks/pre-commit` | taskSSE |
| Python Scripts | `AiMonitor/scripts/hooks/pre-commit` | AiMonitor |
| Noop | `gitService/scripts/hooks/pre-commit` | gitService, mock_run_container |

#### Step 5：更新各仓库 hooks README（可选）

在各参考实现仓库的 `scripts/hooks/` 下添加简短 README，说明此 hook 的用途和被哪些仓库引用。

---

## 变更汇总

| 仓库 | 新建 | 删除 | 修改 |
|------|------|------|------|
| scripts | — | `hooks/pre-commit-{go,django,jest,node,python-scripts,noop}` + `hooks/README.md` | — |
| docs | `reference/git-hooks.md` | — | — |
| runAll | — | — | 可选: `scripts/hooks/README.md` |
| gitOauth | — | — | 可选: `scripts/hooks/README.md` |
| ... | — | — | 可选 |

## 原则验证

| 检查点 | 结果 |
|--------|------|
| 每个 Hook 在所属服务目录下 | ✅ 已就位（13 仓库均有本地 copy） |
| 无集中模板源 | ✅ 删除 scripts/hooks/ 后即达成 |
| 其他仓库可参考 | ✅ 通过 docs/reference/git-hooks.md 查找 |
| 新仓库接入 | ✅ 从参考实现 cp + 按需修改 |

## 价值流影响

无。纯文件组织和文档变更。
