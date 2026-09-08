# taskGateway (APISIX)

Monorepo edge API gateway. Browser `apiBaseUrl` points at `https://<host>:8443`.

## Commands

```bash
bash run.sh setup-tls    # dev self-signed cert
bash run.sh routes-apply # routes/routes.yaml → apisix/apisix.yaml
bash run.sh start
bash run.sh stop
```

Config: `conf/task-gateway/config.yaml`, routes: `routes/routes.yaml`.

## Compose 项目名隔离（OPT-20260901-005）

`DEPLOY_MODE=1` 或 `DEPLOY_ROOT` 存在（clone-run / 部署根）时，`run.sh` 使用
`COMPOSE_PROJECT_NAME=taskgateway-deploy`（容器名 `taskgateway-deploy-apisix-1`），
与源码仓默认的 `taskgateway` 隔离，避免同机双根时 `compose down` 误拆另一棵树的容器。
显式设置 `COMPOSE_PROJECT_NAME` 始终优先。

⚠️ 同机双根时 **:18081 仍只能有一方发布**（两边端口映射相同），项目名隔离只避免误拆容器。

## ⚠️ 路由变更部署规范（bind-mount inode 陷阱）

`apisix/apisix.yaml` 以 bind mount 挂载进 APISIX 容器。**在宿主机用
`git checkout` / 原子写（rename/`cp -f`）直接替换该文件后，容器内的挂载点仍指向旧 inode**，
`docker exec apisix reload` 只会重载旧内容 —— 表现为「路由未生效」或「上游地址还是旧值」
（此前排障即因该陷阱在容器内看到 127.0.0.1 上游旧配置）。

**规范：改动 `routes/routes.yaml` 后统一走**：

```bash
bash run.sh routes-apply   # 内容式新鲜度检查（非 mtime）+ 重生成 + reload
# 或（必要时，重挂载整个 compose）：
bash run.sh reload         # routes-apply + compose restart apisix
```

不要手动 `git checkout -- apisix/apisix.yaml` 后 `docker exec apisix reload`。
验证方法：`routes.yaml` 变更后，容器内 `md5sum /usr/local/apisix/apisix.yaml`
应与宿主机 `md5sum apisix/apisix.yaml` 一致。

## API docs (dev only)

Set `docs.enabled: true` in config (or `TASK_GATEWAY_DOCS_ENABLED=1` for codegen), then:

```bash
bash run.sh routes-apply && bash run.sh start
open https://<host>:8443/gateway/docs/
```

## License

本仓库以 GNU Affero General Public License v3.0 授权，见 [LICENSE](./LICENSE)。
```
