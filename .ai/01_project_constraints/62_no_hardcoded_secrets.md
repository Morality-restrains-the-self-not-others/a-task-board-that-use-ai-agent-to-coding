# 密钥禁止硬编码在源码中（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-27
- 维护者：Trae AI 团队
- 适用范围：整个 monorepo（meta + 全部子仓）第一方源码、脚本、测试与提交门禁
- 架构决策：ADR-0046
- 约束索引：第 57 条

## 核心原则

**任何密钥、口令、Token、私钥、客户端机密都必须存放在配置或密钥管理系统中，禁止作为字面量写进业务源码。**

允许的落点（按优先级）：

1. **`conf-local/<area>/<app>/`** — 与 `conf/` 同相对路径；机密与本机非机密覆盖都在这里；**已 gitignore，禁止提交**（第 58 条 / ADR-0054）
2. **环境变量** — 进程注入（`os.Getenv` / `os.environ` / `process.env`），值不进源码
3. **密钥管理系统 / KMS / CI secret store** — 生产级凭据

加载器不读 `conf/**/config.local.yaml`。gitignore 仍可忽略该文件名以防误提交。

禁止的落点：

- `.go` / `.py` / `.js` / `.ts` / `.vue` / `.sh` 等源码中的字符串字面量
- 提交到 Git 的 `.env` / `.env.*`（`.env.example` 等占位文件除外）
- 日志、错误响应、前端 DOM、commit message 中的明文密钥
- 为「方便跑通」把真实口令写进 Playwright / 调试脚本的默认值

本条管**写在哪**；一旦密钥已经进入已推送 Git 历史，处置走第 56 条（废弃 + 重新生成，禁止 filter-repo 清历史）。

## 硬约束（一级，禁止忽略）

### 禁止

- 在业务源码中硬编码 API key、access key、client secret、private key、password、webhook secret、internal secret、JWT secret
- 把私钥 PEM 块、云厂商 Access Key ID、GitHub PAT、Stripe `sk_live_`、Slack token 等粘进仓库
- `git add` `.env`、`.env.local`、`config.local.yaml` 或其它本机密钥文件
- Agent 为让测试变绿而把真实凭据写进源码或测试默认值
- 在日志中打印完整密钥 / Token（只打指纹或「redacted」）

### 允许

- 从 `confload` / 本服务 `config.yaml` 读取后注入
- `os.Getenv("DB_PASSWORD")` 等环境变量引用（源码中只有**名字**）
- 占位符：`changeme`、`test-api-key`、`${VAR}`、`$VAR`、`__PLACEHOLDER__`
- 测试夹具中的明显假值；高置信真实泄露模式（PEM、AKIA、`ghp_`、`sk_live_` 等）即使在测试里也禁止
- 公开客户端的非机密值：须同行或上一行注释 `Secret-Hardcode-OK: <原因>`（例如浏览器扩展 OAuth public client + PKCE）

### 测试凭据

自动化测试需要登录口令时：

- **只**从环境变量读取（`PLAYWRIGHT_TEST_PASSWORD` 等，见 `task2app/测试.ai.md`）
- **禁止** `process.env.X || '真实口令'` 把真实口令当默认值
- 缺环境变量时 skip 或失败，不要静默用仓库里的明文

## 触发

- 新增/修改认证、支付、云 SDK、OAuth、Webhook、数据库连接
- 编写 Playwright / 调试脚本 / `run_task` 类本地工具
- pre-commit 报 `VIOLATION (rule 62_no_hardcoded_secrets.md)`
- 发现仓库或日志中出现密钥字面量

## 验收标准

```bash
python3 db/scripts/ci/test_check_no_hardcoded_secrets.py
python3 db/scripts/ci/check_no_hardcoded_secrets.py
```

1. 自测全绿。
2. 第一方源码扫描退出码 0。
3. 在临时 Go 文件写入 `dbPassword = "hunter2-prod-like-credential"` 后扫描失败。
4. 同一值写在 `conf/<area>/<app>/config.yaml` 则通过。
5. 已 gitignore `.env`、`.env.*`（example/sample/template 除外）与 `conf/**/config.local.yaml`。

## 实现位置

| 组件 | 路径 |
|------|------|
| 约束专文 | `.ai/01_project_constraints/62_no_hardcoded_secrets.md` |
| Cursor 元规则 | `.cursor/rules/no-hardcoded-secrets.mdc` |
| ADR | `docs/adr/0046-no-hardcoded-secrets.md` |
| 门禁 | `db/scripts/ci/check_no_hardcoded_secrets.py` |
| 自测 | `db/scripts/ci/test_check_no_hardcoded_secrets.py` |
| 索引 | `.ai/01_project_constraints/00_project_constraints.md` 第 57 条 |

## 关联

- 第 42 条（conf SSOT）：非机密人工可改配置仍落 `conf/<area>/<app>/`；机密走 `conf-local/`
- 第 30 条（仅读本目录 conf）：进程不得直读他服务 conf 里的密钥块；须 sync；机密走本 app 的 `conf-local/`
- 第 56 条（禁止重写 Git 历史清密钥）：本条防再犯；泄露后走废弃 + 重新生成
- 第 58 条（conf-local）：本条管「不要写进源码」；机密值只放 `conf-local/`
- 第 17 条（支付 KYC）：支付密钥只进安全配置，禁止进卡数据自有库与日志
- Aliyun SDK：`.ai/03_technical_implementation/07_aliyun_sdk_usage.md` 已要求密钥来自环境或配置
