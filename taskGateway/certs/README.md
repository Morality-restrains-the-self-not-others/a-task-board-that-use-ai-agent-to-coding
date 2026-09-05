# taskGateway TLS（开发）

运行 `bash ../run.sh setup-tls` 生成本目录下的 `dev-gateway.pem` 与 `dev-gateway-key.pem`。

- 文件已 `.gitignore`，**勿提交私钥**。
- 浏览器需信任自签证书，或 Playwright 使用 `ignoreHTTPSErrors`。
