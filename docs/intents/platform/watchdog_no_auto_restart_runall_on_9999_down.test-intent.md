# Test Intent: :9999 不可用时看门狗不自动重启 runAll

## 测试目标

证明 watchdog 在 runAll UI 失联时默认不 spawn runAll；禁止文件可挡住 opt-in；crontab 行带 `--no-restart-runall`。

## 测试分层

- 单元：`runAll/scripts/tests/test_ensure_services_healthy.py`

## 用例矩阵

1. **默认不拉起**  
   Given `:9999` 探测失败  
   When `recover_runall(..., restart 默认)`  
   Then 不调用 `spawn_runall`，返回含 `DOWN`

2. **禁止文件挡住 opt-in**  
   Given `.runall/no_restart_runall` 存在且 `restart=True`  
   When `recover_runall`  
   Then 不调用 `spawn_runall`

3. **opt-in 仍可恢复**  
   Given 无禁止文件、`restart=True`、冷却允许  
   When `recover_runall`  
   Then 调用 spawn + start-all（既有 full_flow）

4. **crontab 行**  
   When `watchdog_cron_line(root)`  
   Then 含 `ensure_services_healthy.py --no-restart-runall`

## 数据与环境

- 不依赖现网 `:9999`；`runall_ui_up` / `spawn_runall` 由测例 monkeypatch

## 通过标准

```bash
python3 -m pytest runAll/scripts/tests/test_ensure_services_healthy.py -q
```

全部 PASS。
