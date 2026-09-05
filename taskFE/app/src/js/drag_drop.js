// 拖拽功能模块

// 更新单个任务顺序的函数
function updateTaskOrder(taskId, updateData) {
    return window.apiFetch(`/api/projects/todos/${taskId}/`, {
        method: 'PATCH',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(updateData)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP错误！状态：${response.status}`);
        }
        return response.json();
    })
    .then(data => {
        console.log(`任务 ${taskId} 更新成功`, data);
        return data;
    });
}

// 初始化拖拽功能
function initDragDrop() {
    // 为所有任务列初始化拖拽功能
    document.addEventListener('DOMContentLoaded', function() {
        console.log('DOM加载完成，初始化工作面板功能');
        
        // 检查Sortable是否已加载
        if (typeof Sortable === 'undefined') {
            console.error('Sortable.js 未加载成功！');
            return;
        } else {
            console.log('Sortable.js 已加载成功');
        }
        
        // 获取所有任务状态列
        const taskColumns = document.querySelectorAll('.task-column');
        console.log('找到的任务状态列数量:', taskColumns.length);
        
        if (taskColumns.length === 0) {
            console.error('未找到任务状态列！');
            return;
        }
        
        // 为每个任务状态列初始化拖拽功能
        taskColumns.forEach((column, columnIndex) => {
            console.log(`处理第 ${columnIndex + 1} 个任务状态列`);
            
            // 获取任务卡片容器
            const container = column.querySelector('.task-cards-container');
            if (!container) {
                console.error(`第 ${columnIndex + 1} 个任务状态列中未找到任务卡片容器！`);
                return;
            }
            
            console.log(`第 ${columnIndex + 1} 个任务状态列中的任务卡片容器:`, container);
            
            // 获取任务卡片
            const taskCards = container.querySelectorAll('.task-card');
            console.log(`第 ${columnIndex + 1} 个任务状态列中的任务卡片数量:`, taskCards.length);
            
            try {
                new Sortable(container, {
                    group: 'shared', // 相同的组名允许跨列拖动
                    animation: 150, // 拖动动画持续时间
                    draggable: '.task-card', // 指定可拖拽的元素
                    ghostClass: 'sortable-ghost', // 拖动时的幽灵元素类
                    chosenClass: 'sortable-chosen', // 选中元素的类
                    dragClass: 'sortable-drag', // 拖动中元素的类
                    // 拖动开始时的回调
                    onStart: function(evt) {
                        console.log('拖动开始:', {
                            taskId: evt.item.dataset.taskId,
                            startIndex: evt.oldIndex
                        });
                    },
                    // 当拖动元素进入另一个容器时触发
                    onOver: function(evt) {
                        // 添加拖拽中的样式到目标泳道
                        const targetColumn = evt.to.closest('.task-column');
                        if (targetColumn) {
                            targetColumn.classList.add('drag-over');
                        }
                    },
                    // 当拖动元素离开当前容器时触发
                    onOut: function(evt) {
                        // 移除拖拽中的样式
                        const currentColumn = evt.from.closest('.task-column');
                        if (currentColumn) {
                            currentColumn.classList.remove('drag-over');
                        }
                    },
                    // 拖动结束时的回调
                    onEnd: function(evt) {
                        // 移除所有泳道的拖拽样式
                        document.querySelectorAll('.task-column').forEach(column => {
                            column.classList.remove('drag-over');
                        });
                        
                        const taskId = evt.item.dataset.taskId;
                        const toColumn = evt.to.closest('.task-column');
                        
                        // 获取目标状态ID
                        // 从HTML中获取状态信息
                        let statusList = [];
                        const statusElements = document.querySelectorAll('.task-column');
                        statusElements.forEach((element, index) => {
                            const statusName = element.querySelector('h3').textContent.trim().split(' ')[0];
                            statusList.push({ id: index + 1, name: statusName });
                        });
                        
                        const toColumnElement = toColumn;
                        const toColumnTitle = toColumnElement.querySelector('h3').textContent.trim();
                        let targetStatusId = null;
                        
                        console.log('可用的状态列表:', statusList);
                        console.log('目标列标题:', toColumnTitle);
                        
                        // 从statusList中查找对应的状态ID
                        for (let i = 0; i < statusList.length; i++) {
                            const status = statusList[i];
                            console.log(`检查状态: ${status.name}, 标题是否包含: ${toColumnTitle.includes(status.name)}`);
                            if (toColumnTitle.includes(status.name)) {
                                targetStatusId = status.id;
                                console.log('找到匹配的状态ID:', targetStatusId);
                                break;
                            }
                        }
                        
                        if (!targetStatusId) {
                            console.error('未找到目标状态ID');
                            return;
                        }
                        
                        console.log('任务已移动:', {
                            taskId: taskId,
                            fromColumn: evt.from.closest('.task-column').querySelector('h3').textContent.trim(),
                            toColumn: toColumnTitle,
                            targetStatusId: targetStatusId,
                            newIndex: evt.newIndex,
                            oldIndex: evt.oldIndex
                        });
                        
                        // 获取当前列和目标列的所有任务卡片
                        const fromColumnContainer = evt.from;
                        const toColumnContainer = evt.to;
                        
                        // 获取当前列的所有任务ID和它们的新顺序
                        const fromColumnTaskIds = Array.from(fromColumnContainer.querySelectorAll('.task-card'))
                            .map(card => card.dataset.taskId);
                        
                        // 获取目标列的所有任务ID和它们的新顺序
                        const toColumnTaskIds = Array.from(toColumnContainer.querySelectorAll('.task-card'))
                            .map(card => card.dataset.taskId);
                        
                        console.log('当前列任务ID:', fromColumnTaskIds);
                        console.log('目标列任务ID:', toColumnTaskIds);
                        
                        // 准备批量更新数据
                        const updatePromises = [];
                        
                        // 更新当前列的所有任务顺序
                        fromColumnTaskIds.forEach((currentTaskId, index) => {
                            const updateData = {
                                order: index
                            };
                            updatePromises.push(updateTaskOrder(currentTaskId, updateData));
                        });
                        
                        // 更新目标列的所有任务顺序
                        toColumnTaskIds.forEach((currentTaskId, index) => {
                            const updateData = {
                                order: index
                            };
                            
                            // 如果是正在移动的任务，还需要更新状态
                            if (currentTaskId === taskId) {
                                updateData.status = targetStatusId;
                            }
                            
                            updatePromises.push(updateTaskOrder(currentTaskId, updateData));
                        });
                        
                        // 发送所有更新请求
                        Promise.all(updatePromises)
                            .then(results => {
                                console.log('所有任务更新成功:', results.length);
                            })
                            .catch(error => {
                                console.error('任务更新失败:', error);
                            });
                    }
                });
                console.log(`第 ${columnIndex + 1} 个任务状态列的拖拽功能初始化成功`);
            } catch (error) {
                console.error(`初始化第 ${columnIndex + 1} 个任务状态列的拖拽功能时发生错误:`, error);
            }
        });
    });
}

// 导出函数，供其他模块使用
window.dragDropModule = {
    initDragDrop
};

// 初始化拖拽功能
initDragDrop();