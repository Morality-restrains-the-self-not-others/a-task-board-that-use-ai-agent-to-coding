# 价值流 — 管理端赠送 GitLab 须选区域

Mapping the approved design into a value stream.

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-admin-grant-gitlab-region-design.md`

## Related Value Streams

- `2026-08-18-pluggable-multi-region-gitservice-value-stream.md`：租户**购买**须选区域。本流是其管理端赠送缺口补齐（modification / extension），不撤销购买约束。

## 价值增量（最小可交付）

单一垂直切片即可交付：

1. **管理员指定区域赠送磁盘（含流量同约束）**  
   系统管理员打开赠送页 → 选租户 → 资源类型 GitLab 磁盘 → 选区域 → 填数量 → 确认 → 该区域配额增加。

步骤：

| 步 | 角色 | 系统 | 产出 |
|----|------|------|------|
| 加载区域列表 | 系统管理员 | taskFE → GET gitlab-regions | 下拉选项 |
| 选择磁盘 + 区域 | 系统管理员 | taskFE | 行内 `region` |
| 提交赠送 | 系统管理员 | taskBill `adminGrantResources` | 区域配额 + 订单行 region |
| 失败：未选区 | 系统管理员 | FE 拦截 / BE 400 | 不写默认区 |

## 不做

- 不改购买页、不开通 GitLab group、不新增 Kafka、不限制赠送 1GB。
