# deploy_fetch.go Companion

`ParsePackageRef` / `FileArtifactFetcher` / `GitHubReleaseFetcher` 只下载预编译产物。

- 禁止 `go build`。
- HTTP `Transport.Proxy = nil`（元规则：应用不走环境代理）；下载超时 60 分钟（含 taskEvents-bin.tar.gz）。
- **禁止 HTTP/2**（`TLSNextProto` 空 map）：GitHub 对 `gh`/Go 客户端常回 `PROTOCOL_ERROR` / stream ID 1。
- 禁止日志打印 token / `Authorization`。
- `github://owner/repo/asset@tag`：`@` 后是 **Release tag**；YAML `sha` 是文件内容哈希（sidecar / 跳过重复下载），禁止拿内容哈希去查 tag。
