# 测试意图：解析镜像架构时提示云厂商内网 Registry 不可达

## 测试目标

验证 VPC/内网 Registry 被立即拒绝，错误文案含「无法触及」，公网地址不受影响。

## 测试分层

- 单元（前端）：`taskAiProvider/frontend/tests/privateRegistryHint.unit.test.js`
- 单元（Go 共享包）：`shareLib/registryhost/registryhost_test.go`
- 单元（Go）：`taskAiProvider/infrastructure/private_registry_test.go`、`taskAiProvider/src/vendor_resolve_handlers_test.go`
- 单元（Go）：`taskCloudService/src/private_registry_test.go`

> host 分类与 Aliyun 公网映射由共享用例表 `shareLib/registryhost/testdata/registry_cases.json`
> 驱动：Go `registryhost_test.go` 与前端 `privateRegistryHint.unit.test.js` 消费同一文件，
> 两实现任一漂移即红（OPT-20260818-012）。

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 阿里云 VPC 超时原文 / image_url | `isCloudVendorPrivateRegistryText` / `enrichResolveArchitectureError` | 判定为内网；文案含无法触及与 `registry.cn-qingdao.aliyuncs.com`；不重复附加 |
| T2 | RFC1918 仓库地址 | 检测函数 | 判定为内网 |
| T3 | 公网 ACR / docker.io / ghcr.io | 检测函数 | 不判定为内网 |
| T4 | `registry-vpc.cn-qingdao.aliyuncs.com/...` | `ResolveContainerImageMetadata`（panic client） | 不发起 HTTP；错误含无法触及 |
| T5 | 同上 | `POST .../resolve-target-architectures/` | 2s 内 400，body 含无法触及 |
| T6 | taskCloudService 同样 URL | resolve + HTTP handler | fail-fast 400，含无法触及 |

## 数据与环境

- 不访问真实阿里云；Go 侧用 panic HTTP client / 不拨号断言 fail-fast。

## 通过标准

```
cd shareLib/registryhost && go test -count=1 ./...
cd taskAiProvider/frontend && npm run test:unit -- tests/privateRegistryHint.unit.test.js
cd taskAiProvider && go test ./infrastructure/ ./src/ -count=1 -run 'PrivateRegistry|ResolveTargetArchitecturesRejects|ResolveContainerImageMetadataRejects'
cd taskCloudService && go test ./src/ -count=1 -run 'PrivateRegistry|ResolveTargetArchitecturesRejects|ResolveContainerImageMetadataRejects'
```

全绿。
