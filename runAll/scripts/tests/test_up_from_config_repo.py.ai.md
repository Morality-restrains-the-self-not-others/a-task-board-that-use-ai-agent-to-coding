# test_up_from_config_repo.py Companion

`up-from-config-repo.sh` 的 conf-local overlay、GitHub HTTP/1.1 引导、`SYNC_ARTIFACTS=1` 缺 `runAll` 硬失败。本地产物安装见 `test_up_local_artifacts.py`。测试必须 `_clean_env`（unset `DEPLOY_ROOT`）。`_seed_repo` 同时拷 `install-local-artifacts.sh`。`TASKAIPROVIDER_FRONTEND` overlay 须写入 `taskAiProvider/frontend/dist`。
