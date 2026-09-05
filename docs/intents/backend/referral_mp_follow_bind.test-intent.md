# 测试意图：服务号关注绑定（动态 scene 票）

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 签名正确 GET | 200 且 body=echostr |
| T2 | 签名错误 POST | 非 success |
| T3 | subscribe 已知 unionid 无 scene | 同一 user_id 出现 app_key=mp 行（v112 旁路） |
| T4 | subscribe 未知 unionid 无 scene | pending 一行；user 数不变 |
| T5 | 重放同一 openid | upsert 幂等，不双写冲突 |
| T6 | follow-status 用户 unionid=pending | bound=true 且 pending 删除 |
| T7 | follow-status 无 unionid 无 mp | bound=false |
| T8 | SCAN EventKey=temp_id 命中票且 unionId 空闲 | 票用户获得 mp 别名；票 status=bound |
| T9 | subscribe EventKey=qrscene_temp_id 且 unionId 属他人 | 不抢绑；票 conflict；follow-status 有 conflict_code 且 body 无他人 id |
| T10 | POST follow-qr 缺 Idempotency-Key | 400 |
| T11 | POST follow-qr 未登录 | 401 |
| T12 | 复用未过期 pending 票 | 不第二次调用微信创码 |
| T13 | follow-status pending 票且粉丝 `qr_scene_str`=temp_id | bound=true；票 bound；不绑其他粉丝 |
| T14 | follow-status pending 票但粉丝 scene 不匹配 | bound=false；票仍 pending |
| T15 | follow-status scene 命中但 mp openid 属他人 | 不抢绑；ticket_status=conflict；body 无他人 id |
| T16 | Encrypt XML + 正确 encodingAESKey | 解密后按明文 subscribe 绑定 |

可执行：`taskAuth/src/auth_wechat_mp_test.go`、`taskAuth/src/auth_wechat_mp_followers_test.go`、`taskAuth/domain/wechat_mp_subscribe_test.go`
