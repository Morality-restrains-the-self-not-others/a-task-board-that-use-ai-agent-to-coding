# DaydaymoneyGrafana — task2app 报警捕获 Grafana 插件

Grafana App 插件：登录 task2app、配置报警捕获规则、自动创建任务。

**Grafana 版本要求：** `>= 11.5.0`（与 AiMonitor `grafana/grafana:11.5.2` 对齐）。

详细使用说明见 [USAGE.md](./USAGE.md)。

## 服务 IP / 地址配置

生产环境 task2app API 网关为 **`http://<gateway-host>:18081`**（`:4000` 为 Vue 前端，不代理 `/api`）。

按部署方式修改下列文件中的 `task2appApiBaseUrl`：

| 场景 | 配置文件路径 | 说明 |
|------|-------------|------|
| **AiMonitor 一体栈（推荐）** | [`../AiMonitor/grafana/provisioning/plugins/apps.yaml`](../AiMonitor/grafana/provisioning/plugins/apps.yaml) | **勿** provisioning 本插件（文件内 `apps: []`）；否则每次 Grafana 重启会覆盖 `jsonData` 并清掉登录态 |
| **插件独立 Docker 开发** | [`provisioning/plugins/apps.yaml`](./provisioning/plugins/apps.yaml) | 同上，仅保留空 `apps: []` |
| **代码默认值（兜底）** | [`src/components/AppConfig/AppConfig.tsx`](./src/components/AppConfig/AppConfig.tsx) | 控制台首次打开时的 API 基地址初始值 |
| **运行时覆盖** | Grafana UI → Administration → Plugins → Daydaymoney Grafana → **Daydaymoney 控制台** | 登录与捕获规则由插件自动 `POST /api/plugins/{id}/settings` 写入 Grafana 数据库，重启后保留 |

环境默认 API 地址（写在 `AppConfig.tsx`，可按部署修改源码或登录后在 UI 改）：

```typescript
task2appApiBaseUrl: 'http://<gateway-host>:18081'
```

**为何不能 provisioning 插件：** Grafana 对 `provisioning/plugins/apps.yaml` 中列出的 app 会在**每次启动**用文件里的 `jsonData` **整体替换**数据库配置。即便只写 API 地址，也会把 `task2appSessionToken`、`captureRules` 等 UI 保存字段清掉。因此 AiMonitor 栈中该文件保持 `apps: []`。

本地开发若 API 在宿主机，可改为 `http://localhost:8001` 或 `http://host.docker.internal:8001`（视 Grafana 容器网络而定）。

**跨域（CORS）**：Grafana 控制台（`:3000`）会直接请求 API（`:18081` 或 `https://www.daydaymoney.com`）。浏览器 Origin 若为 **`http://${INFRA_HOST}:3000`**（与 `localhost:3000` 不同源），须在 [`conf/gateway/task-gateway/config.yaml`](../../conf/gateway/task-gateway/config.yaml) 使用模板 `http://${INFRA_HOST}:3000`。`INFRA_HOST` 解析顺序：环境变量 → [`conf/infra/docker-infra/config.yaml`](../../conf/infra/docker-infra/config.yaml) 的 `infraHost` → `localhost`。修改后执行 `bash taskGateway/run.sh routes-apply`。详见 `.ai/09_failure_experience/02_runtime_errors/11_daydaymoney_grafana_login_failed_to_fetch_cors.md`。

## Get started

### Frontend

1. Install dependencies

   ```bash
   npm install
   ```

2. Build plugin in development mode and run in watch mode

   ```bash
   npm run dev
   ```

3. Build plugin in production mode

   ```bash
   npm run build
   ```

4. Run the tests (using Jest)

   ```bash
   # Runs the tests and watches for changes, requires git init first
   npm run test

   # Exits after running all the tests
   npm run test:ci
   ```

5. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

6. Run the E2E tests (using Playwright)

   ```bash
   # Spins up a Grafana instance first that we tests against
   npm run server

   # If you wish to start a certain Grafana version. If not specified will use latest by default
   GRAFANA_VERSION=11.3.0 npm run server

   # Starts the tests
   npm run e2e
   ```

7. Run the linter

   ```bash
   npm run lint

   # or

   npm run lint:fix
   ```

# Distributing your plugin

When distributing a Grafana plugin either within the community or privately the plugin must be signed so the Grafana application can verify its authenticity. This can be done with the `@grafana/sign-plugin` package.

_Note: It's not necessary to sign a plugin during development. The docker development environment that is scaffolded with `@grafana/create-plugin` caters for running the plugin without a signature._

## Initial steps

Before signing a plugin please read the Grafana [plugin publishing and signing criteria](https://grafana.com/legal/plugins/#plugin-publishing-and-signing-criteria) documentation carefully.

`@grafana/create-plugin` has added the necessary commands and workflows to make signing and distributing a plugin via the grafana plugins catalog as straightforward as possible.

Before signing a plugin for the first time please consult the Grafana [plugin signature levels](https://grafana.com/legal/plugins/#what-are-the-different-classifications-of-plugins) documentation to understand the differences between the types of signature level.

1. Create a [Grafana Cloud account](https://grafana.com/signup).
2. Make sure that the first part of the plugin ID matches the slug of your Grafana Cloud account.
   - _You can find the plugin ID in the `plugin.json` file inside your plugin directory. For example, if your account slug is `acmecorp`, you need to prefix the plugin ID with `acmecorp-`._
3. Create a Grafana Cloud API key with the `PluginPublisher` role.
4. Keep a record of this API key as it will be required for signing a plugin

## Signing a plugin

### Using Github actions release workflow

If the plugin is using the github actions supplied with `@grafana/create-plugin` signing a plugin is included out of the box. The [release workflow](./.github/workflows/release.yml) can prepare everything to make submitting your plugin to Grafana as easy as possible. Before being able to sign the plugin however a secret needs adding to the Github repository.

1. Please navigate to "settings > secrets > actions" within your repo to create secrets.
2. Click "New repository secret"
3. Name the secret "GRAFANA_API_KEY"
4. Paste your Grafana Cloud API key in the Secret field
5. Click "Add secret"

#### Push a version tag

To trigger the workflow we need to push a version tag to github. This can be achieved with the following steps:

1. Run `npm version <major|minor|patch>`
2. Run `git push origin main --follow-tags`

## Learn more

Below you can find source code for existing app plugins and other related documentation.

- [Basic app plugin example](https://github.com/grafana/grafana-plugin-examples/tree/master/examples/app-basic#readme)
- [`plugin.json` documentation](https://grafana.com/developers/plugin-tools/reference/plugin-jsonplugin-json)
- [Sign a plugin](https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin)
