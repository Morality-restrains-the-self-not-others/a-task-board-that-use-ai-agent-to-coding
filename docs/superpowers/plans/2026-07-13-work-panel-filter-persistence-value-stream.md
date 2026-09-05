# 价值流：工作面板过滤选项持久化

- 日期：2026-07-13

## 端到端价值流

```
用户打开 work-panel
  → 解析当前 workspace
  → GET filters（Go）
  → 应用 deliverableFilterBars
  → 用户增删改过滤栏
  → debounce PUT（Go）
  → 刷新 / 再打开 → 栏数与路径恢复
  → 切换 workspace → GET 新空间偏好 → 自动切换布局
```

## MVP 增量

| 增量 | 价值 | 验收 |
|------|------|------|
| I1 | 后端可存取偏好 | Go GET/PUT 测例绿 |
| I2 | 打开自动应用 | 刷新后栏数不变 |
| I3 | 切空间自动切换 | 两空间各自栏布局独立 |

## 测试点（价值流）

| ID | 场景 | 对应测例 |
|----|------|----------|
| TP1 | 无记录 GET 返回默认 1 栏 | Go TestGetDefault |
| TP2 | PUT 后 GET 回显 N 栏 | Go TestPutGetRoundtrip |
| TP3 | 两 workspace 隔离 | Go TestWorkspaceIsolation |
| TP4 | 两 user 隔离 | Go TestUserIsolation |
| TP5 | 前端 normalize 非法→默认 | JS unit |
| TP6 | 切换 workspace 加载对应 | WorkPanel 集成 / 手工 |
