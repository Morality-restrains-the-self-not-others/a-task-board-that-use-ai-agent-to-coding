# 测试意图：PeopleManage 成员 Git 身份

| ID | 场景 | 断言 |
|----|------|------|
| F1 | 点击 Git 身份 | 请求 tenant/member API |
| F2 | 新增成功 | 列表刷新含新项 |
| F3 | API 失败 | 中文错误 + data-traceId |
