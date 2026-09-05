# Kafka `kafka-go-*` 无意义命名 Topic 污染 — 根因分析与修复设计

- **迭代**: 2026-08-10-kafka-go-topic-pollution-fix
- **作者**: claude
- **设计日期**: 2026-08-10
- **状态**: 🎯 target（待实施）

## 1. 问题背景

Kafka 集群共 **3131 个 topics**，其中 **3071 个是 `kafka-go-<16hex>` 前缀**（占 98%）。这些 topic 名称由随机 16 进制数构成，无任何业务语义，违反「topic 创建应使用有意义的名称」原则，并造成：

- metadata 膨胀（3131 topics 使 list_topics/controller 元数据管理压力增大，实测影响 topic 删除-重建可靠性）
- 排查干扰（与业务 topic 混在一起难以区分）
- 存储与运维成本（每个 topic 独立日志目录与副本）

## 2. 根因分析（调查证据链）

`kafka-go-<16hex>` 前缀 topic 的来源是 **kafka-go 客户端库的测试代码**，而非任何业务代码：

| # | 环节 | 证据 |
|---|------|------|
| 1 | 命名来源 | `third_party/kafka-go/conn_test.go:100` — `makeTopic()` 返回 `fmt.Sprintf("kafka-go-%016x", rand.Int63())`；全仓 **80 处**调用（30 个测试文件） |
| 2 | 自动建 topic 机制 | `conn.go:982` — metadata 请求 `AllowAutoTopicCreation: true`；`DialLeader` 对不存在的 topic 自动创建 |
| 3 | 测试执行者 | **随机单测抽测**：taskAuth pre-commit 门禁（`random_test_runner.sh`，30% 目录命中率）+ 夜间巡检（cron `*/20 0-7 * * *` nightly_test_sweep.py，每 20 分钟对全部子仓抽测） |
| 4 | 为何被抽中 | `rt_go_collect_dirs` 排除规则仅 `-not -path './vendor/*'`，**不含 `third_party/`** — vendored 上游库的集成测试被误当作本仓单测资产 |
| 5 | 为何可达 | 测试默认连 `localhost:9092`；本机 docker kafka 双端口映射（9092/9093 均在监听，已实测） |
| 6 | 为何累积 | 测试无 teardown 清理；`TestConn` 单次运行 36 个 scenario 各建 1 topic，另有 30 个测试文件含 makeTopic → 每轮运行创建数十个 topic 且永久残留 |

**量化**：3071 个 ÷ 每轮约 20-40 个 ≈ 100-150 轮随机抽测积累（pre-commit 每提交 + 夜间每 20 分钟，数月必然达到）。

**业务代码核查结论**：业务 topic 全部语义命名（28 业务 + 28 DLT，定义在 `taskEvents/config/config.go` EventTopic/DeadLetterTopic 映射，与 `db/_infra/kafka_recreate.py` 同步）。**无任何业务代码以 ID/hash 命名 topic**。

## 3. 修复方案（用户已确认：完整修复）

### 3.1 根治 — 随机单测抽测排除 third_party（治本）

**修改 `scripts/lib/random_test_runner.sh` 的 `rt_go_collect_dirs`**（SSOT），排除 `third_party/`：

```bash
find . -type f -name '*_test.go' \
    -not -path './.git/*' \
    -not -path './node_modules/*' \
    -not -path './vendor/*' \
    -not -path './third_party/*' \
    | ...
```

**同步修改模板** `scripts/hooks/templates/lib/random_test_runner.sh`（SSOT 副本，deploy 分发源），并执行 `bash scripts/deploy_repo_random_precommit.sh` 重新分发到各仓 `.githooks/lib/`。

**双保险**：`rt_go_run_dir` 执行测试时注入 `KAFKA_SKIP_NETTEST=1` 环境变量（kafka-go 测试自身的门控：设置后跳过 nettest 子测试），防止未来 third_party 排除失效或他处直接运行该库测试时仍创建 topic。

**理由**：`third_party/` 与 `vendor/` 同属 vendored 第三方代码，其测试是上游库的集成测试（需要真实 broker），不属于本仓单测资产，不应被随机门禁/夜间巡检运行。

### 3.2 存量清理 — 一键删除 3071 个残留 topics

新增 `db/_infra/kafka_cleanup_junk_topics.py`（复用 kafka_recreate.py 的 AdminClient 模式）：

- `--dry-run`（默认）：列出将删除的 topic 数量与名称采样，不执行
- `--confirm DELETE_JUNK`：删除全部 `kafka-go-*` 前缀 topics（保留业务 topics + DLT + `__consumer_offsets`）
- 删除前验证：仅匹配 `^kafka-go-[0-9a-f]{16}$` 模式（防止误删）
- 执行时机：服务停止期间或确认无活跃连接时（Kafka 对活跃连接阻塞删除 —— 本会话已实证该行为）

### 3.3 约束固化 — 命名规范

- 将「禁止测试/代码自动创建非语义命名 topic；`kafka-go-*` 前缀保留给 kafka-go 上游测试，业务 topic 一律走 EventTopic 映射」写入 `.ai/01_project_constraints/` 或设计文档（命名规范单一来源：`taskEvents/config/config.go` EventTopic + `db/_infra/kafka_recreate.py`）
- OPT 追踪：清理脚本执行记录 + 后续监控（如再出现 kafka-go-* 增量即告警）

## 4. 测试策略

| 测试 | 内容 |
|------|------|
| `random_test_runner.sh self-test` | 内置断言自测（collect-go 排除规则变更后回归） |
| 新增断言 | `rt_go_collect_dirs taskAuth` 输出不含 `./third_party/kafka-go` |
| 清理脚本 | 单测：mock AdminClient 验证 `--dry-run` 不删除、`--confirm` 只删 `^kafka-go-[0-9a-f]{16}$` 模式 |
| 实机验证 | 执行清理脚本 dry-run → 确认数量 → 确认删除 → 复查 list_topics 仅剩业务 topics |

## 5. 🕸️ Code Review Graph 分析

`skipped_non_code` — 本次变更主体为 bash 脚本（random_test_runner.sh）与运维脚本，非 Go 符号；codegraph 索引无相关覆盖，静态检查已人工完成（排除规则 grep 验证）。

## 6. 业务意图 → 事件对照

不适用 — 本设计为测试基建/运维治理，不改变系统事实，不引入业务事件。

## 7. 🏛️ 架构变更影响

**不更新架构文件**（`docs/architecture/` v17/v69 current 保持不动）— 依据判断原则：「纯 Bug 修复、配置微调 → 不需要更新架构」。本设计不增删任何服务组件/数据流/基础设施，仅修正测试基建的目录排除规则与存量清理。

## 8. 实施步骤

1. 修改 `scripts/lib/random_test_runner.sh`：collect-go 排除 `third_party/` + run-go 注入 `KAFKA_SKIP_NETTEST=1`
2. 同步 `scripts/hooks/templates/lib/random_test_runner.sh`，执行 deploy_repo_random_precommit.sh 重新分发
3. 运行 `random_test_runner.sh self-test` + collect-go 断言回归
4. 新增 `db/_infra/kafka_cleanup_junk_topics.py`（dry-run + confirm 两段式）
5. 执行清理（dry-run → confirm），复查 list_topics
6. 约束固化 + OPT 落盘
