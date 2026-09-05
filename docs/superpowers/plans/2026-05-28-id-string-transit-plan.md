# Plan: ID 字符串传输

- [x] 设计文档
- [x] `taskBill/src/ids.go`：`formatID` / `parseIDField` / `stringField`
- [x] 更新 `handlers.go` 所有 JSON 出站 ID
- [x] 更新 `charge.go` internal 响应
- [x] 更新 `billing_bridge/client.py` `_json_safe_payload`
- [x] 更新 `task_post_billing.py` / `server_start_billing.py` / `paypal_recharge.py`
- [x] `go build` + curl 回归
