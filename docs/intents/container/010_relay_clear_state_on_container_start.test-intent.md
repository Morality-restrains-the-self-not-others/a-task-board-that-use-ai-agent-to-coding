# 测试意图：selected_image 启动前清空 relay 状态残留

对应功能意图：`010_relay_clear_state_on_container_start.intent.md`

| ID | 场景 | 类型 | 预期 |
|----|------|------|------|
| T1 | `ensureSelectedImageHostStateRoot` 前存在 layers/logs/旧 token 与 sibling task 目录 | 单元 | 全部删除；仅剩当前 task 的空 `runtime/` |
| T2 | `RemoveAll` 失败（模拟权限） | 单元 | 走 `docker run --entrypoint rm` 回退并删净 |
| T3 | `TestStartSelectedImageContainer_PullAndRun` 回归 | 单元 | pull → 清残留 → run + volume mount 仍成功 |

证据：`cd go_relayToTrae && go test ./src -count=1 -run 'TestEnsureSelectedImageHostStateRoot_ClearsResidualBeforeStart|TestWipeHostStatePath_FallsBackToDockerWhenRemoveAllFails|TestStartSelectedImageContainer_'`
