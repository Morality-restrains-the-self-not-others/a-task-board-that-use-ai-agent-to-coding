# [运行时] 任务详情 clone-log 等报错「Invalid or missing access token」

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-18
- 最后修改：2026-08-14（exchange-refresh 幂等返回已有 refresh）
- 维护者：Trae AI 团队

## 现象

- 页面：`…/task-detail/<task>/` 执行区 / 项目文件树相关请求
- 可见错误：`Invalid or missing access token`（`p.text-xs.text-red-600`，onlineServiceJS `auth.mjs` 401 `detail`）
- 网关日志：`cloud_resolve` 200，但上游 `repos/clone-log`、`layers/.../files`、`jobs/.../steps` 等全部 `upstream_status:401`
- 时间特征：此前长时间 200，在约 1 秒内翻转为全线 401；credential 侧同秒出现 `exchange-refresh` 失败，之后该任务不再有 `validate-token: OK`（心跳未再调度）

## 根因

1. 容器重启后仍注入**首次预埋** `ACCESS_TOKEN`；该 access 在首次 `exchange-refresh` 后已被 credential **清空**。
2. 云主机 UserData 常 `docker rm` 同名容器再 `docker run`，**不保留** `container_refresh_token.json`；新容器再调 `exchange-refresh` 得到 403 `TOKEN_EXCHANGE_ALREADY_DONE`，且无落盘 refresh → fail-closed → 任务详情「容器通信 idle」。
3. 重启换票再调 `exchange-refresh` 时（历史）：
   - credential 原返回 **401** `TOKEN_ACCESS_INVALID`（access 查无）；
   - onlineServiceJS **仅**在 **403**（已换票）时回退 `refresh-access`，401 直接失败。
4. 非 strict 启动下换票失败 → 旧行为：`bootstrapCtx.skipped=true`，**不** `register-reachability`、**不**启心跳，但 HTTP 仍监听并接受受保护 API，进程带着**无效** `ACCESS_TOKEN`。新行为（OPT-20260718-056）：同场景 **fail-closed**——`/api/*`、`/ui/*` 与 `/api/session/ui-redirect` 返回 **503** `TOKEN_BOOTSTRAP_FAILED`；公开 `GET /healthz` 亦为 503；仅 intentional skip（无 TaskApi 前缀）仍可正常提供本地 API。
5. 平台 `container-target` / by-scope 持有**当前有效** access，转发时容器一律 401（旧行为）或上游 503 fail-closed（新行为，避免「假活」）。
6. 次因：403 `TOKEN_EXCHANGE_ALREADY_DONE` 的 detail 原先不含 `refresh-access` 字样，而前端判定 `isExchangeRefreshForbiddenError` 依赖该子串，导致即使用有效 access 撞「已换票」也可能无法回退。

## 解决方案

1. **taskCredentialService**：`ExchangeRefresh` 在调用方能证明持有预埋/当前 access（`token_issued`/`exchange_refresh` 审计哈希，或仍在库的 access 明文）且 scope 已有 refresh 时，**幂等返回已有 refresh_token**（200）。未知 access 仍 403 `TOKEN_EXCHANGE_ALREADY_DONE`，不泄露 refresh。UserData `docker rm` + `docker run` 同一 `ACCESS_TOKEN` 且无落盘 `container_refresh_token.json` 时，容器可凭预埋 token 自愈。
2. **onlineServiceJS**（既有）：
   - `isExchangeRefreshForbiddenError` 识别 `TOKEN_EXCHANGE_ALREADY_DONE` / 中文 detail（不单靠 `refresh-access` 子串）；
   - 新增 `isExchangeRefreshInvalidAccessError`（401 `TOKEN_ACCESS_INVALID`）；
   - `isExchangeRefreshFallbackEligibleError`：403/401 均可在有落盘 `refresh_token` 时回退 `refresh-access`。
   - 换票抛错且非 strict：`setTokenBootstrapFailed` + `authMiddleware` / UI 门禁 503（`src/auth.mjs`）；跳过 post-listen bootstrap。

## 预防

- 容器重启换票必须能凭落盘 refresh **或** 预埋 access 的幂等 exchange-refresh 自愈；禁止「换票失败仍对外提供受保护 API」（fail-closed 已落地）。
- 修改 credential 错误文案时，同步核对 onlineServiceJS 的错误分类（优先 `error_code` / `structuredPayload`）。
- 任务详情「容器通信 idle」：先看容器 `docker logs` 是否 `TOKEN_EXCHANGE_ALREADY_DONE` + `no persisted refresh_token`，再查 credential 是否已部署幂等 exchange-refresh。
- 换票成功后若进程立刻退出：查 `register-reachability` 是否 400 `缺少评论ID`。存量镜像不传 `comment_id`；`taskCloudService` `resolveInboundCommentCSC` 须能按 `public_ip` / `server_url` 主机 / 任务下唯一评论实例回退。`server.mjs` 在 reachability 失败时 `process.exit(1)`，Docker restart 会空转。
- 任务详情出现全线容器 401/503 时：先查 credential `exchange-refresh`/`refresh-access`、容器 `GET /healthz` 的 `token_bootstrap`，与心跳 `validate-token` 是否中断，再查前端。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test src/bootstrap.tokenExchange.test.mjs src/auth.failClosed.test.mjs src/server.tokenBootstrapFailClosed.test.mjs
cd taskCredentialService && go test . ./application ./interfaces -count=1
# 首次 exchange 后用同一预埋 access 再调 exchange-refresh 应 200 返回同一 refresh
# 未知 access 在已有 refresh 的 scope 上仍 403 + TOKEN_EXCHANGE_ALREADY_DONE
```

## 存量恢复

1. 部署含幂等 `ExchangeRefresh` 的 `task-credential-service`（精准编译重启）。
2. **重启**该任务的 onlineServiceJS 容器（`docker restart <id>` 即可；进程会再调 exchange-refresh 拿到已有 refresh 并落盘）。
3. 确认心跳恢复：任务详情「容器通信」非 idle；`validate-token: OK`；网关 `container-clone-log` upstream 200。
