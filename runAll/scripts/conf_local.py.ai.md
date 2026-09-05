# conf_local.py Companion

`conf-local/` 与 `conf/` 同相对路径深合并。机密只放 conf-local；本模块不打印密钥值。
`overlay_conf_file` 给只持有 `conf/...` 绝对路径的启动脚本用，禁止再 `yaml.safe_load` 绕过 overlay。
