# Test Intent: 精准编译重启前重载 runAll.yaml

## 测试目标

证明精准编译重启会读取磁盘 `runAll.yaml` 补齐内存中缺失的服务，且不会把真正未知的登记名当成已配置服务。

## 测试分层

- 单元：`runAll/src/precise_restart_reload_test.go`、`runAll/src/status_test.go`
- 回归：既有 `TestPreciseRestart_UnknownServiceRetainedInFile` 仍通过（无 cfgPath 时不重载）

## 用例矩阵

1. **EnsureNames 不覆盖已有状态**  
   Given 服务 a 已为 healthy  
   When `EnsureNames(["a","b"])`  
   Then a 仍为 healthy，b 为 pending

2. **磁盘新服务可解析**  
   Given 内存配置只有 `mem-old-svc`，磁盘 YAML 含 `disk-new-svc`  
   When `reloadConfigFromDisk`  
   Then `resolveRegisteredServices("disk-new-svc")` 返回该服务

3. **过期内存不再报 unknown**  
   Given 登记 `disk-new-svc`，内存配置无此名，磁盘 YAML 有可启动条目  
   When `PreciseRestart`  
   Then 错误列表不含 `unknown service`，登记文件在成功后清空

4. **真未知名仍保留**  
   Given cfgPath 指向不含 `ghost-svc` 的 YAML，登记 `ghost-svc`  
   When `PreciseRestart`  
   Then keep 含 `ghost-svc`，错误含 `unknown service`

5. **非法 YAML 不覆盖内存**  
   Given cfgPath 指向非法 YAML，内存已有 `mem-old-svc`  
   When `reloadConfigFromDisk`  
   Then 返回 error，且仍能解析 `mem-old-svc`

## 数据与环境

- 临时 YAML + httptest 健康探针；`RUNALL_PRECISE_RESTART_FILE` 指向临时登记文件
- 不依赖现网 :9999

## 通过标准

```bash
cd runAll && go test ./src/ -count=1 -run 'EnsureNames|ReloadConfigFromDisk|PreciseRestart_Reloads|PreciseRestart_Unknown'
```

全部 PASS。

## 业务意图 → 事件对照

本意图无领域事件；测试不断言 MQ publish。
