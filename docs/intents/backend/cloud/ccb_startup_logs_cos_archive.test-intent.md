# 测试意图：评论启动日志 COS 归档

## 对应意图

`ccb_startup_logs_cos_archive.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | RenderStartupLogObjectKey 默认规则 | `workspace_{ws}/task_{task}/comment_{cmt}/startup_logs.json` |
| T2 | pathRule 含 `..` | 拒绝 |
| T3 | insert 后 Get bundle | 含该 log id |
| T4 | 同 id persist 两次 | 仍一行 |
| T5 | COS Put 失败 | insert 成功且分片行保留 |
| T6 | Put 成功后 list | 分片无该 id，仍从 COS 还原 |
| T7 | 非平台角色 GET COS | 403 |
| T8 | 平台角色 GET | 有 startupLogsPathRule，无 secretKey |

## 可执行测试

- `taskCloudService/domain/startup_log_key_test.go`
- `taskCloudService/domain/startup_log_bundle_test.go`
- `taskCloudService/src/ccb_log_cos_test.go`
- `taskCloudService/src/step_full_admin_test.go`（startupLogsPathRule）
- `taskEvents/config/config_test.go`（CommentStartupLogArchived）
- `taskFE/app/src/views/SystemAdminStepFullCOS.test.js`
