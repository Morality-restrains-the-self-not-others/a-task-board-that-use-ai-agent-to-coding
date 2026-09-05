# 测试意图：切换容器镜像后硬件规格与镜像架构对齐

## 对应意图

`image_switch_hardware_binding.intent.md`

## 摘要

跨架构切换项目默认镜像时，仅保存镜像必须 400；须同单提交与新架构匹配的完整运行模版。同架构切换可只改镜像。后端推断实例架构不得把 `ecs.r6.*` 当成 arm64。

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | `resolvePrimaryImageArchitecture` 在能命中已安装镜像时以 `selectedImageId` 为准 | 单元 | `containerImageArchitecture.test.js` |
| T2 | 实例架构 ∉ 镜像架构集合 → 判定不兼容 | 单元 | 抽出的兼容性纯函数 + 对应 test |
| T3 | `runTemplateMode` 换镜像后仍拉取 available-instances；不兼容则清空选中 | 单元 | `useServerConfigHardwarePanel` 相关 test |
| T4 | 项目 PATCH 新 `container_image_id` 且现模版不兼容 → 400 且 DB 未改；同单带匹配完整模版 → 200；同架构只改镜像 → 200 | 单元 | `taskProjectService` PATCH handler test |
| T5 | `inferInstanceArchitecture("ecs.r6.xlarge")==x86_64`；`ecs.r6r.xlarge`/`g8y`==arm64 | 单元 | `taskCloudService` `compute_image_resolve_test.go`（或同包新文件） |
| T6 | 项目详情：x86 模版下改选 arm 镜像并保存 → 失败提示；重选 arm 实例后保存成功 | Playwright | `ProjectDetail.image-architecture-filter.playwright.test.js` 或 `hardware-config-sync` |
| T7 | 任务详情：`@镜像` 从 arm 换 x86 后实例列表查询带新 `image_architecture` | Playwright | 现有 TaskDetail hardware / imageSelect 测扩展 |

## 成功标准

- T1–T5 本地单测全绿
- T6–T7 mock 环境下：跨架构未重选实例时镜像保存失败；匹配后成功；启动不带旧规格
