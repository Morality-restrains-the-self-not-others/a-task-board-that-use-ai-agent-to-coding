# deploy_pin.go Companion

`LoadReleases` / `SyncPinnedArtifacts` 是部署主机「下载后替换」路径（ADR-0027 / ADR-0052）。

- **禁止**在本文件调用 `go build` 或其它源码编译。
- 拉取失败必须保留 last-good 二进制与 `.sha` sidecar。
- 日志可打服务名与 sha，禁止打 Packages token / `Authorization`。
- `releases.yaml` 不含生产密钥；密钥只在主机 `config.local.yaml` / KMS。
- Archive pin：`dest` + `unpack: tar.gz`；sidecar 写 `artifacts/<name>.sha`。ELF 仍落 `bin/<name>`。
