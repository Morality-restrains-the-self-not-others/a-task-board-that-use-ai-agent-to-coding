# 测试意图：所有进程加载 conf 必须叠加 conf-local

## 覆盖的功能意图

`docs/intents/platform/conf_local_overlay_all_processes.intent.md`

## 测例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `ReadAppConfig` + conf-local docker-infra | redis.host 来自 overlay |
| T2 | `UnmarshalYAMLMerged` PayPal | client_secret 来自 conf-local |
| T3 | `MergeYAMLAtPath` provider YAML | client_secret overlay |
| T4 | `loadDomainEventsGlobal` docker-infra conf-local | redis/kafka 来自 overlay |
| T5 | `loadPaypalConfig` | 空骨架被 overlay |
| T6 | `overlayConfLocalRel` nested app | internalSecret overlay |
| T7 | `overlay_conf_file` Python | nested rel overlay |
| T8 | taskSSE 源码 | 无 `config.local.yaml`，有 `conf-local` 与 `taskSSE` |
| T9 | `ResolveBaseYaml` / `BaseYAMLLoader` | `baseDomain` 来自 `conf-local/base.yaml` |
| T10 | CI `check_conf_local_overlay.py` | 直读 tracked YAML / `config.local.yaml` 判违规；live repo 全绿 |

## 对应测试文件

- `shareLib/confload/overlay_test.go`
- `taskEvents/config/conf_local_overlay_test.go`
- `taskBill/src/paypal_pay_test.go`
- `runAll/src/config_conf_local_test.go`
- `runAll/scripts/test_conf_local.py`
- `runAll/scripts/test_conf_loader.py`
- `taskSSE/src/config.quoteTrim.test.mjs`
- `shareLib/confload/overlay_test.go` (`TestResolveBaseYamlMergesConfLocal`)
- `runAll/scripts/tests/test_base_yaml_loader.py` (`test_overlays_conf_local_base_yaml`)
- `db/scripts/ci/test_check_conf_local_overlay.py`
