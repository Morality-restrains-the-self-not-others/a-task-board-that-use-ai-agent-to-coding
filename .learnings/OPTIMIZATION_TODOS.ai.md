# OPTIMIZATION_TODOS.md AI 指令

## 文件体系

| 文件 | 用途 | 包含状态 |
|------|------|---------|
| `OPTIMIZATION_TODOS.md` | **本地可执行、无阻塞**的 pending 优化项 SSOT | `pending` |
| `BLOCK_TODO_BROWSER.md` | 分类：浏览器/公网验收（硬刷新、登录 Cookie、CDP）。**必须用 Playwright 当场闭环**，禁止等人目视 | `pending` + `Blocked-By: BROWSER` |
| `BLOCK_TODO_OPS.md` | 阻塞：运维与破坏性操作（清库、厂商门户重生模板、云 SG、重建容器、外部安装） | `pending` + `Blocked-By: OPS` |
| `BLOCK_TODO_INFRA.md` | 阻塞：基础设施或外部资产（腾讯云 COS、正式品牌物料） | `pending` + `Blocked-By: INFRA` |
| `PRODUCT_DECISIONS.md` | **需产品决策**的待办项 | `pending` → 产品确认后 `decided` → 移入 `OPTIMIZATION_TODOS.md` |
| `OPTIMIZATION_TODOS_EXTERNAL.md` | 对外部资源文件的索引（当前指向 PRODUCT_DECISIONS.md） | 索引 |
| `OPTIMIZATION_TODOS_COMPLETED.md` | **仅当日**已完成 / 已取消条目 | `completed` / `cancelled` |
| `archive/completed/OPT_COMPLETED_YYYY-MM-DD.md` | **按天归档**：非当日的所有历史条目 | `completed` / `cancelled` |

## 禁止人工验收（硬约束）

所有 OPT 条目（含 `OPTIMIZATION_TODOS.md`、`BLOCK_TODO_*.md`、完成后的 COMPLETED）的**完成判据必须是机器可复现的证据**。  
**禁止出现需要人工验收才能闭环的情况。**

产品拍板（尚未决定做什么）≠ 人工验收（做完了等人确认对不对）。前者走 `PRODUCT_DECISIONS.md`；后者一律禁止。

### 什么算人工验收（禁止写入 / 禁止作为完成条件）

| 禁止形态 | 示例 |
|---------|------|
| 请人打开浏览器看一眼 | 「需硬刷新公网目视确认」「等人打开 Chrome 验收」 |
| How to apply / Action 把人当断言器 | 「人工确认」「目视」「手验」「请用户点一遍」「登录后肉眼检查」 |
| Completion-Note 只有人话、无机器证据 | 「用户确认已验收」「看起来对了」 |
| Status 伪装成等人 | `pending（待人工）` / `pending（待目视）` / `pending（待用户确认 UI）` |
| 把 BROWSER 停放写成等人 | 「等人类打开浏览器后再标 completed」 |
| 无法写成断言的体验项 | 「页面好不好看」「交互是否顺手」且没有 `data-testid`/文案/DOM/截图像素可断言 |

### 必须改成什么

每条 OPT 的 `How to apply` **必须**给出机器验收步骤（至少一项），否则**不录入**：

| 场景 | 机器验收（写入 How to apply） |
|------|------------------------------|
| 代码/逻辑 | 点名 `go test` / vitest / pytest 测例路径，exit 0 |
| 公网 UI / SPA / Cookie | Playwright：URL + 选择器/`data-testid` + `expect(...)`；或已有 `*.playwright.test.js` |
| HTTP / 配置 | `curl`/脚本 + 退出码或 JSON/响应断言 |
| 日志 | 可复现的检索命令（本地日志或 Loki 查询）及期望命中/不命中 |
| 构建/发布 | 点名命令 + 产物路径或 HTTP 200 断言 |

`completed` 的 `Completion-Note` **必须**引用上述证据（测例名、Playwright locator 结果、命令输出摘要）。禁止只写「已验收」。

写不出机器断言的事项：**禁止写入任何 OPT 清单**（含 `BLOCK_TODO_*`）。不要用「先停放等人验收」凑条目；应改为补测例/Playwright，或向用户说明后放弃录入。

## 产品决策分流规则

需产品拍板的二期功能、UX 决策、业务规则等（**尚未决定做什么**），**必须**写入 `PRODUCT_DECISIONS.md`，**禁止**写入 `OPTIMIZATION_TODOS.md`：

- 产品决策项 Status 为 `pending`，等待产品在合适时机逐一拍板（这是决策，不是功能验收）
- 产品确认后改 Status 为 `decided`，在 Details 追加决策结论
- 决策完成后将条目移至 `OPTIMIZATION_TODOS.md` 执行（若落地已完成则直接迁入 `OPTIMIZATION_TODOS_COMPLETED.md`）
- `PRODUCT_DECISIONS.md` **不得**长期堆积 `decided`/`accepted` 项——决策即分流

## 阻塞项分流（项目元规则）

`OPTIMIZATION_TODOS.md` **只允许**无需浏览器/运维窗口/新基础设施即可闭环的 pending。  
若条目存在阻塞，**必须**按类别移入 `BLOCK_TODO_<CATEGORY>.md`，禁止留在开放清单里用「队列索引分组」假装分流。  
`BROWSER` 进 block 文件后**仍须本会话用 Playwright 验收**（见「BROWSER 阻塞项的 Playwright 解除」），不是永久停放。

| 阻塞类别 | 目标文件 | 典型阻塞 |
|---------|---------|---------|
| `BROWSER` | `BLOCK_TODO_BROWSER.md` | 公网硬刷新、登录 Cookie、CDP 9222、DOM/`data-testid` 断言 → **必须用 Playwright 解除**（禁止目视） |
| `OPS` | `BLOCK_TODO_OPS.md` | 清库/初始化、厂商门户重生 UserData、云安全组、重建容器、外部 App 安装 |
| `INFRA` | `BLOCK_TODO_INFRA.md` | 需新组件（腾讯云 COS）或外部资产（正式品牌物料） |

分类实现（新增/迁移时必须走同一函数，禁止凭感觉）：`.learnings/_opt_queue.py` → `classify_opt(title, body)`。

### 写入时判定

1. How to apply **写不出机器验收步骤** → **不录入**（禁止改成「待人工验收」）
2. 需要产品拍板 → `PRODUCT_DECISIONS.md`（**不是** BLOCK_TODO，也**不是**验收）
3. `classify_opt` 返回 `BROWSER` / `OPS` / `INFRA` → 对应 `BLOCK_TODO_*.md`，并补 `- **Blocked-By**: <CATEGORY>`
4. 返回 `OPEN` → `OPTIMIZATION_TODOS.md`
5. 编号仍用 `python3 .learnings/_move_opt.py --next-id`（扫描含 `BLOCK_TODO_*.md`）

### 阻塞解除

按类别处理，**不要**把 `Blocked-By: BROWSER` 当成「等人类打开浏览器」：

| 类别 | 解除方式 |
|------|---------|
| `BROWSER` | **本会话用 Playwright 验收**（下一节）。通过后直接 `completed` 迁入 `OPTIMIZATION_TODOS_COMPLETED.md`，不必先移回开放清单。 |
| `OPS` / `INFRA` | 真实阻塞消失后（清库窗口已批准、COS 桶/密钥已配置等）移回 `OPTIMIZATION_TODOS.md` 再执行；完成后仍须脚本/测例证据，**禁止**改成「请运维目视确认」。本会话已完成则直接 `completed`（Completion-Note 带命令/测例证据）。 |

### BROWSER 阻塞项的 Playwright 解除

看到 `- **Blocked-By**: BROWSER` 时，**唯一验收手段是 Playwright**，禁止跳过、禁止改成人工硬刷新/目视。分类仍进 `BLOCK_TODO_BROWSER.md`（避免 50+ 条公网验收淹没开放清单）；执行权在本会话。

#### 可闭环 vs 前置缺失（停放 ≠ 等人验收）

左列必须当场 Playwright 闭环。右列是**本会话跑不了 Playwright 的前置缺失**，不是「改请人看一眼」：下次仍走 Playwright。禁止把右列写成人工验收。

| 必须用 Playwright 当场闭环 | 前置缺失时的处理（仍禁止人工验收） |
|---------------------------|-----------------------------------|
| 打开 `How to apply` 公网 URL、硬刷新、断言 `data-testid` / 文案 / DOM 顺序 / Teleport 槽 | 真实第三方同意屏无法自动化点击 → **不写 OPT**；用 mock/回调测例覆盖授权后路径。禁止「请用户走一遍 OAuth」 |
| 登录后检查主内容非空、滚动、Tab、开关持久化、错误节点 `data-traceId` | 所需账号不存在（如 ≥2 公司 bootstrap-admin）→ 先跑 seed 脚本创建；不可安全创建才停放并记证据，下次仍 Playwright |
| 条目已点名 `*.playwright.test.js` / `playwright.config.cdp.js` | 登录凭据确认失效且无替代测号 → 记证据后停放；禁止改成「请用户登录后目视」 |
| 确认公网 SPA 已吃到新 dist（选择器/文案与本地修复一致） | 须停生产服务、清库、改云 SG → 改走 `OPS`，解除后仍用脚本/Playwright 验收，不目视 |

#### 执行顺序

1. **已有测例**：Action / How to apply 点名了 `taskFE/tests/*.playwright.test.js` → 直接跑该文件。
2. **无测例**：按条目写一次性 Playwright 验收（优先复用登录 helper）。**禁止**先等人类硬刷新或把「目视确认」写进 How to apply。
3. **前置**：若 Action 要求 9999「精准编译重启」，先确认目标服务已发布/已登记重启；未发布则先发布再验，仍算本会话可执行。
4. **通过**：`python3 .learnings/_move_opt.py <ID> completed '<Playwright 证据：测例或选择器 + 结果>'`。
5. **失败**：留在 `BLOCK_TODO_BROWSER.md`，在 Context 追加失败证据（截图路径、URL、实际 DOM、traceId）；禁止无证据改 `completed`。

#### 技术入口（按优先级）

1. **CDP 9222（推荐）**：外部 Chrome 已 `--remote-debugging-port=9222`（参考仓库根 `scripts/runDebugChrome.sh`；Linux 可加 `--no-sandbox`）。测例侧 `connectOverCDP('http://127.0.0.1:9222')` 或：

   ```bash
   cd taskFE
   PLAYWRIGHT_SITE_ORIGIN=https://www.daydaymoney.com \
     npx playwright test --config=playwright.config.cdp.js tests/<已有或新建>.playwright.test.js
   ```

2. **登录**：复用 `taskFE/tests/playwrightLogin.js` 的 `playwrightLoginWithLegalAccept`，或 `taskFE/tests/helpers/gatewayLoginE2e.js` 的 `loginViaGatewayApi`（密码明文，与 Login.vue 一致）。凭据用 `PLAYWRIGHT_TEST_EMAIL` / `PLAYWRIGHT_TEST_PASSWORD`（或 `E2E_PASSWORD`），**禁止**把密码写进 OPT 正文。
3. **公网 origin**：`PLAYWRIGHT_SITE_ORIGIN` 或 `SITE_BASE` = `https://www.daydaymoney.com`（`readE2eOrigins()` 已读这些变量）。
4. **硬刷新**：`page.goto(url)` 后 `page.reload({ waitUntil: 'networkidle' })`；断言前等到目标选择器可见（动态 SPA，禁止首屏未就绪就猜 CSS）。优先 `data-testid` 与条目 How to apply 给出的选择器。
5. **规则 SSOT**：`.ai/05_testing_quality/01_core_testing_rules.md`（CDP 9222、侦察后再断言）。

一次性脚本也可：`chromium.connectOverCDP` → 登录 → goto 条目 URL → reload → `expect(locator).toBeVisible()` / 文案 / 计算样式。有稳定回归价值时再落成 `taskFE/tests/*.playwright.test.js`。

#### 与 `classify_opt` 的关系

- 标题含「公网验收 / 公网硬刷新」等仍分类为 `BROWSER`，写入 `BLOCK_TODO_BROWSER.md`。
- `_OPEN_OVERRIDE_MARKERS` 的「新增 playwright / playwright：」用于编写测例本身的 OPEN 项。
- **用户明确要求**将「Playwright 可闭环」项移入开放清单时：去掉 `Blocked-By`，Status 改为 `pending`（禁止残留「待浏览器」），How to apply 以 `Playwright：` 开头（走 override），`BLOCK_TODO_BROWSER.md` 只留前置缺失停放项。
- 未经用户明确要求，不要为了「能跑 Playwright」把公网验收改写成 OPEN。Playwright 默认仍是 BROWSER 的解除手段。

### 禁止

- **禁止**把阻塞项留在 `OPTIMIZATION_TODOS.md`（含「pending（待浏览器）」这种带括号的假可执行）
- **禁止**任何形式的人工验收：目视、手验、请用户点一遍、Completion-Note 只写「用户确认已验收」
- **禁止** How to apply / Action 缺少机器断言步骤仍写入 OPT
- **禁止**看到 `Blocked-By: BROWSER` 就跳过不验或改成等人硬刷新（须先走 Playwright；仅上表「前置缺失」可留下，且下次仍 Playwright）
- **禁止**把无法自动化的第三方同意屏 / 「体验好不好」写成 OPT 等人走一遍
- **禁止**把无需浏览器的本地项（纯代码/单测/文档）写入 `BLOCK_TODO_*.md`
- **禁止**把产品决策写入 `BLOCK_TODO_*.md`（走 `PRODUCT_DECISIONS.md`）
- **禁止**只改队列索引表、不移动正文
- **禁止**未跑 Playwright、无 DOM/测例证据就把 BROWSER 项标 `completed`

## 外部资源与不可执行项

以下情况**不应作为 OPT 条目追踪**，因为无法在当前环境中自动完成。若已有此类条目，应标记 `cancelled` 归档：

| 资源类型 | 说明 |
|---------|------|
| 公网容器滚动（远程） | 需操作远程 ECS/容器实例（**注意：本地的 `runAll` 编译+重启是本地可执行的，不属此类**） |
| DNS 操作 | 需 DNS API 凭证或 root 权限 |
| 远程服务器 | 需 SSH 登录远程主机 |
| 外部服务恢复 | 需操作外部服务或其凭证 |
| Git 远程操作 | 需远端服务可达 |

> 以上资源类别的条目若已存在，应标记 `cancelled` 归档。新增条目若涉及以上类别，**禁止**写入任何 OPT 清单。
>
> ⚠️ **关键区分**：「部署」≠ 一定是远程操作。本项目的 `runAll` 服务运行在本地，可执行编译 (`build-all`)、重启 (`/api/restart-all`)、SPA 构建发布 (`runall-lifecycle.sh build`)。这些**都是本地可执行操作**，对应的 OPT 条目应写入 `OPTIMIZATION_TODOS.md`。只有需要 SSH 登录远程主机、操作远程 ECS/容器实例的部署才属于不可执行的外部资源。
>
> 需要部署时，优先通过 runAll 完成 — 详见下方「服务重新编译启动（runAll）」章节。

**仅以下情况可写入 `OPTIMIZATION_TODOS.md`**（本地可执行，且 How to apply 含机器验收）：
- 纯本地代码修改（Go/Python/JS/Vue 等，可 `go build` / `npm run build` / 对应单测验证）
- 纯本地文档修改
- 纯本地测试（`go test`、vitest、pytest、编写 Playwright CDP E2E 测例）
- 公网/Cookie/硬刷新类验收写在 `BLOCK_TODO_BROWSER.md`，用 Playwright 执行后直接 `completed`（不要塞回本文件）
- 纯本地配置修改（`conf/` YAML、`sync.manifest.yaml`）
- **服务编译部署**（`./bin/runAll -command build-all`、`POST /api/restart-all`、`runall-lifecycle.sh build`）
- **Docker 镜像推送**（`DOCKER_PUSH=1 ./buildDocker.sh` — 仅需在对应目录执行此命令，无需额外 registry 凭证）
- **数据迁移/回填**（本地 DB 批量扫描/修复/回填 — 当前环境即为生产环境，直接执行迁移脚本）
- SPA 构建发布（`runall-lifecycle.sh build` 含 Vite build + collectstatic + rsync）

## 写入 OPT 时的分流规则

当智能体在会话收尾追加新 OPT 条目时，**必须**：

1. 判断 How to apply 能否写成机器验收（测例 / Playwright / curl / 脚本）→ **不能** → **不录入**（禁止改成待人工验收）
2. 判断是否需要产品决策 → 若需要 → 写入 `PRODUCT_DECISIONS.md`，Status 为 `pending`
3. 判断是否为不可执行的外部资源（对照上表） → 若是 → **不录入** OPT，改为向用户说明
4. 用 `classify_opt` 判断是否阻塞 → `BROWSER`/`OPS`/`INFRA` → 写入对应 `BLOCK_TODO_*.md`（BROWSER 的验收仍是 Playwright，不是人）
5. 若**本地可执行且无阻塞** → 写入 `OPTIMIZATION_TODOS.md`，Status 为 `pending`
6. 编号保持统一序列（`OPT-YYYYMMDD-NNN`），全局递增，跨文件不重复（含 `BLOCK_TODO_*.md`）

## 强制迁移流程（完成/取消时）

当智能体完成一条 `OPT-*` 项（Status 变为 `completed` 或 `cancelled`）时，**必须**：

### 1. 标记完成并迁移到归档文件

无论条目来自 `OPTIMIZATION_TODOS.md` 还是 `PRODUCT_DECISIONS.md`，完成后统一执行以下步骤：

1. **原地标记**：在源文件中将条目状态更新为 `completed`，补充完成时间和说明：
   ```markdown
   ## [OPT-YYYYMMDD-NNN] completed

   **Status**: completed
   **Completed**: <ISO-8601 完成时间>
   **Completion-Note**: <机器证据：测例路径 / Playwright locator 结果 / 命令与退出码；禁止只写「用户确认已验收」>
   ```
2. **读取归档文件**：读取 `OPTIMIZATION_TODOS_COMPLETED.md`，确认无重复编号
3. **追加到归档文件**：将该条目完整追加到 `OPTIMIZATION_TODOS_COMPLETED.md` 末尾
4. **从源文件删除**：从源文件中删除该条目
5. **触发归档检查**：追加完成后，扫描 `OPTIMIZATION_TODOS_COMPLETED.md` 是否存在非当日的条目，若有则按「归档规整」流程执行按天归档

### 2. 产品决策确认后

若 `PRODUCT_DECISIONS.md` 中的条目经产品确认：
- 在其 Details 中追加决策结论
- 改 Status 为 `decided`
- 将其从 `PRODUCT_DECISIONS.md` 移至 `OPTIMIZATION_TODOS.md` 执行

## 禁止行为

- **禁止**将需要产品决策的条目写入 `OPTIMIZATION_TODOS.md`
- **禁止**将本地可执行的条目写入 `PRODUCT_DECISIONS.md`
- **禁止**写入或完成依赖人工验收的条目（目视、手验、请用户确认 UI、无机器证据的「已验收」）
- **禁止** How to apply 不给出机器断言步骤
- **禁止**将不可执行的外部资源条目（DNS/remote-ops/git-remote 等）录入任何 OPT 清单
- **禁止**只改 Status 不迁移：`completed` / `cancelled` 条目不得留在 pending 清单中
- **禁止**删除历史条目而不归档
- **禁止**修改条目正文内容：迁移时保留原条目所有字段不变
- **禁止**让非当日的条目停留在 `OPTIMIZATION_TODOS_COMPLETED.md` 中
- **禁止**删除或修改已写入按天归档文件中的条目

## 归档规整（防止 `OPTIMIZATION_TODOS_COMPLETED.md` 无限膨胀）

### 设计原理

`OPTIMIZATION_TODOS_COMPLETED.md` **仅保留当日**完成的条目，作为当日工作上下文供 AI 参考。非当日的条目全部按天归入 `archive/completed/OPT_COMPLETED_YYYY-MM-DD.md`，既保留完整历史审计线索，又彻底杜绝主文件膨胀。

### 归档判定标准

- **留存条件**：`Completed` 日期 = **今天**的条目保留在主文件
- **归档条件**：`Completed` 日期 ≠ **今天**的条目归入按天归档
- **无 Completed 字段**：以 `Logged` 日期代替判定
- **cancelled 条目**：与 completed 同等对待（按 Completed/Logged 日期归档）

### 按天归档文件格式

```
archive/completed/
├── OPT_COMPLETED_2026-07-17.md
├── OPT_COMPLETED_2026-07-18.md
├── OPT_COMPLETED_2026-07-19.md
└── ...
```

每个按天归档文件格式：

```markdown
# Completed OPT Archive — 2026-07-17

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 N 条。
> 归档时间：<ISO-8601>

（条目按原顺序排列，格式与主文件一致）
```

### 归档执行流程

当智能体完成「强制迁移流程」步骤 5（触发归档检查）时，执行以下步骤：

1. **扫描非当日条目**：遍历 `OPTIMIZATION_TODOS_COMPLETED.md` 中所有 `## [OPT-*]` 条目，提取每个条目的 `**Completed**:` 或 `**Logged**:` 日期，筛选出非当日的条目
2. **按天分组**：将非当日条目按 `YYYY-MM-DD` 分组
3. **写入按天归档**：
   - 若目标 `archive/completed/OPT_COMPLETED_YYYY-MM-DD.md` 不存在 → 创建（含文件头）
   - 若已存在 → 在该文件已有条目之后追加（去重：跳过已存在的编号）
4. **从主文件删除**：从 `OPTIMIZATION_TODOS_COMPLETED.md` 中移除已归档条目
5. **更新主文件索引**：在 `OPTIMIZATION_TODOS_COMPLETED.md` 文件头维护归档索引：

```markdown
<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->
<!-- 2026-07-17: 6 条 → [archive/completed/OPT_COMPLETED_2026-07-17.md](./archive/completed/OPT_COMPLETED_2026-07-17.md) -->
<!-- 2026-07-18: 107 条 → [...] -->
...
```

### 批量归档（存量清理）

对于历史上已累积大量条目的 `OPTIMIZATION_TODOS_COMPLETED.md`，可执行一次性批量归档：

```bash
# 将非当日的条目按天批量迁移到 archive/completed/
python .learnings/_archive_completed_opts.py --dry-run    # 预览
python .learnings/_archive_completed_opts.py               # 执行
```

该脚本也可被 AI 在会话中调用（通过 Bash 工具），对存量文件做批量规整。

### 归档完整性约束

- **禁止**删除按天归档文件中的条目（历史不可变）
- **禁止**将已归档条目重新移回主文件
- 按天归档文件条目之间用**一个空行**分隔，格式与主文件完全一致
- 归档索引注释行以 `<!--` 开头，不计入条目计数

## 格式要求

- 四个文件（含按天归档）使用相同的条目格式（编号规则、字段结构见 `25_session_end_optimization_todo.md`）
- 条目之间用**一个空行**分隔
- 编号 `OPT-YYYYMMDD-NNN` 全局唯一，跨所有文件不复用
- `How to apply` **必须**含机器验收步骤（测例 / Playwright `expect` / curl 或脚本断言）；禁止「人工确认」「目视」
- `Completion-Note` **必须**引用机器证据，禁止只写「用户确认已验收」

## 与现有规范的关系

- 本文件是 `.ai/01_project_constraints/25_session_end_optimization_todo.md` 的**自动化执行补充**
- 该约束文件第 102 行将迁移表述为「推荐」；本 `.ai.md` 将其升级为**强制**
- 该约束「验收要点」在本文件中**收紧为机器可复现证据**；人工/目视验收一律禁止
- 三文件 + 按天归档是本文件的扩展：在原 `pending ↔ completed` 基础上增加 `pending-local ↔ pending-external` 维度和 `completed-today ↔ completed-archive` 时间维度

## 服务重新编译启动（runAll）

本项目的所有后台服务（Go 服务、Python/Django、Node.js SSR 等）的**重新编译和启动**统一由 `runAll` 编排，**不需要**逐个进入各服务目录手动执行 `go build` / `npm run build` / `python manage.py runserver` 等操作。

### 编译

```bash
cd runAll
./bin/runAll -command build-all
```

该命令同步编译所有托管服务（复用 `Runner.BuildAll`），失败或部分编译失败时 `exit 1`。

编译前置：`./build.sh`（含 skip-orphan 能力标记校验）。

### 重启

- **全量重启**：UI 按钮「全部重启」或 `POST /api/restart-all`（需 `session_id`），内部先 `stop-all` 再 `start-all`，进度通过 SSE `/api/restart-all/progress` 推送
- **单服务启停**：UI 各服务行按钮，或对应 API

### 热替换

经 `/api/shutdown-self` 热替换 runAll 自身时，**必须**使用 `./build.sh` 编译的含 skip-orphan 标记的新二进制，旧二进制会把仍在跑的托管服务误当孤儿 `SIGKILL`。

### 部署（SPA + 静态资源 + collectstatic）

```bash
cd taskFE/app
bash scripts/runall-lifecycle.sh build
```

该脚本依次执行：Vite build → Django collectstatic → rsync 强制同步 Vite assets 到 STATIC_ROOT。

### 数据迁移/回填

**当前环境即为生产环境**，本地 DB 文件（`data/*.db`、`data/*.sqlite3`）就是生产数据库。所有数据迁移脚本、批量修复脚本、历史数据回填均应**直接执行**，无需等待"切换到生产环境"。

常用路径：
- `db/scripts/` — 数据库迁移脚本（如 `migrate_feature_params_to_task_cloud.sh`）
- `data/task_cloud.db`、`data/task_task.db`、`data/task_project.db` 等 — SQLite 业务库
- `task2app/Saas_project/saas.sqlite3` — Django saas 库
- `data/corrupt-backup/` — 数据修复/恢复脚本

### 对 OPT 分流的影响

- **服务编译/重启/SPA 部署** → 本地可执行 → `OPTIMIZATION_TODOS.md`
- **Docker 镜像推送**（`DOCKER_PUSH=1 ./buildDocker.sh`） → 本地可执行 → `OPTIMIZATION_TODOS.md`
- **数据迁移/回填** → 本地可执行（当前即生产） → `OPTIMIZATION_TODOS.md`
- **公网容器滚动**（重启远程任务容器/ECS） → 需远程操作 → `OPTIMIZATION_TODOS_EXTERNAL.md`
