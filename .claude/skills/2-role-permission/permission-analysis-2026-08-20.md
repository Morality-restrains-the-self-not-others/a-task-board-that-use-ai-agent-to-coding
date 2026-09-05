# 角色权限分析：ztree 执行日志服务端持久化

**日期：** 2026-08-20  
**对应设计：** `docs/superpowers/specs/2026-08-20-ztree-exec-log-server-persist-design.md`  
**结论：** ✅ 绿灯 — 无新角色；沿用容器入站 token + 认证用户 compute 读；须强制查询键含 workspace+task+comment 防 IDOR

---

## 1. 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `POST .../layer-graph-push`（既有） | 容器（持有 comment-scoped access_token） | Task / Comment | write 快照 | `validateContainerAccessToken` + `containerPathMatchesScope` + 评论级 CSC | ✅ 充分 | 落库键必须用 token/CSC 上的 workspace/task/comment，禁止 body 覆盖跨任务 |
| `GET .../container-layer-graph`（改为 Cloud 读库） | 已认证工作区成员 | Task / Comment | read | 网关会话 + compute 路径 tenant/workspace/task/comment | ⚠️ 实施时补齐 | 查询必须 `WHERE workspace_id=? AND task_id=? AND comment_id=?`；缺任一键 400；禁止只靠 comment_id / job_id |
| `GET .../container-job-execution-log` | 已认证工作区成员 | Task / Comment | read | 已有 Cloud hydrate | ✅ 保持 | 本期不改权；查询侧已有 task+comment |
| `GET .../container-clone-log` | 已认证用户 | 容器进程 | read | Gateway→容器（不变） | ✅ 不改 | 关容器失败为预期 |
| Kafka `LayerGraphSnapshotPersisted` | Cloud 进程 | 领域事件 | publish | 无自动消费者 | ✅ | 禁止为 DLT 注册消费者 |
| 表 `cloud_layer_graph_snapshot` | 仅 taskCloudService | 库 `task_cloud` | C/U/R | 单服务所有权 | ✅ | 他服务禁止直连 |

### 1.1 权限差量

- **无新角色 / 无新 region / 无新 page ACL。** 任务详情 ztree 仍走既有工作区任务页。
- **写：** 仅容器入站（token 与 URL 租户/工作空间/任务一致）。
- **读：** 与现网 compute GET 同源（登录会话 + 路径上的 workspace/task/comment）。关容器后仍可读快照，**不扩大**到无工作区权限的用户。

---

## 2. 新增角色/权限建模

**不适用。** 不引入新角色、不新增 tenant `page:`/`region:`、不改 PDP。

租户控制台矩阵：本增量不触及租户 settings/billing 页面，无需登记新 page/region。

---

## 3. 安全审查结论

| 检查项 | 状态 | 说明 |
|--------|------|------|
| **IDOR** | ⚠️ 实施门禁 | 快照按评论一行。必须用 `(workspace_id, task_id, comment_id)` 定位；禁止 `GET ?comment_id=` 无 workspace/task |
| **权限提升** | ✅ | 容器 token 不能写其他 task（pathMatchesScope）；浏览器不能 PUSH |
| **跨租户泄露** | ✅ | company_id 落库审计；查询以 workspace+task+comment 为准 |
| **403 vs 404** | ✅ 沿用 compute | 无会话走网关 401；有会话但键不全 400；无行返回空图 200（避免用 404 探测评论是否存在树） |
| **user_id 注入** | ✅ 不适用 | 接口不接受他人 user_id |
| **敏感操作审计** | ✅ | UPSERT 打 INFO（workspace/task/comment 指纹，不打整棵 graph） |

**风险评级：** 中（IDOR 若漏 workspace 过滤）→ 用唯一键 + 强制 WHERE 缓解。克隆日志不落库，无新增 PII 面。

---

## 4. 需新增的权限测试用例

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 合法容器 push | 匹配 scope 的 token | layer-graph-push | 200，本评论一行 |
| GET 缺 workspace 或 task | 已登录 | GET layer-graph | 400 |
| GET 他工作区同 comment_id | 已登录（路径 ws-B） | GET | 200 空图，不得返回 ws-A 快照 |
| 无 token push | 匿名 | POST push | 401/400（既有 token 校验） |

---

总结清单：
- 新角色：不做
- IDOR：查询三键强制
- clone-log：权限与存储均不改
