# 重写 get_available_instances 方法 - 实现计划

## [x] 任务 1: 分析当前 get_available_instances 方法的实现
- **优先级**: P0
- **依赖**: 无
- **描述**:
  - 分析当前 `get_available_instances` 方法的实现逻辑
  - 了解 `_get_available_resources` 方法的实现
  - 了解云厂商SDK的调用方式和参数
- **成功标准**:
  - 完全理解当前代码的实现逻辑和流程
- **测试要求**:
  - `programmatic` TR-1.1: 理解当前代码的执行流程
  - `human-judgement` TR-1.2: 能够解释当前代码的实现细节

## [x] 任务 2: 重写 get_available_instances 方法 - 第一步：获取可用 InstanceType 列表
- **优先级**: P0
- **依赖**: 任务 1
- **描述**:
  - 修改 `get_available_instances` 方法，首先调用 DescribeAvailableResource 获取可用的 InstanceType 列表
  - 确保正确处理 API 响应和错误情况
- **成功标准**:
  - 能够正确获取并返回可用的 InstanceType 列表
- **测试要求**:
  - `programmatic` TR-2.1: 方法能够成功调用 DescribeAvailableResource API
  - `programmatic` TR-2.2: 能够正确解析 API 响应并提取 InstanceType 列表

## [x] 任务 3: 重写 get_available_instances 方法 - 第二步：获取 InstanceType 对应的可挂载磁盘类型
- **优先级**: P0
- **依赖**: 任务 2
- **描述**:
  - 对于第一步获取的每个 InstanceType，调用 DescribeAvailableResource 获取对应的可挂载磁盘类型
  - 确保正确处理 API 响应和错误情况
- **成功标准**:
  - 能够为每个 InstanceType 获取对应的可挂载磁盘类型
- **测试要求**:
  - `programmatic` TR-3.1: 方法能够为每个 InstanceType 成功调用 DescribeAvailableResource API
  - `programmatic` TR-3.2: 能够正确解析 API 响应并提取可挂载磁盘类型

## [x] 任务 4: 重写 get_available_instances 方法 - 第三步：获取价格信息
- **优先级**: P0
- **依赖**: 任务 3
- **描述**:
  - 对于每个 InstanceType 和对应的可挂载磁盘类型，调用 DescribePrice 获取价格信息
  - 确保正确处理 API 响应和错误情况
- **成功标准**:
  - 能够为每个 InstanceType 和磁盘类型组合获取价格信息
- **测试要求**:
  - `programmatic` TR-4.1: 方法能够为每个 InstanceType 和磁盘类型组合成功调用 DescribePrice API
  - `programmatic` TR-4.2: 能够正确解析 API 响应并提取价格信息

## [x] 任务 5: 整合所有信息并返回格式化结果
- **优先级**: P0
- **依赖**: 任务 4
- **描述**:
  - 将获取的 InstanceType、磁盘类型和价格信息整合到一起
  - 返回格式化的实例列表
- **成功标准**:
  - 返回的实例列表包含所有必要的信息，包括 InstanceType、磁盘类型和价格
- **测试要求**:
  - `programmatic` TR-5.1: 返回的实例列表格式正确
  - `programmatic` TR-5.2: 每个实例包含 InstanceType、磁盘类型和价格信息

## [x] 任务 6: 测试重写后的方法
- **优先级**: P1
- **依赖**: 任务 5
- **描述**:
  - 测试重写后的 `get_available_instances` 方法
  - 确保方法能够正确处理各种情况，包括错误处理
- **成功标准**:
  - 方法能够正确返回可用实例列表，包含所有必要的信息
- **测试要求**:
  - `programmatic` TR-6.1: 方法能够成功执行并返回结果
  - `programmatic` TR-6.2: 方法能够正确处理错误情况
