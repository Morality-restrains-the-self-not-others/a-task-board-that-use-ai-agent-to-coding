# 测试意图：自建 GitLab 平台可达性检测（内网豁免）

## 测试目标

覆盖平台探测公网 GitLab 不可达时的故障展示，以及标记内网后不把不可达当故障。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 领域 | `ClassifyReachability`：未配置 / 内网跳过 / 需探测 |
| 领域 | `InterpretProbeError` / HTTP 响应 → reachable |
| HTTP | GET reachability 内网 skip；公网 unreachable；忽略 `url` 查询 |
| HTTP | PUT `intranet` 往返；GET 连接含 `intranet` |
| 前端 | 未标内网 + unreachable 红字；标内网不报红字；勾选框 |

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 未配置 | status=`unconfigured`，不发 outbound HTTP |
| T2 | intranet=true | status=`skipped_intranet`，不发 outbound HTTP |
| T3 | httptest 200/401/302 | status=`reachable` |
| T4 | 连接超时/拒绝 | status=`unreachable` |
| T5 | `?url=http://127.0.0.1` | 仍探测已保存 base_url |
| T6 | PUT intranet true 后 GET | `intranet: true` |
| T7 | 前端 reachable=false 且非内网 | `data-status=unreachable` 含「无法连接」 |
| T8 | 前端 intranet 勾选 | `data-status=skipped_intranet`，无危险色故障文案 |
| T9 | 保存 PUT body 含 `intranet` | 请求 JSON 含该字段 |

## 数据与环境

- Go：`testApp` MySQL 夹具表含 `intranet` 列。
- 前端：jsdom + mock `apiFetch`。
- 禁止测例打真实 `115.29.110.74`。

## 通过标准

上述用例全绿；`gofmt`/`go test` 与 Vitest 相关文件通过。
