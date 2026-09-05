# UserData 安装 Docker CE 时 download.docker.com TLS reset → 容器从未启动

- **Date**: 2026-08-12
- **Symptoms**: 阿里云 ECS 上 `/root/init_from_task2app.sh.log` 在「安装容器运行时 / 使用 Docker」后中断；实例无 docker、无业务容器；平台侧服务器长期「启动中」或启动失败；verify 回调未触发。
- **Log signature**:
  ```
  curl: (35) OpenSSL SSL_connect: Connection reset by peer in connection to download.docker.com:443
  gpg: no valid OpenPGP data found.
  ```
- **Root cause**: Linux UserData 生成器（`taskAiProvider/frontend/src/utils/userDataScriptLinux.js`）仅从 `https://download.docker.com/linux/...` 拉 GPG/apt 源。国内云主机（尤其阿里云）常无法稳定 TLS 直连该域名；`set -e` 下 `curl|gpg` 失败即退出，Docker 未装、后续 pull/run 全跳过。
- **Not the cause（本次）**: `{REGISTRY_URL}` 未替换（脚本内未使用）；镜像 `registry.cn-qingdao.aliyuncs.com/...` 尚未执行到 pull；ACCESS_TOKEN / TASK_API_ENDPOINT 占位已替换。
- **Fix**: 生成器改为优先 `mirrors.aliyun.com/docker-ce` → 清华 → 官方兜底；已安装 docker 则跳过；全失败才 `exit 1`。回归：`tests/userDataBootProgress.unit.test.js`。
- **Ops（存量）**: (1) 厂商门户重新生成并保存 UserData 模板后启新实例；(2) 已挂死的实例可手工用阿里云 docker-ce 源装 docker 后重跑脚本后半段，或直接重建实例。
- **Verify**: 新实例日志应出现 `Docker CE 已从镜像安装: https://mirrors.aliyun.com/...` 并继续到 pull/run；`bash -n` 生成脚本通过。
- **Related**: OPT-20260812-057；89_userdata_container_name_shell_var_dash_inject.md
