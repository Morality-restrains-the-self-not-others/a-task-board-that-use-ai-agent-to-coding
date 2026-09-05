# deploy_archive.go Companion

Archive pins (`unpack: tar.gz` + `dest:`) unpack into the deploy tree. ELF pins stay `bin/<name>`.

- `dest` 必须是部署根下的相对路径；禁止 `..` 与绝对路径。
- 只解 `tar.gz`/`tgz`；相对 symlink 保留（taskFE `html` → `releases/<stamp>`）；绝对/`..` symlink 拒绝；拒绝 tar slip。
- 解压先写 `dest.new` 再原子改名；失败保留原目录（last-good）。
- sidecar 在 `artifacts/<name>.sha`，不写进解压树。
