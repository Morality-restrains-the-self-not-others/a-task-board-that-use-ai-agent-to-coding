# saascontainer 运行时嵌入副本

人类编辑 SSOT 仍是 `docs/skills/saas-container/`（docs 子仓）。本目录是 **taskAiProvider 二进制内嵌副本**：生产 `daydaymoney-deploy` 没有 docs 子仓，进程不能依赖 `RepoRoot/docs/skills/...`。

改契约时：先改 docs SSOT，再把同名文件复制到这里，跑 `TestEmbeddedCatalogMatchesDocsSSOT`。
