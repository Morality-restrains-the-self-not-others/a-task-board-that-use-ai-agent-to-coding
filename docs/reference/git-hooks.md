# Monorepo Git Hooks 参考

每个服务在自己的 `scripts/hooks/pre-commit` 下维护其 pre-commit 钩子，
已链接到 `.git/hooks/pre-commit`。

## 统一策略

所有钩子遵循相同策略：
1. **优先**运行与本次暂存文件相关的测试
2. **随机抽测**其余测试（默认 30%），若未命中则至少跑 1 个
3. **任一失败**则阻止提交

## Hook 类型与参考实现

| Hook 类型 | 参考实现（权威来源） | 使用者 |
|-----------|---------------------|--------|
| Go | `runAll/scripts/hooks/pre-commit` | go_relayToTrae, go_run_container, runAll, valueStream, taskAuth, taskBill, taskEvents |
| Django | `gitOauth/scripts/hooks/pre-commit` | gitOauth |
| Jest | `DaydaymoneyGrafana/scripts/hooks/pre-commit` | DaydaymoneyGrafana |
| Node | `taskSSE/scripts/hooks/pre-commit` | taskSSE |
| Python Scripts | `AiMonitor/scripts/hooks/pre-commit` | AiMonitor |
| Noop (占位) | `gitService/scripts/hooks/pre-commit` | gitService, mock_run_container |

## 安装到新仓库

```bash
# 以 Go 仓库为例
cd <new-go-repo>
mkdir -p scripts/hooks
cp ../runAll/scripts/hooks/pre-commit scripts/hooks/pre-commit
chmod +x scripts/hooks/pre-commit
cp scripts/hooks/pre-commit .git/hooks/pre-commit
```

或使用 `core.hooksPath`：

```bash
git config core.hooksPath scripts/hooks
```

## 环境变量

| 变量 | Hook 类型 | 说明 |
|------|----------|------|
| `PRECOMMIT_DJANGO_APP` | Django | Django app 标签，默认 `api` |
| `PRECOMMIT_PYTHON_TEST_DIR` | Python Scripts | 测试目录，默认 `scripts` |
| `PRECOMMIT_NODE_TEST_DIR` | Node | 测试目录，默认 `test` |
| `PRE_COMMIT` | 全部 | 标记 pre-commit 环境 |

## 已安装仓库

| 仓库 | 类型 | 机制 |
|------|------|------|
| go_relayToTrae | Go | `scripts/hooks/pre-commit` |
| go_run_container | Go | 同上 |
| runAll | Go | 同上 |
| valueStream | Go | 同上 |
| taskAuth | Go | 同上 |
| taskBill | Go | 同上 |
| taskEvents | Go | 同上 |
| gitOauth | Django | `scripts/hooks/pre-commit` |
| DaydaymoneyGrafana | Jest | `scripts/hooks/pre-commit` |
| taskSSE | Node | `scripts/hooks/pre-commit` |
| AiMonitor | Python Scripts | `scripts/hooks/pre-commit` |
| gitService | Noop | `scripts/hooks/pre-commit` |
| mock_run_container | Noop | `scripts/hooks/pre-commit` |
| task2app | Django + Playwright | `scripts/hooks/pre-commit`（独立维护，不在模板体系内） |
| trae-agent | Python | `.pre-commit-config.yaml` → `make uv-test`（独立维护） |
