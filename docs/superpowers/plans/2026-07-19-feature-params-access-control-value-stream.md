# Feature-Params 访问控制 — 价值流

## 增量价值流

```
成员打开设置页 → 带 Access-Context 请求 full → 审计落库 → 展示/编辑密钥配置
成员打开任务详情 → view=summary → 审计落库 → 仅模型/预算开关 → 无密钥
非法拉取 full → 403 + denied 审计 → 前端不展示密钥
```

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 无 header GET 公司 feature-params | 403 |
| T2 | company_settings GET | 200 + api_key |
| T3 | view=summary GET | 200 + api_key 空 |
| T4 | workspace_settings 打公司接口 | 403 |
| T5 | 任务详情模型下拉走 summary | URL 含 view=summary |
| T6 | 设置页带正确 header | 保存/加载成功 |
