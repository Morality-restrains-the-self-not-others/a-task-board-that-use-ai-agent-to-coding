# 测试意图：容器 TASK_API 经网关换票

| ID | 场景 | 期望 |
|----|------|------|
| T1 | routes.yaml 含 taskAgentSupport upstream 与 container-inbound-token | upstream=taskAgentSupport, auth_mode=none, priority > task-cloud-service |
| T2 | routes.yaml 含 server-userdata-verify | upstream=django, auth_mode=none |
| T3 | routes-to-apisix codegen | apisix.yaml 含 up-taskAgentSupport、container-inbound-token，无 forward-auth |
| T4 | get_userdata_verify_base_url 优先 vue.apiBaseUrl | 返回网关 origin，非 Django :8001 |
| T5 | get_userdata_verify_base_url 优先 taskGateway.publicBase | 同上 |
| T6 | get_userdata_verify_base_url 忽略回环网关 | 回退 django.allowedHost |
| T7 | userdataVerifyBaseURL 优先 PublicTaskAPIBase | Go 单测 |
| T8 | 运行时探针：POST gateway …/exchange-refresh/ | 非 Django 404 JSON |
| T9 | Django 兼容代理：POST :8001 …/exchange-refresh/ | 转发 TAS，非路径 404 |
| T10 | Django 代理 TAS 不可达 | 502 taskAgentSupport unreachable |

对应实现测例：

- `task2app/Saas_project/tests/test_taskgateway_routes_codegen.py`
- `task2app/Saas_project/tests/test_port_config_merge.py`
- `task2app/Saas_project/tests/test_container_inbound_tas_proxy.py`
- `taskCloudService/src/userdata_build_test.go`
