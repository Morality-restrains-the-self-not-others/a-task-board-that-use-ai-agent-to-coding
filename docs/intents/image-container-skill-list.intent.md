# 功能意图：镜像容器技能列表绑定与引用

## 意图

厂商上传容器镜像时，平台解析 `/app/imageSkills.yaml` 并绑定到该镜像版本。租户在镜像市场查看技能列表；创建任务时在描述中引用并高亮 `/技能`；评论 `@镜像` 后可指定 `/{技能}`，未指定则使用列表第一项（默认技能）。

## 参与者

- 厂商（provider.daydaymoney.com 登记镜像）
- 租户成员（镜像市场 / 创建任务 / 评论）
- taskAiProvider、taskCloudService、taskTaskService、taskFE、容器进程

## 成功路径

1. 镜像 COPY `/app/imageSkills.yaml` → 厂商保存 image_url → 抽取 status=ok，第一项 default
2. 市场卡片展示技能名（默认有标记）
3. 创建任务选镜像（含项目默认镜像已绑定、描述仍空）→ 点击技能 chip 将 `$镜像 /技能` 完整 mention 写入描述且不解绑；亦可手动输入 `/name` 高亮
4. 评论 `@镜像 /name` → mentions.skill 落库；未写 `/` 则 default
5. 启动容器环境含 `IMAGE_SKILL`

## 失败路径

- 文件缺失 → extract_status=not_found，镜像仍可上架
- YAML 非法/重名 → failed，列表空，日志 `event=ContainerImageSkillsExtracted`
- `/未知技能` → 不高亮；评论提交若 skill 不在列表则 400

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 抽取并绑定镜像技能列表 | ContainerImageSkillsExtracted | taskAiProvider `runImageSkillsExtract` | 无自动消费者；catalog/安装快照读库 | — |

## 变更记录

- 2026-08-27：创建任务技能 chip 点击改为写入完整 `$镜像 /技能` mention（此前仅插入 `/技能`，空描述时会解绑已选镜像）。
