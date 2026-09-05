# Value Stream：项目自动克隆子仓库开关

- **日期**: 2026-08-12
- **设计**: `docs/superpowers/specs/2026-08-12-project-auto-clone-nested-repos-design.md`

## 主价值流

```
用户打开创建项目
  → 填写 Git 仓库
  → 看到「自动克隆子仓库」勾选（默认开）
  → 提交 → project_entries 持久化
  → 之后创建任务并启动容器
  → container-snapshot 带 auto_clone_nested_repos
  → credential enrich：true 则 merge 子仓并克隆；false 则仅父仓
  → 用户在项目详情可改开关（影响后续容器）
```

## 测试点

| ID | 步骤 | 期望 |
|----|------|------|
| VS-1 | 创建勾选 true | DB=1；snapshot true；enrich 含子仓 |
| VS-2 | 创建取消勾选 | DB=0；enrich 无子仓；父仓仍在 |
| VS-3 | 详情改 false→true | 下次 task-detail 含子仓 |
| VS-4 | 发现 API 在 false 时 | 仍可列出 nested（发现≠克隆） |
| VS-5 | 旧项目无列 | DEFAULT 1，行为不变 |
