# 测试意图：ztree 层级落库与 step_full COS 归档

## 对应意图

`ztree_step_full_cos_archive.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | RenderObjectKey 默认 pathRule | `workspace_{ws}/task_{task}/comment_{cmt}/step_full.json` |
| T2 | 非法 pathRule 含 `..` | 拒绝 |
| T3 | FakeCOS Put+Get bundle 按 job_id 合并 | 两 job 都在 |
| T4 | GET 有 COS 对象 | `source=saas_cos` 且 agent_steps 来自全文 |
| T5 | GET 无对象有 023 | `source=saas_db` |
| T6 | 同 job 重放 push | 一行 UPSERT，不插出行 |
| T7 | 非平台角色 PATCH COS | 403 |
| T8 | 平台角色 GET COS | 不回显 secretKey |
| T9 | onlineServiceJS 收集 step 目录 | 读出 agent_step_full.json 数组 |
| T10 | 层图 GET 有快照 | 仍 `source=saas_db`（回归） |

## 可执行测试

- `taskCloudService/domain/step_full_key_test.go`
- `taskCloudService/src/step_full_store_test.go`
- `taskCloudService/src/step_full_handlers_test.go`
- `taskCloudService/src/step_full_admin_test.go`
- `trae-agent/onlineServiceJS/src/saasStepFullArchive.test.mjs`
- `taskFE/app/src/views/SystemAdminStepFullCOS.test.js`
