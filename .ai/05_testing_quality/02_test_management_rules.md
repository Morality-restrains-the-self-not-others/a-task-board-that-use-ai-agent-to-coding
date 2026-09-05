# 测试管理规则

## 基本信息
- 版本：2.5.2
- 创建日期：2026-01-27
- 最后修改：2026-04-29
- 维护者：Trae AI 团队

## 规则分类

### 核心规则

#### 测试意图伴随文档规则
- 描述：每个测试文件都必须配套一个同名的测试意图文件，使用自然语言说明该测试“要守住的行为边界”，用于后续回归和破坏定位。
- 适用场景：所有新建或修改测试文件（单元测试、集成测试、端到端测试、Playwright 测试等）。
- 优先级：高
- 规则：
  - **同名规则**：测试文件保存后，必须在同目录创建或更新 `${testFileName}.testIntent` 文件。
  - **命名示例**：
    - `test_cloud_get_vpcs.py` 对应 `test_cloud_get_vpcs.py.testIntent`
    - `TaskDetail.layer-agent.playwright.test.js` 对应 `TaskDetail.layer-agent.playwright.test.js.testIntent`
  - **内容要求（自然语言）**：至少写清楚测试目标、关键业务断言、典型破坏信号、回归修复时的验证要点。
  - **模板建议**：优先使用 `docs/intents/00_索引/test_file_intent_template.testIntent` 作为编写模板，确保团队结构一致。
  - **维护要求**：当测试代码发生语义变更时，必须同步更新对应 `.testIntent`；禁止只改测试代码不改测试意图。

#### 测试结果存储
- 描述：规范测试结果的存储和管理
- 适用场景：所有测试结果管理
- 优先级：高
- 规则：
  - 所有测试结果（包括单元测试、集成测试、前端自动化测试等）必须放置在专门的目录中，避免与项目其他文件混合
  - 单元测试结果目录：`{项目根目录}/tests/test_results/unit_tests/`
  - 集成测试结果目录：`{项目根目录}/tests/test_results/integration_tests/`
  - 前端自动化测试结果目录：`{项目根目录}/tests/test_results/frontend_tests/`
  - 测试报告目录：`{项目根目录}/tests/test_results/reports/`
  - 测试日志目录：`{项目根目录}/tests/test_results/logs/`
  - 每次运行测试前，应清理对应目录的旧测试结果，确保测试结果的准确性和完整性

#### 授权问题排查
- 描述：规范授权问题的排查流程
- 适用场景：所有测试中的授权问题排查
- 优先级：高
- 规则：
  - 当测试目标不是测试用户授权问题，但出现授权相关错误时，首先检查是否已注册并登录测试用户
  - 确保测试环境中存在有效的测试账号，并在测试开始前完成登录操作
  - 对于API测试，检查是否正确传递了认证信息（如token、session等）

#### 测试身份模拟规则
- 描述：规范测试身份的模拟
- 适用场景：所有需要模拟用户身份的测试
- 优先级：高
- 规则：
  - 除非测试目标明确是测试登录鉴权功能，否则所有测试都应该模拟已登录用户进行
  - 对于API测试，应该在测试用例中正确设置认证信息（如使用force_authenticate或提供有效的token）
  - 对于前端测试，应该在测试开始前完成登录操作，或者使用已登录的会话
  - 确保测试环境中存在用于模拟登录的有效测试账号

#### 测试鉴权用户创建规则
- 描述：规范测试鉴权用户的创建和管理
- 适用场景：所有需要鉴权用户的测试
- 优先级：高
- 规则：
  - 当测试需要鉴权登录时，应该在测试用例的setup阶段创建一个新的测试用户
  - 使用新创建的用户进行登录和后续的测试操作，确保测试的隔离性和可重复性
  - 测试完成后，在teardown阶段清理创建的测试用户，避免测试数据残留
  - 对于API测试，使用新创建用户的认证信息进行请求
  - 对于前端测试，使用新创建的用户账号密码进行登录

#### Playwright 测试环境变量登录规则
- 描述：使用 Playwright 测试时，可通过环境变量提供账号与 GitHub 相关凭证，避免硬编码测试账号信息，提高测试的灵活性与安全性。根目录 [`测试.ai.md`](../../测试.ai.md) 为同名约定速查。
- 适用场景：所有 Playwright 测试中需要登录账号或 GitHub 相关操作的场景
- 优先级：高
- 规则：
  - **通用租户/应用登录**：测试代码必须从 `PLAYWRIGHT_TEST_EMAIL`、`PLAYWRIGHT_TEST_PASSWORD` 读取口令；邮箱未设置时可用文档约定的测试身份（租户 contact@daydaymoney.com 或管理员 author@example.com）。**禁止**把口令写成源码或 `env || '明文'` 默认值（约束第 57 条）
  - **GitHub 绑定或 OAuth 相关用例**：应优先使用 `PLAYWRIGHT_TEST_GITHUB_EMAIL`、`PLAYWRIGHT_TEST_GITHUB_PASSWORD`；若某用例仅需要应用内账号，仍以通用 `PLAYWRIGHT_TEST_EMAIL` / `PLAYWRIGHT_TEST_PASSWORD` 为准，按需选用 GitHub 专用变量，避免不必要混用
  - 禁止在测试代码中硬编码账号密码；应通过环境变量或 CI Secrets 注入
  - 环境变量设置示例：
    - `export PLAYWRIGHT_TEST_EMAIL=your_email@example.com`
    - `export PLAYWRIGHT_TEST_PASSWORD=your_password`
    - `export PLAYWRIGHT_TEST_GITHUB_EMAIL=your_github_account@example.com`
    - `export PLAYWRIGHT_TEST_GITHUB_PASSWORD=your_github_password`
  - 在 CI/CD 中通过平台的环境变量或密钥管理配置上述值，禁止将真实口令提交到仓库

#### 测试短信号码目标规则
- 描述：凡测试流程会触发真实或沙箱短信投递、需在用例中填写收件手机号的，号码必须取自 `conf/port_config.test.json` 的 `django.test_phone_numbers`，不得使用列表外的号码；细则与实现参考见 [核心测试规则](./01_core_testing_rules.md) 中的「测试短信号码目标规则」及 `Saas_project/tests/test_phone_config.py`。
- 适用场景：Playwright/接口测试中填写手机号、充值绑定等短信相关用例
- 优先级：高

### 最佳实践
&gt; 提升测试效率和质量的建议

#### 业务流程测试
- 描述：规范业务流程测试的实施
- 适用场景：所有业务流程测试
- 优先级：中
- 规则：
  - 每个业务流程需有对应的 wsd 流程文件和测试用例，文件名与用例名称一致

#### 前端界面测试
- 描述：规范前端界面测试的实施
- 适用场景：所有前端界面测试
- 优先级：中
- 规则：
  - 开发完成后，对页面设计进行 review，确保符合苹果人机交互界面设计规范
  - 使用 playwright 和 chrome devtools mcp 进行浏览器自动化测试分析调试
  - 使用 chrome devtools mcp 调试时禁用密码管理器及泄露检测
  - 使用 playwright 测试时，必须添加超时时间设置，以避免无限悬挂
  - **Playwright 浏览器可见性**：本地与智能体执行测试时应显示浏览器窗口以便观察与调试；`playwright/front_project/playwright.config.js` 使用 `use.headless: !!process.env.CI` 约定「未设置 `CI` 时有头、设置 `CI` 时无头」。需要额外强制有头时可加 CLI 参数 `--headed`。禁止在规则或文档中要求「默认必须无头」。
  - playwright 测试超时设置建议：
    - 全局超时：使用 `--timeout` 参数设置全局测试超时时间
    - 示例命令：`npx playwright test -c front_project/playwright.config.js --timeout=60000`（在 `task2app/playwright/` 下执行，与上述 config 一致时本地默认有头；或在 `front_project/app/` 下 `npm run test:e2e`）
  - 端到端测试文件命名和存放规则：
    - 主站 playwright 测试脚本必须放在 `playwright/front_project/tests/` 目录下，与业务源码（`front_project/app/src/`）分离；依赖由 `task2app/playwright/package.json` 统一管理
    - 前端应用根目录仍为 `front_project/app/`（Vite/Vue）；E2E 工程根为 `task2app/playwright/`
    - 命名规则：`{测试对象}.{测试意图}.playwright.test.js`
      - 组件级测试：`{组件名}.{测试意图}.playwright.test.js`（如 `WorkPanel.登录后已安装镜像API不401.playwright.test.js`）
      - 流程级测试：`{流程简述}.playwright.test.js`（如 `login-then-installed-images.playwright.test.js`）
    - 主站 playwright.config.js 位于 `playwright/front_project/`，需设置 `testDir: './tests'`
  - 测试结果放在 `playwright/front_project/tests/test_results/`，与 testDir 保持一致
  - 测试完成检测：当 playwright 测试输出中出现 "Serving HTML report at" 时，说明测试已经运行完成，应该自动跳出该测试
  - 登录模块独立：为避免测试时卡在登录环节，应将登录模块独立出来作为单独的测试工具或函数
  - 登录账号使用：使用以下测试账号进行登录测试：
    - 租户测试账号：contact@daydaymoney.com/rgNodkdq8677!ci
    - 管理员测试账号：author@example.com/rgNodkdq8677!ci
    - 非超管 E2E 专用账号：e2e.nonadmin.sysadmin.redirect@ljytest.com/E2eNonAdmin!8677（**禁止删除**；由 `ensureE2eNonAdminAccount.mjs` 幂等保障）
  - 登录状态复用：在测试中实现登录状态的复用机制，避免每次测试都重新登录，提高测试效率
  - 登录失败处理：添加登录失败的重试机制和错误处理，确保测试的稳定性
  - 测试交互规范：使用 Playwright 撰写测试时，不可以直接调用 API，而是只能采用模拟点击的方式，以确保功能是前端实现的
    - 禁止使用 `page.request` 或 `context.request` 直接调用 API
    - 必须通过模拟用户操作（如点击按钮、填写表单等）来触发前端功能
    - 测试应该验证前端的实际行为和状态变化，而不是直接测试 API 响应

#### 功能修改测试要求
- 描述：规范功能修改后的测试要求
- 适用场景：所有功能修改后的测试
- 优先级：中
- 规则：
  - 功能修改后，和修改点有关的所有功能都需要通过 playwright 测试，确保功能完整性和回归正确性

#### 测试环境
- 描述：规范测试环境的配置和管理
- 适用场景：所有测试环境配置
- 优先级：中
- 规则：
  - 测试环境的启动必须通过项目根目录下的 `run.sh` 脚本进行
  - 运行测试前需在仓库根目录执行 `source activate_env.sh unit`，统一激活单元测试环境

#### 测试结果反馈
- 描述：规范测试结果的反馈流程
- 适用场景：所有测试结果反馈
- 优先级：中
- 规则：
  - 运行单元测试后，必须更新 `docs/flows/value-stream-test-integration.wsd` 中的测试结果标记
  - 测试成功的用例标记为绿色，测试失败的用例标记为红色
  - 测试报告中对于测试没有通过的单元测试，必须包含以下信息：
    - 具体的测试文件路径
    - 具体的单元测试名称
    - 测试失败的详细原因

#### 测试改动审批
- 描述：规范测试改动的审批流程
- 适用场景：所有测试改动操作
- 优先级：中
- 规则：
  - 对旧有单元测试/集成测试的任何修改（包括删除、修改、重命名等操作）都必须经过审批
  - 审批流程：提交测试修改申请，说明修改原因和影响范围，经相关负责人审批通过后方可执行
  - 审批记录：所有测试改动的审批记录必须妥善保存，以便追溯

### 风格指南
&gt; 统一测试风格和格式的规范

#### 疑难问题处理
- 描述：规范测试中疑难问题的处理方法
- 适用场景：测试中遇到的疑难问题
- 优先级：低
- 规则：
  - 可删除编译文件（如 `Saas_project/frontend/static/vue`）后重新生成

#### 测试账号
- 描述：提供测试用的账号信息
- 适用场景：所有需要测试账号的测试
- 优先级：低
- 规则：
  - 租户测试账号：
    - 邮箱: contact@daydaymoney.com/rgNodkdq8677!ci
    - 作为主要测试账号（`is_superuser=false`）
  - 管理员测试账号：
    - 邮箱：author@example.com/rgNodkdq8677!ci
    - 系统超管（`is_superuser=true`）；用于 `/system-admin/` 等管理端 E2E
  - 非超管 E2E 专用账号（保留，禁止误删）：
    - 邮箱：e2e.nonadmin.sysadmin.redirect@ljytest.com/E2eNonAdmin!8677
    - 用途：非超管访问 `/system-admin/` 应跳转 work-panel 对照用例
    - 维护：`playwright/front_project/tests/ensureE2eNonAdminAccount.mjs`；环境变量 `E2E_NONADMIN_EMAIL` / `E2E_NONADMIN_PASSWORD`

#### AccessKey 使用
- 描述：测试用 AccessKey 的获取方式
- 适用场景：所有需要阿里云 AccessKey 的单元测试
- 优先级：低
- 规则：
  - 从数据库表 `cloud_cloudplatformauthorization` 中 remark 为 `ljy080829@gmail.com` 的记录获取
  - 禁止在规则文件或代码中硬编码 AccessKey 和 SecretKey

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 &gt; 最佳实践 &gt; 风格指南
  2. 文件级规则 &gt; 目录级规则 &gt; 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- 2026-07-15：版本 2.5.3 - 登记非超管 E2E 专用账号 `e2e.nonadmin.sysadmin.redirect@ljytest.com`（禁止误删）；明确超管账号用于 system-admin 对照
- 2026-04-29：版本 2.5.2 - Playwright：补充 `PLAYWRIGHT_TEST_GITHUB_EMAIL`、`PLAYWRIGHT_TEST_GITHUB_PASSWORD`；与根目录 `测试.ai.md` 对齐
- 2026-04-21：版本 2.5.1 - 新增测试短信号码目标规则条目，指向 `port_config.test.json` 的 `django.test_phone_numbers` 与核心测试规则
- 2026-04-13：版本 2.5.0 - 新增 Playwright 测试环境变量登录规则，支持通过 PLAYWRIGHT_TEST_EMAIL 和 PLAYWRIGHT_TEST_PASSWORD 环境变量进行账号登录
- 2026-04-13：版本 2.4.2 - Playwright：本地/智能体默认有头（可见浏览器），仅 CI（`CI` 环境变量）默认无头；与 `playwright.config.js` 约定一致
- 2026-04-11：版本 2.4.1 - 为测试意图伴随文档规则补充统一模板路径 `docs/intents/00_索引/test_file_intent_template.testIntent`
- 2026-04-11：版本 2.4.0 - 新增测试意图伴随文档规则，要求每个测试文件配套 `${testFileName}.testIntent` 自然语言说明并随测试语义同步维护
- 2026-03-18：版本 2.3.0 - 明确 Playwright 测试目录为 `playwright/front_project/tests/`（2026-05 起自 `front_project/app/tests/playwright` 迁入）；补充组件级/流程级命名规则及 config 配置要求
- 2026-03-18：版本 2.2.0 - 移除 AccessKey 明文，改为从数据库获取；修正测试环境启动脚本路径为项目根目录 run.sh
- 2026-03-16：版本 2.1.0 - 从主文件拆分
