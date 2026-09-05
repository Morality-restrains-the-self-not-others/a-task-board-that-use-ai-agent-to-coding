# step_full_config.go Companion

公开字段写 `conf/taskCloudService/step-full-cos.yaml`；`secretId` / `secretKey` 只写 `conf-local/taskCloudService/step-full-cos.yaml`。读取走 `ReadAppFragment`（已合并 conf-local）。禁止把密钥写进已跟踪 YAML。
