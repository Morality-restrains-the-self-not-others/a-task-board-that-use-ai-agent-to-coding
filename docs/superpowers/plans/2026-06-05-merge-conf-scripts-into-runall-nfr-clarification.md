# NFR 澄清: conf-read/sync 脚本合并进 runAll

> 输入:
> - 设计文档: `docs/design/merge-conf-scripts-into-runall.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-05-merge-conf-scripts-into-runall-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可维护性 | L2 | 配置脚本归属单一 owner（runAll），消除 scripts/→task2app 反向依赖 |
| 性能 | L0 | 不适用——脚本迁移不改变运行时行为 |
| 可伸缩性 | L0 | 不适用——无数据量级变化 |
| 可用性 | L0 | 不适用——无运行时路径变化 |
| 安全性 | L0 | 不适用——无认证/授权变更 |
| 数据一致性 | L0 | 不适用——无数据流变化，sync 行为保持一致 |
| 容错机制 | L0 | 不适用——无网络调用变更 |
| 可观测性 | L0 | 不适用——无日志/指标变更 |
| 合规与隐私 | L0 | 不适用——无数据处理变更 |

## 逐增量 NFR 分析

### Increment 1: conf_loader 提取 + 脚本迁移

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 
  - 配置脚本反向依赖数：从 1（task2app）减少到 0
  - 代码重复消除：`deep_merge`/`load_yaml`/`repo_root` 从 2 份实现收敛到 1 份
- **质量场景**: QS-01

### Increment 2: 外部调用方路径更新
无新增 NFR 要求。

### Increment 3: 旧文件清理 + 全仓验证
无新增 NFR 要求。

## 质量场景

### QS-01: 配置脚本归属一致性
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者（修改配置加载逻辑） |
| 刺激 | 需要了解「配置读取/同步」代码的位置 |
| 制品 | conf-read.py, conf-sync.py, conf_lib.py, conf_loader.py |
| 环境 | 正常开发 |
| 响应 | 在 runAll/scripts/ 下找到所有配置管理脚本，无需跨 task2app/ 追溯 |
| 响应度量 | `find runAll/scripts -name 'conf*' | wc -l` ≥ 5；`grep -rn 'task2app' runAll/scripts/conf*.py` 无结果 |

## 领域模型影响

无。本次变更是脚本文件级重组，不引入新领域概念或改变现有领域模型边界。
设计文档中已列出的领域概念（`AppConfig`、`ConfApp` 聚合、`ConfigSynced` 事件）
将在 `/5-ddd-领域设计驱动` 中按需建模。

## 权衡与边界

### 取舍
- 选择 Python 迁移（非 Go 重写）以保持与现有调用方（taskSSE/Node.js, sync.sh/Bash）的兼容性
- 接受 `conf_loader.py` 与 `task2app/Saas_project/config/conf_loader.py` 短期共存两份实现。长期 task2app 可委托 runAll 版本

### 明确不做什么
- 不重写 conf-read/conf-sync 为 Go——调用方通过 CLI 消费，非 Go import
- 不修改 task2app 的 conf_loader.py——task2app 内部仍在使用，不在本次范围
- 不改变 sync.manifest.yaml 格式或 sync 行为——纯路径迁移

### 升级触发条件
- 当 task2app/conf_loader.py 准备废弃时，可替换为 `runAll/scripts/conf_loader.py` 的委托
- 当需要性能优化时，可考虑将 conf-read 逻辑编译进 runAll Go binary（`conf-read` 子命令）

## 跳过声明

- **性能、可伸缩性、可用性、安全性、数据一致性、容错机制、可观测性、合规与隐私**: 跳过。本次变更是脚本文件物理位置迁移，不影响运行时路径、数据流、网络调用或安全边界。所有 skip 的 NFR 类别与变更内容无关。
