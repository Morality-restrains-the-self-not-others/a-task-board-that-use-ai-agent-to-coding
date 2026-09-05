# Test Intent: 公网 taskFE 由 nginx 静态常驻

| ID | 用例 | 期望 |
|---|---|---|
| T1 | `docker ps` `taskfe-nginx` + `ss` `:4000` | 容器 Up；宿主机 `0.0.0.0`/`*` 监听（docker-proxy），不是 vite preview |
| T2 | `GET /user/<id>/profile/` 经网关 | 200，`text/html` |
| T3 | 构建进行中再 `GET /` | 仍 200（容器 Id 可不变） |
| T4 | `GET /static/assets/does-not-exist-<token>.js` | **404**，body 不是 `index.html` |
| T5 | `GET /health` | 有 `public/html/index.html` → 200 JSON `ok: true`；否则 503 |
| T6 | 仅 `runall-lifecycle.sh build` | symlink 指向新 release；容器 **未 recreate** |
| T7 | compose `ports` / conf `host` | 发布到 `0.0.0.0`（SSOT `conf/frontend/vue/config.yaml`） |
| T8 | 容器内 `html` | 仍为 symlink（`readlink`），不是 start 时钉死的目录 |
