# 价值流：创建任务可选字段显隐

**日期**: 2026-07-18

## 端到端流

```
管理员打开 settings/task-panel
  → 打开「创建字段」配置
  → PUT create-task-field-settings
  → 成员打开 work-panel 创建任务
  → GET create-task-field-settings
  → CreateTaskModal 仅渲染 enabled 可选字段
  → 提交 todos（隐藏字段用默认/空）
```

## MVP 切片

1. Go GET/PUT + 默认全 true
2. Settings modal 保存
3. WorkPanel 拉取并门禁 UI + feature_params 提交门禁联动

## 测试点

| ID | 场景 |
|----|------|
| VS-1 | 无记录 GET → 全 true |
| VS-2 | PUT priority=false → GET 反映 |
| VS-3 | 创建表单无优先级控件 |
| VS-4 | feature_params=false 时可不选环境变量即可提交 |
