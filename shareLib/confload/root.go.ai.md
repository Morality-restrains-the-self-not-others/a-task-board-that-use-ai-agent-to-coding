# root.go Companion

配置根解析（`FindConfigRoot` / `FindMonorepoRoot`）须遵守目录 [ai.md](./ai.md) 的 ADR-0052 约定。

- 返回值是含 `conf/` 的部署根或源码仓根。
- 新增环境变量优先级时不得跳过 `conf/base.yaml` 存在性检查。
- 不得在日志中打印 GitHub token / Packages 凭据。
