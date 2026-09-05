# DDD — 部署机 9999 源码编译重启（轻量）

- **Date:** 2026-09-02

## 限界上下文

仍在既有 **runAll 编排** 上下文。不新建 bounded context、不新建聚合、不引入领域事件。

意图文档已声明 **no-event**：ops 控制面按钮，非业务状态机。

## 应用服务（编排，非领域层）

`prepareDeploySourceArtifacts(names, all)`：

1. require `SOURCE_ROOT` / `DEPLOY_ROOT`
2. `precise-compile.sh`（子进程无 `DEPLOY_MODE`）
3. `rsync conf-local`
4. `install-local-artifacts`

`PreciseRestart` 在 restart 循环前调用；`BuildAll` 在 `DEPLOY_MODE` 下整段替换为上述 + 清登记。

## 端口

无新 Repository。文件系统 + `exec` 视为基础设施适配器，集中在 `source_compile_deploy.go` 以便测试替换函数变量。
