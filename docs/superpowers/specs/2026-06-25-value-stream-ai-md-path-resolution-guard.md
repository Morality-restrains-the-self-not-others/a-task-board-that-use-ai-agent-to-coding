# value-stream.yaml.ai.md 路径解析规则加固

- **状态**: 设计中
- **创建日期**: 2026-06-25
- **关联问题**: valueStream 启动报 `test_file not found: /tmp/ram-work/task2app/Saas_project/gitService/scripts/fix_oidc_ssl.sh`

## 问题根因

`oidc-ssl-protocol-fix` 价值流 4 个 step 的 `test_file` 缺少 `../../` 前缀：

| step | 错误路径 | 正确路径 |
|------|----------|----------|
| swd-url-builder-http-fix | `gitService/scripts/fix_oidc_ssl.sh` | `../../gitService/scripts/fix_oidc_ssl.sh` |
| oidc-playwright-diagnostic | `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` | `../../gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` |
| oidc-playwright-e2e-login | `gitService/playwright/tests/oidc-sso-login.playwright.test.js` | `../../gitService/playwright/tests/oidc-sso-login.playwright.test.js` |
| oidc-ssl-fix-verify | `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` | `../../gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` |

**为什么 `../../` 必须存在：** `runner.working_dir` = `../task2app/Saas_project`（相对 `conf/`），解析后为 `<repo_root>/task2app/Saas_project`。从该目录出发，`gitService/...` 解析到 `<repo_root>/task2app/Saas_project/gitService/...`（不存在），需要 `../../gitService/...` 向上两级到达 repo root 再进入 `gitService/`。

## 已有防护

`value-stream.yaml.ai.md`（v1.0.0, 2026-05-28）已覆盖三段式字段名、服务名白名单、port_config 映射等规则，但：
- `runner.working_dir` 文档表述有歧义（说"相对仓库根"，实际相对 config 文件目录）
- 缺少 `test_file` 路径解析的逐目录对照表
- 检查清单未显式要求验证路径可达

## 设计方案

### 改动范围：仅 `conf/value-stream.yaml.ai.md`

对现有文件做 4 处增量更新（非重写）：

#### 1. 新增「test_file 路径解析规则」章节（一级规则）

放在现有「step 与 test_file 约束」之后。内容包括：

- `runner.working_dir` 精确语义：`filepath.Join(ConfigDir, WorkingDir)`，即相对 **config 文件所在目录**（`conf/`）
- 当前值及解析结果：`../task2app/Saas_project` → `<repo_root>/task2app/Saas_project`
- 路径模式对照表（按 `test_file` 前缀分组）

#### 2. 更正 `runner.working_dir` 文档

将「相对仓库根」改为「相对 `conf/`（config 文件所在目录）」。

#### 3. 新增历史违规条目

在「历史违规条目」表中追加本次 2026-06-25 的 4 条修复记录。

#### 4. 更新检查清单

新增第 6 项：对照路径模式表确认 `test_file` 前缀正确

### 非目标

- Go 代码自动纠错提示
- CI/pre-commit 自动化脚本
- 修改 `runner.working_dir` 的值

## 实现步骤

1. `conf/value-stream.yaml.ai.md` 增量编辑（4 处）
2. 自检：确认所有现有 active step 的 test_file 解析正确
3. 更新变更日志

## 验证方式

- AI 修改 value-stream.yaml 后，对照路径模式表自检
- `go test ./src/ -run TestLoadProductionValueStream` 通过
