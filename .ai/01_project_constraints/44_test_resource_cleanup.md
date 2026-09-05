# 约束 44 — 测试临时资源必须清理（一级）

## 目标

任何测试（单元/集成/Playwright/夜间巡检）运行时**临时创建的资源，运行结束后必须清理**，禁止向 dev 环境留下无主残留。残留资源会污染共享环境（Kafka metadata 膨胀、僵尸进程阻塞删除、磁盘占用），且难以追溯来源。

## 背景（2026-08-10 事故，两次实证）

1. **Kafka topics 污染（3131→61 治理）**：kafka-go 库集成测试被随机单测抽测运行，`DialLeader` 自动创建 3071 个 `kafka-go-<16hex>` topics 且从不清理（约束 43 根治）。
2. **僵尸进程阻塞删除**：3 个历史遗留进程（ppid=1 孤儿）持有 Kafka 活跃连接 → 阻塞 topic 真正删除 → 「清空全部数据库」Kafka 部分失败。

两者都是**测试/遗留进程创建资源后未清理**所致。

## 规则

### 1. 测试代码清理义务（每类语言模板）

| 语言/框架 | 强制模式 | 示例 |
|-----------|---------|------|
| Go | `t.Cleanup` / `defer` 删除测试创建的 topic/文件/进程 | `t.Cleanup(func(){ admin.DeleteTopics(...) })` |
| Python (unittest) | `addCleanup` / `tearDown` | `self.addCleanup(fs.delete, ...)` |
| Python (pytest) | `yield` fixture 的 teardown 段 / `request.addfinalizer` | — |
| JS (vitest/jest) | `afterEach` / `afterAll` | — |
| Playwright | 浏览器 profile 用 `userDataDir` + 结束后删除；trace 指定输出目录并清理 | `fs.rm(tmpProfileDir, {recursive:true})` |

**硬约束**：测试创建了 Kafka topic / MySQL 行 / Redis key / 文件 / 子进程 / 浏览器 profile，**测试结束后必须删除或销毁**。仅当资源位于 gitignore 的临时目录且由基建统一清理时可豁免（须注明）。

### 2. 测试不得向真实 broker 自动创建 topic

- 随机单测抽测排除 `third_party/`（vendored 库测试需真实 broker，不得运行）；`KAFKA_SKIP_NETTEST=1` 双保险（见约束 43）
- 测试如需 Kafka，使用 mock 或隔离容器，禁止连 `localhost:9092/9093` 创建无主 topics

### 3. 残留核查（测试后必查）

**每次会话结束 / 每轮测试运行后**，运行扫描脚本核验：

```bash
bash scripts/lib/test_resource_cleanup.sh            # 扫描（只读）
bash scripts/lib/test_resource_cleanup.sh --fix      # 执行安全清理项
```

扫描覆盖六类资源：
| # | 资源类型 | 检测方法 | 清理方式 |
|---|---------|---------|---------|
| 1 | Kafka junk topics | 匹配 `^kafka-go-[0-9a-f]{16}$` | `python3 db/_infra/kafka_cleanup_junk_topics.py --confirm DELETE_JUNK` |
| 2 | 孤儿进程（ppid=1 + 持有 db/kafka 连接 + 非 runAll 托管名） | ss + /proc cmdline 白名单 | `kill <pid>` |
| 3 | SQLite 残留（tmp/ 下测试生成） | find tmp/ | `rm -rf tmp/<目录>` |
| 4 | 日志/临时大文件 >200MB | find logs/ tmp/ | `truncate-ram-work-logs.sh --all` / `rm -f` |
| 5 | MySQL 测试库残留 | `*_test_*` / `test_*`（排除 registry 生产库） | `python3 db/_infra/drop_mysql_test_databases.py --fix`（`--drop-templates` 才删 `tpl_*`） |

| 6 | 临时容器/端口 | docker ps / ss 自查 | 手动核验 |

### 4. 钩子/自动化

- 扫描脚本设计为**只读 + 引导**；`--fix` 仅执行无争议项：Kafka junk topics、以及 **OpenTestMySQL 残留库**（`*_test_*` / `test_*`）。**禁止**用 `--fix` 跑 `mysql-reset.sh`（会 DROP 生产库）。
- `dbload.makeTestDBCleanup`：`DROP DATABASE` 失败必须让测试失败（panic），禁止只打日志。
- `cloneMySQLSchema`：克隆失败必须 DROP 已创建的 dest 库。
- SessionEnd 钩子（auto-commit.sh）不自动跑扫描（避免每次会话阻塞）；**测试后应跑** `bash scripts/lib/test_resource_cleanup.sh`，发现 MySQL 残留则 `--fix`。

## 验收标准

1. `bash scripts/lib/test_resource_cleanup.sh` 对干净环境输出 `✅ 未发现测试残留资源`（exit 0）
2. 任何新增测试代码若创建外部资源，review 时必须能指出对应的清理语句（t.Cleanup/tearDown/afterEach）
3. 集群 metadata 无 `kafka-go-*` 残留（与约束 43 验收一致）
4. 测试套件运行后执行扫描脚本，无新增残留项

## 关联

- 约束 43：Kafka topic 命名规范与测试污染防治（本约束的 Kafka 专项细则）
- 工具脚本：`scripts/lib/test_resource_cleanup.sh`
- MySQL 清理：`db/_infra/drop_mysql_test_databases.py`（+ `test_drop_mysql_test_databases.py`）；全库重置仍用 `mysql-reset.sh`（会停服）

- 设计文档：`docs/superpowers/specs/2026-08-10-kafka-go-topic-pollution-fix-design.md`
