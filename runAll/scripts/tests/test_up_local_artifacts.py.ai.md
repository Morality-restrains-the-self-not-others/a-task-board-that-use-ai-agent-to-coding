# test_up_local_artifacts.py Companion

断言 `up.sh` 从 clone 根 `artifacts/` 或 `deploy-binaries/` 安装 ELF 并解包已知 tarball（含 `taskAiProvider-frontend-dist.tar.gz` → `taskAiProvider/frontend/dist`），且 `SYNC_ARTIFACTS=1` 时不调用 `gh`。逃逸路径的 tarball 必须失败。`taskEvents/run.sh` 列出的 intent 若包内缺失，`install-local-artifacts.sh` 必须失败。正在执行的 ELF / worker 必须能被旁路 `mv` 替换（ETXTBSY）。无 pin 时仍须按文件名解包厂商门户 SPA。
