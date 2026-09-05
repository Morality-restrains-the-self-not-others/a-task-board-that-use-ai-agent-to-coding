# Plan: relay status-push csc_* 主键兼容

日期：2026-07-09

## Tasks

- [x] 确认 `_cfg_snapshot_from_model` 使用 `pk=str(cfg.pk)`（源码已修）
- [x] 单测 `test_cfg_snapshot_pk_is_string_for_non_numeric_ids` + lookup 集成
- [x] go_relay：status push HTTP 401 → 立即 unregister
- [x] 修复 `test_relay_to_trae_status.py` 的 GoTokenValidator stub
- [x] 扩展 Playwright：启动后断言「任务引导完成」+ 无 INTERNAL_DISPATCH / 无效 token
- [x] 重启 go-relay；验证 saas-backend status-push 200；跑单测 + E2E
