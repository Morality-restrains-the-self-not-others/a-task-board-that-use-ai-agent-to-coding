# 实施计划：AiMonitor managed detach 修复

**日期**: 2026-06-02

## Tasks

- [x] **Task 1**: `AiMonitor/run.sh` — `managed` 改用 `compose up -d --remove-orphans`，移除前台 trap
- [x] **Task 2**: 远程验证 `bash scripts/runall-remote-docker.sh stack up aimonitor`
- [x] **Task 3**: 探活 `http://${INFRA_HOST}:3000/api/health`
- [x] **Task 4**: `cd runAll && go test ./src/... -count=1`
- [x] **Task 5**: AiMonitor 仓库 commit（PR 需 gh auth + remote）

## 验证命令

```bash
bash scripts/runall-remote-docker.sh stack up aimonitor
curl -sf "http://172.20.10.7:3000/api/health"
cd runAll && go test ./src/... -count=1
```
