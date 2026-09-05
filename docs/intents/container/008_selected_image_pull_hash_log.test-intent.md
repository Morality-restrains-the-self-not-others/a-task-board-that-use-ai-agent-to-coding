# 测试意图：selected_image 启动日志镜像 hash

对应功能意图：`008_selected_image_pull_hash_log.intent.md`

## 单元测试

| 用例 | 位置 | 状态 |
|------|------|------|
| parseImageInspectIdentity | `go_relayToTrae/src/container_image_test.go` | ✅ |
| formatImageIdentityLog | 同上 | ✅ |
| PullAndRun 含 inspect + 日志 hash | 同上 | ✅ |

## E2E / CDP

| 用例 | 位置 | 状态 |
|------|------|------|
| 启动后日志含 `docker pull ok:` + `id=sha256:` | `task2app/playwright/front_project/tests/TaskDetail.relay-image-hash-log-verify-cdp.mjs` | ✅ VERIFY_OK |

## 手工验收

1. 打开任务详情 `?relayToTrae=true`，点「启动」
2. 容器/侧车日志应出现类似：`[relayToTrae] docker pull ok: image=… id=sha256:… digest=…@sha256:…`
