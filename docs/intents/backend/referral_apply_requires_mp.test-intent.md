# 测试意图：申请须服务号绑定

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 无 mp 别名 apply | error=`service_account_not_followed` |
| T2 | 有 mp 别名 apply | 与原 pending/approved 行为一致 |
| T3 | status 无 mp | `service_account_bound=false` |
| T4 | status 有 mp | `service_account_bound=true` |

可执行：`taskReferral/src/referral_mp_bind_gate_test.go`
