# 测试意图：多区域 Registry 副本与同区优先拉取

## 测试目标

验证公网→内网推导、副本 Upsert/级联删除、安装快照、start-vm 选址优先级、存量回退、VPC 公网字段拒绝。

## 测试分层

- 单元：`shareLib/registryhost`（公网→内网用例表）
- 单元：`taskAiProvider` association+replica
- 单元：`taskCloudService` 安装快照 + `selectContainerImagePullRef` + start-vm
- 单元：厂商门户区域环境表单（registry_public_url / 只读 intranet）

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | `registry.cn-hangzhou.aliyuncs.com/ns/img:tag` | SuggestIntranetRegistryRef | host 改为 `registry-vpc.cn-hangzhou.aliyuncs.com`，path/tag 不变 |
| T2 | `registry-vpc.cn-hangzhou.aliyuncs.com/ns/img:tag` | 保存 replica public_url | 400 无法触及 |
| T3 | 已有 association 杭州 + public_url | POST association upsert | 一行 replica；GET 带 intranet |
| T4 | 杭州 replica 有 intranet | start-vm region=cn-hangzhou | container_image_url = intranet |
| T5 | 杭州 replica 仅 public（intranet 空） | start-vm 杭州 | 用 public_url |
| T6 | 已安装镜像 replicas=[] | start-vm | 规范 image_url:version |
| T7 | 青岛 replica、启机杭州 | start-vm 杭州 | 回退规范 URL（不用青岛副本） |
| T8 | csiID=0 清杭州 CSI | upsert | association 与 replica 均无杭州行 |
| T9 | internal image resolve | GET resolve | 规范公网，不读副本 |
| T10 | docker.io 公网 URL | 推导 | intranet 空 |

## 数据与环境

SQLite/MySQL 测试夹具；不访问真实 ACR。推导用例进入 `shareLib/registryhost/testdata/registry_cases.json`（或并列 intranet 用例文件，Go 与门户单测共用）。

## 通过标准

上表全绿；`gofmt`/`go test` 相关包通过。
