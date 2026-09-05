# 测试意图：同工作空间同镜像调用人员闲置复用

**Status:** superseded（ADR-0013）

原 T1–T4（跨任务 reuse / dequeue）不再作为产品验收。替代验收：

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1' | 同 workspace 闲置评论 CSC | start-vm-auto 新评论 | 不 reuse；源 instance 仍绑在源评论 |
| T2' | 策略白名单不含本次授权 | start-vm-auto | 403 |

## 自动化落点

- `taskCloudService/src/compute_start_vm_idle_reuse_test.go`
