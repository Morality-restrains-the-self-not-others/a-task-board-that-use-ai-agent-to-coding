# [运行时] Daydaymoney Grafana 插件登录 `Failed to fetch`（CORS OPTIONS 无 ACAO）

## 现象

打开 `http://183.250.1.132:3000/plugins/daydaymoney-grafana-app`，填写 API 基地址（`https://www.daydaymoney.com` 或 `http://183.250.1.132:18081`）、账号与 `at_` 访问令牌后点「登录」，界面提示：

`登录失败：Failed to fetch`

浏览器 Network 中对 `/api/accounts/users/login-with-access-token/` 的 **OPTIONS** 预检返回 200，但响应头**没有** `Access-Control-Allow-Origin`；实际 POST 可能已被浏览器拦截而未发出。

## 根因

1. Grafana 插件在浏览器内用 `fetch(..., mode: 'cors')` 直连 API（非同源）。
2. 访问 Grafana 的 Origin 为 **`http://183.250.1.132:3000`**（经 `INFRA_HOST` IP），与白名单中的 `http://localhost:3000` / `http://127.0.0.1:3000` **不同源**。
3. APISIX `cors` 插件仅对 `allow_origins` 内的 Origin 在 OPTIONS 上回写 ACAO；未命中时预检无 CORS 头 → 浏览器抛出 `TypeError: Failed to fetch`。
4. 上游对实际 POST 偶发仍可能回显 Origin，但预检失败时浏览器不会继续 POST。

## 解决方案

1. 在 `conf/gateway/task-gateway/config.yaml` 的 `cors.allowedOrigins` 使用模板（勿硬编码 IP）：

   ```yaml
   - http://${INFRA_HOST}:3000
   ```

   `${INFRA_HOST}` 解析顺序：环境变量 `INFRA_HOST` → `conf/infra/docker-infra/config.yaml` 的 `infraHost` → `localhost`。

2. 换 IP 时只改 `infraHost`（或导出 `INFRA_HOST`），然后执行：

   ```bash
   bash taskGateway/run.sh routes-apply
   ```

   （会 codegen `apisix.yaml` 并 `apisix reload`。）

3. 验证预检：

   ```bash
   curl -sS -D - -o /dev/null -X OPTIONS \
     'https://www.daydaymoney.com/api/accounts/users/login-with-access-token/' \
     -H 'Origin: http://'"${INFRA_HOST:-183.250.1.132}"':3000' \
     -H 'Access-Control-Request-Method: POST' \
     -H 'Access-Control-Request-Headers: content-type'
   # 期望含 Access-Control-Allow-Origin: http://<INFRA_HOST>:3000
   ```

## 预防

- 换 INFRA IP 时只改 `conf/infra/docker-infra/config.yaml` 的 `infraHost`（或导出环境变量 `INFRA_HOST`），再 `bash taskGateway/run.sh routes-apply`；勿在 `apisix.yaml` 或 CORS 列表里硬编码 IP。
- 本地仅用 `http://localhost:3000` 联调时不会暴露本问题，公网/INFRA_HOST 验收必做 OPTIONS 对照。
- 插件登录失败若文案含 `Failed to fetch`，优先查 CORS 预检而非账号/令牌本身（`dist` 内已带 Origin 提示）。

## 验证

- OPTIONS（上节）返回 ACAO。
- 控制台重新登录成功，显示「已登录」；捕获规则可选择公司/工作空间/责任人。
