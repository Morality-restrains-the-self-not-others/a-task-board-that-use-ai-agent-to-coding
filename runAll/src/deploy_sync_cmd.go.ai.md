# deploy_sync_cmd.go Companion

`runAll -command deploy-sync` 在加载 `runAll.yaml` **之前**执行，以便无源码主机也能拉钉。

- 根目录：`DEPLOY_ROOT`，否则 `dirname(dirname(config))`。
- 失败保留 last-good（见 `SyncPinnedArtifacts`）。
