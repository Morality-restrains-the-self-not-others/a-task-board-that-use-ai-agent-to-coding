# NFR Clarification: taskFE build_command

> 配置变更，NFR 影响极小。按 auto-flow 默认 L2。

## 适用类别

| 类别 | 等级 | 说明 |
|------|------|------|
| 可用性 | L2 | 编译按钮成功率应与 go-run-container 等已有 build_command 服务一致 |
| 性能 | 不适用 | 构建耗时由 vite 决定，无新要求 |
| 安全 | 不适用 | 无新攻击面 |
| 可维护性 | L2 | 双配置文件保持同步 |

## 质量场景

### QS-01: 编译按钮成功

| 维度 | 内容 |
|------|------|
| 刺激 | healthy/stopped 的 taskFE 点击「编译」 |
| 环境 | runAll UI :9999，node_modules 已安装 |
| 响应 | 无 "no build command configured" 错误；构建完成 |

## 跳过理由

纯配置补齐，不引入新数据流或外部依赖。
