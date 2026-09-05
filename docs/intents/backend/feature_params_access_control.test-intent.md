# Feature-Params 访问控制 — 测试意图

| 用例 | 步骤 | 期望 |
|------|------|------|
| T1 无 context | 成员 GET `/feature-params/` | 403；审计 view_mode=denied |
| T2 company_settings | Header=`company_settings` GET | 200；含 api_key；审计 full |
| T3 summary | GET `?view=summary` | 200；api_key=""；审计 summary |
| T4 错 context | Header=`workspace_settings` 打公司接口 | 403 |
| T5 workspace full | Header=`workspace_settings` GET workspace 接口 | 200 |
| T6 前端模型选项 | company source | 请求 URL 含 `view=summary` |
| T7 设置页 | 加载/保存公司参数 | 请求带 Access-Context header |
