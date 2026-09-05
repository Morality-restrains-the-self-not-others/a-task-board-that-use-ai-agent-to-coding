# config_conf_local.go Companion

`LoadConfig` 在反序列化前把 `conf-local/<basename>` 按下标深合并进 `conf/runAll.yaml`（env 机密）。禁止把密钥打进日志。
