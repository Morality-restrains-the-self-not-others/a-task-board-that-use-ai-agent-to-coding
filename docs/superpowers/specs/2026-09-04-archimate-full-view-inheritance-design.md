# 设计：`.full.archimate` 视图目录须继承上一版本

- **Status:** accepted（goal-mode 自动采纳）
- **Date:** 2026-09-04
- **Iteration:** archimate-full-view-inheritance
- **Author:** cursor

## 问题陈述

打开近期 `*.full.archimate`（如 v126–v132）时，`Views` 目录只有本迭代的 2 张图（「全量拓扑 — 本迭代」+「架构变迁 vN-1→vN」），**看不到上一版本已积累的视图**。这与「全量模型 = 改完后的完整架构」的约定冲突。

## 当前架构理解（基线）

- 架构制品按视图分文件：`enterprise-landscape` / `application-integration` 等。
- 每版本交付四类伴生：`.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`。
- **`.diff`**：仅本迭代变更；**`.full`**：变迁后完整拓扑（应可在 Archi 看全量）。
- 最近 current/target：v131（application-integration）、v132 target（wechat-mp-egress）；enterprise-landscape 近期随迭代偶发更新。

📋 架构版本历史（相关）：

- v122–v125 ✅ — `.full` **曾正确累积 Views**（v125 enterprise-landscape：8 视图，含 v122–v125 拓扑与各版变迁）
- v126 起 ❌ — `.full` 退化为「本迭代切片」，与 `.diff` 软等价（同元素数/同 2 视图）

## 根因分析（不是「Archi 丢视图」，是生成路径断了）

### 证据

| 版本 | enterprise-landscape `.full` 视图数 | 是否继承上一版视图名 | 与同版 `.diff` 关系 |
|------|-------------------------------------|----------------------|---------------------|
| v124 → v125 | 6 → **8** | ✅ 保留全部旧名并追加 | full ≫ diff |
| v125 → v126 | 8 → **2** | ❌ 交集为空 | full 接近切片 |
| v126 → v132 | 恒为 **2** | ❌ 无继承 | 多版 soft-identical（元素数=视图数=diff） |

附加断裂点：

1. **元素 id 全量换号**：v124∩v125 稳定 id 重叠 65/65；v125∩v126 重叠 **0**。`migrate-legacy-archimate.py` 的 `merge_full()` 依赖 **稳定 id**；换号后无法机械合并。
2. **技能只写意图、无可执行继承算法**：Step 3e-2 有一句「以上一版 `.full` 为基叠加」，但未规定 `cp` → 保留 Views → 追加 → 稳定 id；验收清单也不检查「视图数 ≥ 上版」。
3. **无 CI 门禁**：`Loaded model:` 只证明 XML 能打开，不证明「全量」。
4. **Agent 捷径**：在 MCP 不可用 / goal 零交互下，把本迭代 `.puml`/`.diff` 复制成 `.full`，仍能通过现有清单勾选。

### 结论（对技能的判定）

**是技能设置不完整（可执行性缺口），不是「没有说要全量」。**

- 意图层：✅ 已写「全量 / 叠加上一版 full」
- 操作层：❌ 缺强制继承步骤、稳定 id 规则、Views 保留规则
- 门禁层：❌ 缺「full 视图 ⊇ 上版视图」自动检查

仓内已有正确先例与工具：`docs/architecture/scripts/migrate-legacy-archimate.py`（`merge_full` = 上版 full + 本版 diff），以及 v122–v125 的累积 Views。

## 🕸️ Code Review Graph 分析

`skipped_non_code` — 本任务为架构制品生成技能/门禁缺陷分析，不涉及业务符号调用链。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 修复架构制品生成规范 | — | 纯文档/技能/CI；无服务端状态变更，无对应 MQ 事件 |

## 方案决策（已采纳）

### 生成 `.full.archimate` 的强制算法

```
1. 定位同视图上一版 .full（current 或 archive 中最近同 @view）
2. cp 上一版 .full → 新文件 vN-….full.archimate
3. 保留上一版全部 ArchimateDiagramModel（Views）与稳定 element/relationship id
4. 将本版 .diff 中的 🟢/🟡/🔴 元素与关系 merge 进语义层（新元素才新发 id）
5. 追加（不替换）：「架构变迁 vN-1→vN」视图 + 「vN 全量拓扑 — <迭代>」视图
6. 更新 model name；删除已 [DEPRECATED] 且交付要求移除的元素时，同步清 Views 中对应 DiagramObject
7. Archi --loadModel + CI inherit 检查
```

`.diff.archimate` 仍可只含本迭代（不变）。

### 技能 / 规则改动

1. `1-brainstorming-design-docs` Step 3e-2：改为上述算法；禁止「从零写 full」或「复制 diff 改名」。
2. `archimate/SKILL.md` + `archimate-architecture-artifacts.mdc`：增加 **稳定 id**、**Views 累积**、验收项。
3. 新增 CI：`db/scripts/ci/check_archimate_full_inherits_views.py`（及自测）。

### 存量债务（已完成）

v126–v132 的 `.full` 继承链已由 `docs/architecture/scripts/backfill_full_archimate_views.py`（OPT-20260904-011）回填；CI `DEBT_VERSIONS` 已清空。

## 🏛️ 架构变更影响

本迭代**不新增业务组件**；不创建新的 `vN-*.puml` 架构版本。变更对象为技能、Cursor 规则与 CI 门禁。

## 验收标准

1. 技能/规则明文禁止「full = 本迭代切片」
2. CI：同视图新 `.full` 的 Views 名集合必须 ⊇ 上一版 `.full`（允许额外新增）
3. CI：若上一版 Views ≥ 3，则新版不得只有 2 张且与 `.diff` 元素数相等（反切片启发式）
4. 自测脚本全绿

## 价值流影响

无业务 value-stream 变更；影响架构制品生产质量门禁。
