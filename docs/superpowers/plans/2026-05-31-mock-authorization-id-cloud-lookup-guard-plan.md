# Implementation Plan: Mock authorization_id 云平台查库防护

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:test-driven-development when executing.

**Goal:** 非数值 authorization_id 不再触发 CloudPlatformAuthorization ORM ValueError；真实 PK 行为不变。

**Architecture:** VO `CloudPlatformAuthorizationReference` 解析 PK；`authorization_lookup` 封装 Django 查库；四处 service 替换 inline filter。

**Tech Stack:** Django 4.2, pytest, existing CloudServerConfigHistory CharField

---

### Task 1: VO + lookup helper + unit tests

**Files:**
- Create: `cloud/domain/value_objects/cloud_platform_authorization_reference.py`
- Create: `cloud/services/authorization_lookup.py`
- Create: `tests/test_authorization_lookup.py`
- Create: `tests/cloud/domain/test_cloud_platform_authorization_reference.py`

**Steps:** Red → Green → pytest both files

### Task 2: Refactor get_previous_server_config

**Files:**
- Modify: `cloud/services/get_previous_server_config.py`
- Existing: `tests/test_get_previous_server_config.py`

### Task 3: stop_vm guard

**Files:**
- Modify: `cloud/services/stop_vm.py`
- Create: `tests/test_stop_vm_mock_authorization_id.py`

### Task 4: get_server_runtime_status + start_vm reuse path

**Files:**
- Modify: `cloud/services/get_server_runtime_status.py`
- Modify: `cloud/services/start_vm.py`
- Extend tests as needed

### Task 5: value-stream.yaml step + full pytest

**Files:**
- Modify: `value-stream.yaml` (append step under cloud-integration)

**Verify:** `pytest tests/test_authorization_lookup.py tests/test_get_previous_server_config.py tests/test_stop_vm_mock_authorization_id.py -v`
