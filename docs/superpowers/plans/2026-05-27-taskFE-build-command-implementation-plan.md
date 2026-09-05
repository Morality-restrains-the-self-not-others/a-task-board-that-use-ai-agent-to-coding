# Implementation Plan: taskFE build_command

## Task 1: 配置补齐

- [ ] `runAll/config.yaml` — taskFE 增加 `build_command: "npm run build"`
- [ ] `runAll.yaml` — 同步同上

## Task 2: 回归测试

- [ ] `runAll/src/config_test.go` — `TestProductionConfig_VueFrontendHasBuildCommand` 加载真实 config 断言 build_command 非空

## Task 3: 验证

- [ ] `cd runAll && go test ./...`
- [ ] 手动：UI 编译按钮（需 reload runAll）
