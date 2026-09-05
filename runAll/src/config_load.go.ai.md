# config_load.go Companion

`LoadConfig` 入口：`loadCutoverEnvForConfig`（bash source `cutover.env`）→ 读 YAML → `overlayConfLocal` → 校验。哈希只覆盖已跟踪主文件，不含 conf-local。
