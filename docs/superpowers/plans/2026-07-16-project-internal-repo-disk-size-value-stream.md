# 价值流：内部仓磁盘占用展示

日期：2026-07-16

## 触发

用户打开项目详情页，查看「Git 仓库」列表。

## 价值流（最小增量）

```
用户打开详情
  → Gateway JWT/tenant
  → taskProjectService GET project
  → enrich OAuth status（既有）
  → enrich disk size（仅 gitlab-local）
  → Vue 渲染：内部仓显示「磁盘：x.x MB」；外部仓无该徽章
```

## 验收切片

1. 内部仓 + statistics 成功 → 可见磁盘徽章。
2. 外部仓 → DOM 无 `git-repo-disk-size`。
3. 内部仓无 token/API 失败 → 无磁盘徽章（不显示错误数字）。

## 事件

纯查询，无 MQ 事件（书面例外）。
