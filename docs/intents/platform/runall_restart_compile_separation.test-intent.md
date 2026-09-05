# Test Intent: 全部重启只拉起 last-good 二进制（编译与重启分离）

## 测试目标

证明 restart/start 路径不编译工作树；task-events start 不隐式 `go build`；精准编译重启在编译失败时保留旧进程。

## 测试分层

- 单元：`runAll/src/runner_restart_all_test.go`、`runner_restart_lifecycle_test.go`、新建 compile-then-swap 测例
- 脚本：`taskEvents/run.sh` 的 start 无 bin 失败（可 `bats` 或 Go 调 bash fixture）
- 回归：既有 `TestRestartAllWithActor_StopThenStart` 仍通过，且断言无 build

## 用例矩阵

1. **RestartAll 不 build**  
   Given 服务配置了 `build_command` 且会失败（`exit 1`）  
   When `RestartAll`  
   Then start 成功（或至少不因 build 失败），`build_command` 未被执行

2. **task-events start 不 compile**  
   Given 删除 intent 二进制  
   When `run.sh start <path>`  
   Then 非零退出，stderr 不含 `go build`，且未创建新 ELF

3. **精准重启编译失败保活**  
   Given 服务 healthy，`build_command` 失败  
   When `PreciseRestart`  
   Then 状态仍 healthy（或 restored），PID 不变，登记保留

4. **精准重启编译成功切进程**  
   Given 服务 healthy，`build_command` 成功写出新 bin  
   When `PreciseRestart`  
   Then 新 PID，健康检查通过

5. **start 缺 bin 有明确错误**  
   Then 错误信息含 build / 精准编译重启，不含 `undefined:`

## 数据与环境

- 临时 working_dir + fake `build_command`/`start_command`；不依赖现网 :9999
- taskEvents 测例可用空 `GOPATH` 包装探测是否调用 `go`

## 通过标准

```bash
cd runAll && go test ./src/ -count=1 -run 'RestartAll|PreciseRestart|CompileThenSwap|StartDoesNotBuild'
# taskEvents start 无 bin（实现后补具体脚本名）
```

全部 PASS。本意图无领域事件；测试不断言 MQ publish。
