# Completed OPT Archive — 2026-08-09

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 3 条。
> 归档执行时间：2026-08-13T13:17:38+08:00

## [OPT-20260809-005] completed

- **Status**: completed
- **Completed**: 2026-08-09
- **Summary**: 12 个视图文件修复，主目录 19 文件全量 check 0 critical/0 warning；verify-archimate-load.sh 复核 12 文件全部可加载；docs 子仓 f919ebc 已推送
- **Created**: 2026-08-09
- **Context**: archimate-tool.py check 批量扫描发现主目录 22 个文件中 15 个有 critical（v13-v16 landscape full/diff 1-19 个、v63-v68 application 19-37 个），典型：连线端点元素 ≠ 关系 source/target（cf-rel-trigger 引用 rel-trigger 但端点 plateauV12/13≠plateauV14、gapHooks/Session≠gapPost）、targetConnections 引用不存在的连线（cf-rel1~4 跨视图复用后悬空）、id 重复（referralSvc）。
- **Action**: (1) 对每个文件用 `archimate-tool.py check` 定位 critical；(2) 修正连线 archimateRelationship/端点或关系端点；(3) check 归零 → `verify-archimate-load.sh` 复核；(4) 修复后全量基线应 0 critical。
- **Why**: 引用不一致会导致 Archi 加载失败或视图连线错乱；工具已交付（OPT-20260809-002），修复为纯机械批处理。
- **How to apply**: `docs/architecture/scripts/archimate-tool.py`；不影响工具交付，排期另行。

## [OPT-20260809-006] completed

- **Status**: completed
- **Completed**: 2026-08-09
- **Summary**: spa-catch-all hosts 绑定静态门禁已落地（taskGateway 5ef176b，已推送）。validate_spa_catch_all_no_hosts 在 generate() 最前执行，hosts 字段（含空列表）即 ValueError 硬失败，--check/--lint/写出路径全部拦截；连带修复 --lint 原本会写文件的脚枪（无 docker 环境写 loopback 上游 → 502，与 OPT-026 同类）。新增 8 个回归单测（含 lint 不写文件守护），生成产物字节一致，--check 通过。
- **Created**: 2026-08-09
- **Context**: 2026-08-09 修复 www.daydaymoney.com /api/auth/wechat/login/ 502/500（回归点 8f6194c：spa-catch-all 加 hosts [daydaymoney.com, www.daydaymoney.com] 后劫持该域名全部 GET/HEAD，priority 10 压过 taskauth-login 850，转发 taskFE:4000 → 500）。已移除 hosts 并加注释防护，但无自动化门禁。
- **Action**: 在 taskGateway/scripts/routes-to-apisix.py（或 .githooks/check 脚本）加静态检查：spa-catch-all 路由若声明 `hosts` 字段 → 生成失败并报错，防止未来再次引入 hosts 回归。
- **Why**: 该回归静默存在 6 天（08-03→08-09），仅靠注释提醒不可靠；生成脚本是 SSOT 生成链，是天然门禁点。
- **How to apply**: routes-to-apisix.py 在写 apisix.yaml 前校验 `rule.id == 'spa-catch-all' and 'hosts' in rule` → raise；同时补 --lint 规则与用例。

## [OPT-20260809-007] completed

- **Status**: completed
- **Completed**: 2026-08-09
- **Summary**: auto-commit.sh 新增 --checkpoint-threshold 模式（Stop hook 每轮触发，按「距上次成功检查点提交 N 秒」阈值低频去重，幂等由 git diff 空检查兜底，状态文件 .git/auto-commit-checkpoint 记 unix 时间戳；检查点提交独立 message「会话中途检查点提交」）。模板 + live settings 的 Stop hook 增挂 --checkpoint-threshold 1800（30 分钟，timeout 900s）。scratch repo 7 例功能测试全过。
- **Created**: 2026-08-09
- **Context**: SessionEnd hook 自动提交（`scripts/lib/auto-commit.sh`，CLAUDE.md 元规则）只在**正常退出**时触发；终端强杀/断电/崩溃等异常路径无保护，超长会话中间数小时的工作会丢失。
- **Action**: 在 `.claude/settings.json`（模板 `scripts/hooks/templates/claude-settings.json`）为 Stop hook 增加低频检查点调用（如每 20 次工具调用或 30 分钟一次；auto-commit.sh 已幂等，同内容 no-op 可复用），提交频率低不污染历史。
- **Why**: 恢复安全网目前只覆盖「会话正常结束」；崩溃场景是数据丢失的最高风险点。
- **How to apply**: Stop hook 命令可带 `--checkpoint-threshold` 参数复用 auto-commit.sh（内部记 last-checkpoint 时间戳/内容哈希去重）；注意 Stop 每轮都触发，须用阈值+幂等双重去重，且走同一完整门禁。


### OPT-20260809-010 — 评论容器启动日志细化：由单状态日志扩展为多阶段时间线
- **Status**: completed
- **日期**: 2026-08-09
- **Completed**: 2026-08-09
- **Summary**: 修复「服务器启动日志只有一条 [13:47:51] 正在启动容器实例，无法了解启动详细过程」。根因：前端 per-binding 启动日志仅按 binding 状态机（pending→starting→running）每状态追加一行，starting 阶段（实例创建+启动+可达性等待，通常数分钟）只有一条日志。方案（前端 taskFE，真实信号驱动非伪造）：1) 新增 BINDING_STAGE_LOG_MESSAGES 阶段消息——cscAllocated（starting+已挂接评论级 CSC →「容器实例已分配，等待服务就绪」）、heartbeatEstablished（SSE container_heartbeat 首达 →「容器心跳已建立，服务启动中…」）、probeOk（「服务健康探测通过」）、bidirectionalOk（「双向通信通道已建立」）；2) 追加函数重构为 appendBindingLogLine（精确去重跨全部历史行，支撑幂等回填）；3) 三条信号路径接入——refreshBindings 轮询/冷打开回填、applyPerBindingHeartbeatFromSse 心跳阶段、applyBindingAdvancedFromSse SSE 实时路径。完整启动时间线由 1 条 → 至多 7 条（排队→启动→分配→心跳→探测→双向→就绪）。
- **Verification**: 新增 3 单测（refresh 回填+顺序+幂等、心跳阶段去重、SSE 实时路径）全绿；useCommentContainerBindings.test.js 5/5 通过；taskDetail 组合式函数目录回归 243/243 通过（41 文件，0 错误）——注：最初报告的「2 个 jsdom 缺包 ERR_MODULE_NOT_FOUND」为误报，根因是测试调用方式（npx 缓存 vitest 4.1.9 解析不到项目 jsdom），改用项目自带 vitest（app/node_modules v4.0.18）后 14/14 通过。改动文件：taskFE useCommentContainerBindings.js + 测试。

