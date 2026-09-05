# 测试意图：源码机精准编译写入 deploy-binaries

## 覆盖点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | YAML 解析 task-auth | working_dir=`taskAuth`，build_command=`./build.sh` |
| T2 | 别名 `taskAuth` 选中 `task-auth` | 一条待编译记录 |
| T3 | 无 build_command（docker-mysql） | `--all` 不含该项 |
| T4 | 空登记且无参数 | 非零退出 |
| T5 | 登记 `task-auth` + 假 build.sh | dest 含新 `taskAuth` ELF 与 `MANIFEST.txt` |
| T6 | 错 pin 的 releases.yaml | 编译归集仍 0（SKIP_SHA） |
| T7 | wrapper `scripts/precise-compile.sh` | 转调 runAll 脚本 |

## 可执行

- `python3 -m pytest runAll/scripts/tests/test_precise_compile.py -q`
- `python3 -m pytest runAll/scripts/tests/test_collect_deploy_binaries.py -k skip_sha -q`

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 初版 |
