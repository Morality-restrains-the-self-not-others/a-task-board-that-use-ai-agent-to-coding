# 实现todo任务传递给docker容器并将结果添加到评论中

## 1. 需求分析
- 在todo应用中创建echo hello任务
- 将任务传递给docker容器
- docker容器显示当前时间
- 将docker容器的输出添加到任务评论中

## 2. 实现步骤

### 2.1 修改视图函数
- 在`views.py`中添加`run_docker_task`视图函数
- 使用Python的`subprocess`模块调用docker命令
- 将docker容器的输出添加到任务评论中

### 2.2 添加URL路由
- 在`urls.py`中添加`run_docker_task`的URL模式

### 2.3 更新模板
- 在`todo_list.html`中为每个任务添加"Run Docker"按钮
- 在`todo_detail.html`中添加"Run Docker"按钮

### 2.4 实现docker命令调用
- 使用`docker run`命令运行容器
- 使用`alpine`或`ubuntu`镜像
- 执行`echo hello && date`命令
- 捕获容器输出

### 2.5 添加评论
- 将docker容器的输出作为评论内容添加到任务中

## 3. 代码实现

### 3.1 修改`views.py`
- 添加`import subprocess`
- 添加`run_docker_task`函数
- 实现docker命令调用和评论添加逻辑

### 3.2 修改`urls.py`
- 添加`path('<int:pk>/run-docker/', views.run_docker_task, name='run_docker_task'),`

### 3.3 修改模板
- 在任务列表和详情页添加"Run Docker"按钮

## 4. 测试
- 创建echo hello任务
- 点击"Run Docker"按钮
- 查看评论中是否添加了docker容器的输出

## 5. 注意事项
- 确保docker服务正在运行
- 确保当前用户有执行docker命令的权限
- 处理docker命令执行失败的情况
- 处理命令超时的情况