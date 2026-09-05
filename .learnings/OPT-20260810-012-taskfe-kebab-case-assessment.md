# taskFE 服务名 kebab-case 对齐评估（OPT-20260810-012）

**日期**: 2026-08-11
**状态**: 评估完成 — 结论：**暂不重命名**，保留 `taskFE` 作为受控例外

## 背景

runAll 托管服务绝大多数为 kebab-case（`task-auth`、`task-bill`、`task-gateway`、`task-events-*`…），
前端服务 `name: taskFE` 是唯一 CamelCase 特例（`conf/runAll.yaml:406`）。
OPT-20260810-012 要求评估是否统一为 `task-fe`。

## 引用面盘点

| 位置 | 引用数 | 性质 |
|------|--------|------|
| `conf/runAll.yaml` | 2 | `name: taskFE` + `working_dir: taskFE/app`（后者是 **submodule 目录路径**，非服务名） |
| `conf/value-stream.yaml` | 70 | `taskFE.*` 字段前缀，value-stream 指标标识符 |
| `valueStream/src/fields.go:8` | 1 | 已有兼容层注释：Service segment 允许 CamelCase 匹配 runAll 服务名 |
| `conf/frontend/vue/config.yaml`、`conf/auth/...` | 少量 | 前端 / git-oauth 配置引用 |
| `scripts/register-precise-restart*.sh`、`run-all-intent-tests.sh` | 少量 | 服务登记 / 意图测试引用 |
| `scripts/ci/check_country_codes_sync.py` | 1 | 仅文件系统路径 `taskFE/app/...`，与服务名无关 |
| `.runall/ownership.json` | 运行时 | 服务所有权登记（gitignore，可重建） |
| 文档 | 3+ | `conf/runAll.yaml.ai.md`、`conf/value-stream.yaml.ai.md`、`conf/README.md` |

## 关键约束

1. **submodule 目录路径耦合**：`working_dir: taskFE/app` 指向 meta 仓 gitlink 子模块目录。
   若只把服务名改为 `task-fe` 而子模块目录仍为 `taskFE`，则「服务名 ↔ 工作目录」出现新的不一致，
   比现状（CamelCase 名 + CamelCase 目录）更分裂。
2. **value-stream 指标标识符**：70 处 `taskFE.*` 是 value-stream 基线指标的前缀。改名会破坏
   已录指标的历史对照/趋势分析，除非配合一次 value-stream 基线整体重置。
3. **兼容层已存在**：`valueStream/src/fields.go` 已放宽服务段以兼容 CamelCase——「对齐」的
   实际成本已通过 shim 支付，继续改名的边际收益下降。
4. **全量对齐需连子模块目录改名**：真正把 `taskFE` 全链路对齐为 `task-fe` 需改 gitlink 路径、
   全部 import、脚本、CI、文档，改动量与收益不成比例。

## 结论与建议

- **本次不重命名**：保留 `taskFE` 为受控例外，原因见上（目录路径耦合 + value-stream 基线 +
  compat shim 已生效 + 全量改名成本高）。
- 若未来确需对齐，建议**低成本别名方案**：在 runAll 服务注册层为 `task-fe` 建立指向 `taskFE`
  的别名（或反之），value-stream 字段前缀沿用 `taskFE` 不动；待 value-stream 基线可整体重置时
  再一次性完成字段前缀迁移。
- 无代码变更；value-stream 兼容层保持现状。

## 关联

- OPT-20260810-012（本条目）
- `valueStream/src/fields.go`（服务段兼容层）
