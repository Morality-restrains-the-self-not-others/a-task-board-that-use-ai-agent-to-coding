# 测试意图：厂商申请证照与联系方式验证

| ID | 场景 | 期望 |
|----|------|------|
| T1 | upload 合法 jpeg | 200 + file_key |
| T2 | upload 超限/非法类型 | 400 |
| T3 | 申请缺证照 | 400 |
| T4 | 申请未短信验证 | 403 |
| T5 | 完整申请 | 200 pending + 列落库 |
| T6 | rejected 重提带新证照 | review_note 清空 |
| T7 | staff 下载证照 | 200 |
| T8 | 非 staff 下载 | 401/403 |
| T9 | FE 表单：无验证不可提交 | 按钮 disabled |
| T10 | FE 表单：齐全后 body 含 keys + phone | postBody 断言 |
