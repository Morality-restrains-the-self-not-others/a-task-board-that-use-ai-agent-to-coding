# runAll 源码/产物分离 + build.sh

**Date:** 2026-05-19  
**Status:** implemented

## Summary

将 `runAll` 的 Go 源码迁入 `src/`，编译产物输出到 `bin/runAll`，并提供 `build.sh`、`test.sh`、`build_test.go`，与仓库内 `valueStream/` 布局一致。

## Layout

```text
runAll/
├── src/           # *.go + status.html (go:embed)
├── bin/           # runAll binary (gitignored)
├── build.sh
├── test.sh
├── build_test.go
├── run.sh         # build.sh + bin/runAll
├── config.yaml
├── go.mod
└── README.md
```

## Build

```bash
cd runAll && ./build.sh
# -> bin/runAll
```

## Usage

```bash
./bin/runAll --config ./config.yaml
# or from repo root config:
./run.sh   # uses ../config.yaml
```

## Tests

```bash
cd runAll && ./test.sh
```

`go test ./...` covers `src` package tests and root `build_test.go` (invokes `build.sh`).

## Related

- `valueStream/build.sh` — same pattern
- Root `.gitignore`: `runAll/bin/`
