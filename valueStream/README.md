# valueStream

价值流单元测试编排：读取 YAML 配置，按环节执行 pytest 测试文件，Web UI 展示通过/失败与各环节注册的数据字段。

## 目录结构

```text
valueStream/
├── src/           # Go 源码与 embed 的 index.html
├── bin/           # 编译产物（git 忽略）
├── testdata/      # 测试用配置样例
├── build.sh       # 编译脚本
├── go.mod
└── README.md
```

## Build

```bash
cd valueStream
./build.sh
```

产物：`bin/valueStream`

## Usage

```bash
# 仓库根目录完整五条价值流（planned + active）
./bin/valueStream --config ../conf/value-stream.yaml --ui-port :9998

# 精简示例
./bin/valueStream --config ../docs/examples/value-streams.example.yaml --ui-port :9998
```

`--config` **必填**。打开 `http://localhost:9998`。

## YAML

- `runall_config`：相对本文件的 runAll `config.yaml` 路径，用于校验字段名中的**应用服务名**（`groups[].services[].name`）。
- `runner.working_dir`：pytest 工作目录（相对本文件）。
- `value_streams[].steps[].test_file`：单元测试文件路径（相对 `working_dir`）。
- `value_streams[].steps[].status`：`active`（默认，可执行，启动时校验文件存在）或 `planned`（目标 `view_test`，不校验文件、不参与整条流执行、UI 不可单跑）。
- `value_streams[].steps[].fields[].name`：三段式 **`应用服务.数据库表.字段`**，例如 `saas-backend.accounts_user.email`。

示例见 [docs/examples/value-streams.example.yaml](../docs/examples/value-streams.example.yaml)。

设计说明：[docs/superpowers/specs/2026-05-19-value-stream-service-design.md](../docs/superpowers/specs/2026-05-19-value-stream-service-design.md)

## Notes

- 不启动 runAll；测试使用 `Saas_project` 的 `settings_test`。
- 需本机已安装 pytest 及 task2app 测试依赖。
- **推荐**将 `value-stream.yaml` 放在 `conf/` 目录，便于路径写作：
  - `runall_config: runAll.yaml`
  - `runner.working_dir: task2app/Saas_project`
- 完整 task2app 五条价值流见 [`conf/value-stream.yaml`](../conf/value-stream.yaml)（含 `planned` + `active` 环节）。
- 精简示例见 `docs/examples/value-streams.example.yaml`。

## API

- `GET /api/streams`：返回当前价值流与环节状态，新增影响字段：`streams[].impact_status`（`known|unknown`）、`streams[].impacted`、`streams[].impacted_steps`、`streams[].steps[].impacted`、`streams[].steps[].impact_reason`。
- `POST /api/test/step`：触发单环节测试，示例：`{"stream":"flow-a","step":"s1"}`。
- `POST /api/test/stream`：触发整条流测试，示例：`{"stream":"flow-a"}`。
- `POST /api/domain-order`：调整业务域顺序并写回 YAML，示例：

```json
{
  "domain_order": ["组织与成员", "用户与认证", "云平台与资源"]
}
```

`domain_order` 仅调整业务域之间的顺序；同一业务域内多条价值流保持原有相对顺序。接口成功后会返回最新 `streams` 数据。

`POST /api/domain-order` 在写回前会执行配置校验（与启动校验一致）；若配置无效会返回 `400` 并拒绝写盘。

## UI 拖拽改序

- 在页面每个业务域标题左侧有“拖拽”手柄，可直接调整业务域显示顺序。
- 拖拽成功后会调用 `POST /api/domain-order`，并将新顺序持久化到 `value-stream.yaml` 的 `value_streams` 顺序中。
- 刷新页面和重启服务后顺序保持一致。
- 若当前有测试正在运行，改序会返回 `409`（`a test is already running`），前端会提示并回退到服务端顺序。
- 拖拽进行中会暂停自动刷新，避免 2s 轮询重渲染导致拖拽被打断；拖拽结束后恢复正常刷新。
- 拖拽改序不会改变价值流内 `steps` 顺序，也不会改变 `failed_steps` 的 YAML 顺序语义。

## Tests

```bash
cd valueStream
./test.sh
```

或分步：

```bash
go test ./... -race -cover   # src 包 + build_test.go（会执行 build.sh 冒烟）
./build.sh
```

`build_test.go` 会调用 `build.sh` 并断言 `bin/valueStream` 已生成；`--config` 缺失时二进制应以非零退出。

覆盖 spec 关键行为：三段式字段校验、runAll 服务名白名单、整条流失败继续、单环节不改变流级状态、API 返回 `fields`、409/400、embed UI、`src/` 编译布局。

## License

本仓库以 GNU Affero General Public License v3.0 授权，见 [LICENSE](./LICENSE)。不附带 AGPL 义务的专有许可见 [COMMERCIAL.md](./COMMERCIAL.md)。
