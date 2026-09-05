# task2app 价值流 YAML（value-stream.yaml）

**Date:** 2026-05-19  
**Status:** implemented (2026-05-19)  
**Scope:** 仓库根目录 `value-stream.yaml` + valueStream `status` 扩展  

## Summary

将 task2app（`Saas_project`）文档中的五条业务价值流映射为可执行的 `value-stream.yaml`，供 `valueStream` 服务编排 pytest。采用 **方案 3**：`steps[].status` 为 `active` 或 `planned`，同一条流内并存目标 `view_test` 与当前 `tests/` 用例。

## 用户确认


| 项         | 决定                               |
| --------- | -------------------------------- |
| 落盘方式      | 方案 3：`status: active | planned`  |
| 范围        | 5 条流、16 个 active step            |
| 文件位置      | 仓库根 `value-stream.yaml`          |
| runAll 配置 | `runAll/config.yaml`（相对 YAML 目录） |


## 价值流一览


| name               | description | planned | active |
| ------------------ | ----------- | ------- | ------ |
| user-auth          | 用户注册与认证     | 5       | 4      |
| company-management | 公司与成员管理     | 3       | 2      |
| cloud-integration  | 云平台集成与租户镜像  | 4       | 5      |
| project-workspace  | 项目与工作空间     | 4       | 2      |
| task-management    | 任务管理        | 3       | 3      |


依赖顺序（业务）：`user-auth` → `company-management` → `cloud-integration` → `project-workspace` → `task-management`。

## `status` 语义（valueStream 实现）


| status       | 启动校验 test_file | 整条流执行 | 单环节 API | UI              |
| ------------ | -------------- | ----- | ------- | --------------- |
| `active`（默认） | 必须存在           | 参与    | 允许      | 可点「测试本环节」       |
| `planned`    | 不校验            | 跳过    | 400     | 显示 planned，按钮禁用 |


- 每条流至少一个 `active` step。
- API JSON：`lifecycle`（配置）与 `status`（运行态）；planned 配置环节运行态恒为 `planned`。

## 字段命名

`<runAll服务名>.<表名>.<列名>`，首段须为 `runAll/config.yaml` 中 `services[].name`（如 `saas-backend`）。表名优先 Django `db_table`（如 `tenant_installed_images`、`accounts_customtoken`）。

## 维护约定

1. 新增 `view_test` 后：将对应 step 的 `status` 从 `planned` 改为 `active`，并修正 `test_file`。
2. 新增 `tests/test_*.py` 且代表价值流环节：增加 `active` step 或替换临时 active。
3. 与 `Saas_project/docs/testing/unit-test-cases.md`、`docs/flows/价值流/` 保持同步。

## 用法

```bash
cd valueStream && ./build.sh
./bin/valueStream --config ../value-stream.yaml --ui-port :9998
```

