/**
 * 遗留看板：DOM 事件绑定与创建任务表单提交（由 WorkPanel 延迟调用 initModals）。
 */
import { getTenantIdFromPath } from './modal-helpers.js';
import { showRequestError } from '../utils/requestErrorDisplay.js';
import { openModal, closeModal } from './modal-create-task.js';
import {
    openTaskDetailModal,
    closeTaskDetailModal,
    toggleTaskDetailMaximize,
} from './modal-task-detail.js';

let _documentModalDelegateAttached = false;
let _windowModalBackdropAttached = false;
let _modalEnterKeyAttached = false;

function handleModalEnterKey(e) {
    // 文本框内回车应换行，不触发弹窗确认
    const tag = (e.target?.tagName || '').toLowerCase();
    if (tag === 'textarea') return;

    // 创建任务模态框：触发表单提交
    const createTaskModal = document.getElementById('create-task-modal');
    if (createTaskModal && !createTaskModal.classList.contains('hidden')) {
        const form = document.getElementById('create-task-form');
        if (form) {
            form.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
            return;
        }
    }

    // 任务详情模态框：无需 Enter 确认（只读）
}

function handleDocumentModalClick(e) {
    if (e.target.closest('#close-modal')) {
        console.log('事件委托 - 点击关闭按钮或其子元素');
        closeModal();
    } else if (e.target.matches('#cancel-btn')) {
        console.log('事件委托 - 点击取消按钮');
        closeModal();
    } else if (e.target.closest('#close-task-detail-modal')) {
        console.log('事件委托 - 点击任务详情关闭按钮或其子元素');
        closeTaskDetailModal();
    } else if (e.target.closest('#maximize-task-detail-modal')) {
        console.log('事件委托 - 点击最大化按钮或其子元素');
        toggleTaskDetailMaximize();
    }
}

function handleWindowModalBackdrop(event) {
    const modal = document.getElementById('create-task-modal');
    const taskDetailModal = document.getElementById('task-detail-modal');
    if (event.target === modal) {
        console.log('业务日志 - 点击模态框外部，关闭创建任务模态框');
        closeModal();
    } else if (event.target === taskDetailModal) {
        console.log('业务日志 - 点击模态框外部，关闭任务详情模态框');
        closeTaskDetailModal();
    }
}

export function initModals() {
    console.log('DOM加载完成，初始化模态框功能...');

    const createTaskBtn = document.querySelector('.btn-primary');
    const closeModalBtn = document.getElementById('close-modal');
    const cancelBtn = document.getElementById('cancel-btn');
    const form = document.getElementById('create-task-form');

    const closeTaskDetailModalBtn = document.getElementById('close-task-detail-modal');
    const closeTaskDetailBtn = document.getElementById('close-task-detail-btn');

    if (closeTaskDetailModalBtn) {
        closeTaskDetailModalBtn.onclick = null;

        closeTaskDetailModalBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            console.log('直接事件监听 - 点击任务详情关闭按钮');
            closeTaskDetailModal();
        });
    }

    if (!_documentModalDelegateAttached) {
        _documentModalDelegateAttached = true;
        document.addEventListener('click', handleDocumentModalClick);
    }

    if (!_windowModalBackdropAttached) {
        _windowModalBackdropAttached = true;
        window.addEventListener('click', handleWindowModalBackdrop);
    }

    if (!_modalEnterKeyAttached) {
        _modalEnterKeyAttached = true;
        document.addEventListener('keydown', function (e) {
            if (e.key === 'Enter') handleModalEnterKey(e);
        });
    }

    console.log('添加任务卡片点击事件监听...');
    const taskCards = document.querySelectorAll('.task-card');
    console.log('找到的任务卡片数量:', taskCards.length);

    if (taskCards.length > 0) {
        taskCards.forEach((card) => {
            let isDragging = false;

            card.addEventListener('dragstart', function () {
                isDragging = true;
                console.log('拖拽开始事件触发');
            });

            card.addEventListener('dragend', function () {
                isDragging = false;
                console.log('拖拽结束事件触发');
            });

            card.addEventListener('click', function () {
                console.log('任务卡片点击事件触发，isDragging:', isDragging);
                if (!isDragging) {
                    const taskId = this.dataset.taskId;
                    console.log('业务日志 - 点击任务卡片，任务ID:', taskId);
                    if (
                        this.hasAttribute('@click') ||
                        this.hasAttribute('data-vue-click') ||
                        this.closest('[data-vue-component]')
                    ) {
                        console.log('ℹ️ 检测到任务卡片可能由Vue组件管理，跳过遗留 modal 处理');
                    } else {
                        try {
                            openTaskDetailModal(taskId);
                        } catch (error) {
                            console.error('调用openTaskDetailModal失败:', error);
                        }
                    }
                }
            });
        });
    } else {
        console.log('未找到任务卡片！');
    }

    if (form) {
        form.addEventListener('submit', function (event) {
            event.preventDefault();

            const formData = new FormData(form);
            const taskData = Object.fromEntries(formData);

            taskData.workspace = window.currentWorkspaceId || null;

            if (taskData.project === '') {
                taskData.project = null;
            }

            if (taskData.container_image === '') {
                taskData.container_image = null;
            }

            console.log('业务日志 - 提交创建任务表单，任务数据:', taskData);

            const tenantId = getTenantIdFromPath();
            console.log('从URL获取到的tenant_id:', tenantId);
            window
                .apiFetch(`/api/tasks/todos/tenant_id/${tenantId}/`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(taskData),
                })
                .then((response) => {
                    console.log('业务日志 - 创建任务请求返回，状态码:', response.status);
                    return response.json();
                })
                .then(() => {
                    console.log('业务日志 - 创建任务成功，刷新页面');
                    window.location.reload();
                })
                .catch((error) => {
                    console.error('业务日志 - 创建任务失败:', error);
                    showRequestError('创建任务失败，请重试', error);
                });
        });
    }

    const addTaskBtns = document.querySelectorAll('.add-task-btn');
    console.log('找到的添加任务按钮数量:', addTaskBtns.length);

    if (addTaskBtns.length > 0) {
        addTaskBtns.forEach((btn, index) => {
            btn.addEventListener('click', function () {
                const statusName = this.closest('.task-column').querySelector('h3').textContent.trim();
                console.log(
                    `业务日志 - 点击第${index + 1}个"添加任务"按钮（状态：${statusName}），打开创建任务模态框`
                );

                const statusId = statusName.split(' ')[0];
                const statusSelect = document.getElementById('task-status');
                if (statusSelect) {
                    for (let i = 0; i < statusSelect.options.length; i++) {
                        if (statusSelect.options[i].textContent === statusId) {
                            statusSelect.value = statusSelect.options[i].value;
                            break;
                        }
                    }
                }
                openModal();
            });
        });
    } else {
        console.error('错误：未找到添加任务按钮！');
    }

    if (createTaskBtn) {
        createTaskBtn.addEventListener('click', function () {
            console.log('业务日志 - 点击顶部"创建任务"按钮，打开创建任务模态框');
            openModal();
        });
    } else {
        console.warn('⚠️ 未找到创建任务按钮，可能由Vue组件管理');
    }

    if (closeModalBtn) {
        closeModalBtn.addEventListener('click', function () {
            console.log('业务日志 - 点击模态框"关闭"按钮，关闭创建任务模态框');
            closeModal();
        });
    } else {
        console.warn('⚠️ 未找到关闭按钮，可能由Vue组件管理');
    }

    if (cancelBtn) {
        cancelBtn.addEventListener('click', function () {
            console.log('业务日志 - 点击"取消"按钮，关闭创建任务模态框');
            closeModal();
        });
    } else {
        console.warn('⚠️ 未找到取消按钮，可能由Vue组件管理');
    }

    if (closeTaskDetailBtn) {
        closeTaskDetailBtn.addEventListener('click', function () {
            console.log('业务日志 - 点击"关闭"按钮，关闭任务详情模态框');
            closeTaskDetailModal();
        });
    } else {
        console.warn('⚠️ 未找到任务详情模态框取消按钮，可能由Vue组件管理');
    }
}
