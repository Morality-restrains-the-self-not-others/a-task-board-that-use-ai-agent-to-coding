# test_precise_compile.py Companion

隔离 `META_ROOT` / `PRECISE_COMPILE_DEST`。假 `build.sh` 只写 `bin/taskAuth`，禁止真 `go build`。断言登记解析、别名、空登记失败、归集 `deploy-binaries/`、跳过 releases.yaml SHA。
