# 价值流：创建任务结构化可选字段

```
[用户打开创建任务] → [填写标题/可选结构化说明] → [提交]
  → [前端 compose description] → [POST todos] → [任务列表/详情可见]
  → (可选 auto_run) → [容器拉取 title+description 作首指令]
```

## MVP 增量

仅前端 compose + UI；后端与 auto_run 编排零改动即可交付价值。

## 测试点（对接 test-intent）

T1–T7 覆盖 compose/parse/UI；E2E 可选（本迭代以单元测试为主）。
