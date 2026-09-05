# 请求报错展示须带 data-traceId（前端可观测）

## 基本信息

- 版本：1.3.0
- 创建日期：2026-07-17
- 最后修改：2026-07-28
- 维护者：Trae AI 团队

## 背景（为何是元规则）

后端与网关已按可观测性规范透传 `X-Trace-Id`，日志字段为 `trace_id`（见 [12_observability_log_shipping.md](../03_technical_implementation/12_observability_log_shipping.md)）。排障时用户或 Agent 常只能从**页面上的报错 UI**入手；若错误节点未挂载请求级 traceId，则无法一键在 Grafana/Loki 中按 `{job=~".+"} | json | trace_id="<id>"` 定位链路。

将 traceId 写在**展示该次请求失败**的 DOM 元素上（`data-traceId`），使 Playwright、浏览器 DevTools 与 Agent 截图/快照均可直接提取，缩短「报错 → 日志」闭环。

## 规则分类

### 核心规则

#### 报错展示元素必须携带 data-traceId

- **描述**：凡因**网络 / HTTP / RPC 请求失败**而在前端展示给用户的错误 UI（含 toast/snackbar、inline 错误文案、表单字段错误、错误页、错误弹层、列表空态中的失败提示等），承载该错误文案或错误状态的**可见 DOM 节点**（或其稳定可查询的根节点）**必须**设置属性 **`data-traceId`**，值为与该次失败请求对应的 traceId 字符串。
- **适用场景**：
  - SPA / Vue / React / 模板页中展示 API 失败、超时、4xx/5xx、业务错误码（由请求响应产生）
  - Chrome 扩展 popup、onlineService 管理页等同类前端
  - 新增或修改错误展示组件、统一错误处理、axios/fetch 拦截器、消息提示封装
- **不适用**：
  - 纯前端校验（未发出请求）产生的提示——**不得**伪造 `data-traceId`
  - 仅写控制台 / 遥测、不向用户展示错误 UI 的路径（仍建议在日志中带 traceId，但不强制 DOM 属性）
- **优先级**：高
- **规则类型**：禁止忽略（新增/改动报错展示时）

##### 属性约定

| 项 | 要求 |
| --- | --- |
| 属性名 | 固定为 **`data-traceId`**（字面；禁止改成 `data-trace-id` / `data-trace_id` / `data-request-id` 等别名作为项目约定） |
| 属性值 | 与该次失败请求一致的 traceId 非空字符串 |
| 挂载位置 | 展示错误的元素本身；若文案在子节点，允许挂在**错误容器根节点**（须可用选择器 `[data-traceId]` 唯一定位到该次错误） |
| 多请求并发 | 每个错误展示实例各自携带**对应请求**的 traceId；禁止多个无关错误共用同一节点覆盖写值 |
| 缺失时 | 请求侧确无 traceId（响应头/请求头/响应体均无）时：**省略**该属性，禁止填占位符（如 `"unknown"`、`"N/A"`） |

##### traceId 取值优先级

按顺序取第一个非空值：

1. 响应头 `X-Trace-Id` / `x-trace-id`（网关或上游回传）
2. 本端发出该请求时使用的 `X-Trace-Id`（客户端已生成并随请求发出）
3. 响应 JSON 中的 `trace_id` / `traceId`（若业务体携带）

与后端日志字段 `trace_id`、领域事件 `data.trace_id` 对齐；前端属性名保持 **`data-traceId`**（camelCase 后缀），便于与既有前端命名习惯一致。

##### 实现与评审检查要点

1. **统一出口优先**：在 axios/fetch 封装或全局错误提示组件中解析并下传 traceId；业务页只消费，避免每处手抄请求头。
2. **模板绑定示例**（原则示意）：
   - Vue：`<div class="error" :data-traceId="traceId">{{ message }}</div>`
   - React：`<div className="error" data-traceId={traceId}>{message}</div>`
3. **Toast / 第三方组件**：若组件 API 不支持透传 DOM 属性，须包一层带 `data-traceId` 的容器，或改用可挂载属性的自研错误展示；禁止「只弹文案、DOM 上无 traceId」。
4. **测试**：Playwright / 组件测在模拟失败响应（带 `X-Trace-Id`）后，断言错误节点 `getAttribute('data-traceId')`（或 `[data-traceId="<id>"]`）等于期望值。
5. **Agent / 排障（日志优先硬门禁）**：用户报告前端报错时：
   1. 从页面快照或 DevTools **读取** `data-traceId`（**大小写不敏感**：快照/HTML 可能呈 `data-traceid`、`data-TraceId`；用户文案亦可能写 `traceid` / `TraceId` / `trace_id`）
   2. **优先**用该值查 Loki/Grafana，按时间线重建整条错误发生路径（跨服务、首错点、上下游）
   3. **禁止**在未完成日志检索（或已确认 Loki 不可达并记录）之前，仅凭错误文案猜根因或直接改代码
   4. 完整查询步骤见 `.claude/skills/1-brainstorming-design-docs/SKILL.md`「TraceId 驱动的 Grafana 日志分析」；短规范见同目录 `references/traceid-log-first-diagnosis.md`
   
   项目**写入**约定仍固定为字面 `data-traceId`，不因读取兼容而改写前端属性名。

##### Loki 按 trace_id 检索验收命令（2026-08-18 复验通过）

`Loki http://127.0.0.1:3100` 经 promtail-local 采集 `logs/*.log`；JSON 日志字段 `trace_id` 与失败响应头 `X-Trace-Id` 首段一致，另含 `otel_trace_id`（traceparent trace-id）。端到端验收命令：

```bash
# 1) 触发一次失败请求并取回 X-Trace-Id（示例网关 :8014，404 同样回头）
curl -s -D - -o /dev/null 'http://127.0.0.1:8014/api/tenant/999/nonexistent-path' | grep -i x-trace-id
#    → X-Trace-Id: 7815a013033a3d278efdc087

# 2) 按该 trace_id 在 Loki 检索整条链路（按单 job 收敛，跨 {job=~".+"} 会触发 series 上限）
curl -sG 'http://127.0.0.1:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={job="task-container-gateway"} | json | trace_id="7815a013033a3d278efdc087"' \
  --data-urlencode "start=$(date -d '1 hour ago' +%s)000000000" \
  --data-urlencode "end=$(date +%s)000000000" --data-urlencode 'limit=20'
```

注意：
- **不要用 `{job=~".+"} | json` 全量查**——匹配流数超 500 会报 `maximum of series`；先按 `job="<服务>"`（如 `task-container-gateway` / `task-cloud-service`）收敛再加 `| json | trace_id="..."`。
- 非单行 JSON（多行栈/截断）会出 `JSONParserErr`，可在 `| json` 后追加 `| __error__=""` 跳过。
- 页面 `data-traceId` 取自响应头 `X-Trace-Id` 首段（`traceId.js` 拆分多行同名头），与 Loki `trace_id` 直接对齐；Grafana Loki datasource（`http://loki:3100`）复用同一查询。

##### modalService.alert() 强制传 traceId 规则

当 `modalService.alert()` 用于展示**API 请求失败**的错误时，**必须**以对象形式传入 `{ message, traceId }`，**禁止**传入纯字符串（纯字符串会导致 traceId 丢失，`data-traceId` 缺位）。

- ✅ 正确：`modalService.alert({ message: '登录失败：xxx', traceId })`
- ✅ 正确（无 traceId 可用时）：`modalService.alert({ message: '登录失败：xxx' })`（不传 traceId，DOM 不挂 data-traceId）
- ❌ 错误：`modalService.alert('登录失败：xxx')` — traceId 丢失，违反元规则

**特别强调：登录/注册/认证流程**的 API 错误展示是用户排障的最高频入口，必须严格遵循此规则。
`showRequestError()` 工具函数已内置 traceId 提取逻辑，新代码应优先使用该函数；若直接调用 `modalService.alert()`，则必须手传 traceId。

##### 内联错误展示模式（不使用 showRequestError/modalService 时）

当错误以**内联方式**展示（表单内嵌错误提示、卡片内错误文案等），不走 `showRequestError` → `modalService` 统一出口时，开发者**必须手动**：

1. 使用 `safeResponseJson`（而非 `safeJson`）解析响应以获取 `traceId`
2. 声明 `ref` 存储 traceId（如 `const errorTraceId = ref('')`）
3. 错误 DOM 节点绑定 `:data-traceId="errorTraceId || undefined"`
4. 打开/重置表单时清空 `errorTraceId.value = ''`

反例（禁止）：仅用 `<div class="bg-red-50 text-red-700">{{ errorMessage }}</div>` 而无 `data-traceId`。

##### 关键文件审计清单

以下文件是 API 错误展示的高频热点，改动时必须确认 traceId 链路完整：

- `taskFE/app/src/composables/auth/useLoginSubmit.js` — 登录提交流程
- `taskFE/app/src/components/AuthForms.vue` — 旧版登录表单
- `taskFE/app/src/composables/auth/useRegisterSubmit.js`（若存在）— 注册提交流程
- `taskFE/app/src/utils/requestErrorDisplay.js` — 统一错误展示出口
- `taskFE/app/src/views/SystemAdminUsers.vue` — 系统管理用户页（含邮箱邀请内联错误）
- `taskChromePlugin/popup/popup.js` — Chrome 插件登录

##### CI 守卫

预提交钩子 `check-trace-id-fixtures` 通过 `db/scripts/ci/check_frontend_error_data_trace_id.py` 扫描 Vue 模板中错误类名（`text-red-*`、`text-danger`、`bg-red-50`）附近是否缺少 `data-traceId`。

**重要**：该脚本的扫描路径必须与实际前端源码目录保持同步。当前扫描 `taskFE/app/src/`（历史路径 `task2app/front_project/app/src` 已移除）。若前端项目迁移目录结构，须同步更新脚本中的 `_FRONT_CANDIDATES` 列表。

### 最佳实践

- 成功路径无需设置 `data-traceId`。
- 错误消失（关闭 toast、重试成功）时应移除对应节点或清空属性，避免陈旧 traceId 误导。
- 用户可见文案可不强制展示 traceId 明文；**DOM 属性是给工具与排障用的契约**，与是否在 UI 文案中打印 traceId 无关。

## 关联

- 项目约束索引：[00_project_constraints.md](./00_project_constraints.md) 第 26 条
- 可观测性 / 透传：[../03_technical_implementation/12_observability_log_shipping.md](../03_technical_implementation/12_observability_log_shipping.md) §1.2
- 前端规范：[../04_frontend_development/00_frontend_development.md](../04_frontend_development/00_frontend_development.md)
- Cursor 元规则：`.cursor/rules/frontend-error-data-trace-id.mdc`

## 变更日志

- 2026-07-28：1.3.0 批量修复 CI 守卫失效后 34+ 处视图/组件的 data-traceId 缺口；新增内联错误展示模式规范；CI 脚本路径修复 + 类名模式扩展至 `text-red-[5-8]00|text-danger|bg-red-50`；更新审计清单（`front_project/` → `taskFE/`）。见 [86_ci_traceid_check_wrong_path_data_traceid_missing.md](../09_failure_experience/02_runtime_errors/86_ci_traceid_check_wrong_path_data_traceid_missing.md)。
- 2026-07-20：1.1.0 Agent/排障明确「有 `data-traceId` 须先日志检索重建全路径，再源码猜测」硬门禁；交叉引用 `traceid-log-first-diagnosis.md`。
- 2026-07-18：1.0.1 Agent/排障读取 `data-traceId` 及用户文案中的 trace 标注须大小写不敏感；写入约定不变。
- 2026-07-17：1.0.0 初版；约定请求报错展示 DOM 必须带 `data-traceId`。

## Golden fixtures

见 [fixtures/traceid_extract_cases.md](./fixtures/traceid_extract_cases.md)。
