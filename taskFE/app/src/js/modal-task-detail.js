/**
 * 遗留 DOM 任务详情模态：详情镜像、启动云主机、评论挂载。
 */
import { fetchWithTimeout, getTenantIdFromPath } from './modal-helpers.js';
import { showRequestError } from '../utils/requestErrorDisplay.js';
import {
    applyLegacyStartupStatusDom,
    getWorkspaceIdFromLegacyContext,
    startLegacyStartupStatusPoll,
    stopLegacyStartupStatusPoll,
} from './modal-task-detail-startup.js';

function loadDetailImages() {
    console.log('🔍 开始执行 loadDetailImages 函数');

    const imageSelect = document.getElementById('detail-image');

    if (!imageSelect) {
        console.warn('⚠️ 未找到详情镜像选择元素，跳过加载镜像列表');
        const taskDetailModal = document.getElementById('task-detail-modal');
        if (taskDetailModal) {
            console.log('📋 任务详情模态框元素存在');
            const taskDetailContent = document.getElementById('task-detail-content');
            if (
                taskDetailContent &&
                taskDetailContent.querySelector(
                    'TaskDetailView, [data-vue-component="TaskDetailView"]'
                )
            ) {
                console.log(
                    'ℹ️ 检测到使用Vue组件显示任务详情，镜像列表可能由Vue组件管理'
                );
            }
        } else {
            console.warn('⚠️ 任务详情模态框也不存在！');
        }
        return;
    }

    console.log('✅ 找到详情镜像选择元素:', imageSelect);

    console.log('🔄 清空现有选项，添加加载中提示');
    imageSelect.innerHTML = '';
    const loadingOption = document.createElement('option');
    loadingOption.value = '';
    loadingOption.textContent = '-- 加载中... --';
    imageSelect.appendChild(loadingOption);

    const requestOptions = {
        method: 'GET',
        headers: {},
        credentials: 'same-origin',
    };

    console.log('📤 准备发送GET请求获取镜像列表');

    const tenantId = getTenantIdFromPath();
    console.log('从URL获取到的tenant_id:', tenantId);

    fetchWithTimeout(`/api/cloud/installed-images/tenant_id/${tenantId}`, requestOptions, 10000)
        .then((response) => {
            console.log('📥 镜像列表API响应状态:', response.status);
            if (!response.ok) {
                const err = new Error(`HTTP错误！状态：${response.status}`);
                err.traceId = response.traceId || '';
                throw err;
            }
            console.log('📥 响应头:', Object.fromEntries(response.headers.entries()));
            return response.json();
        })
        .then((data) => {
            console.log('📊 镜像列表API返回数据类型:', typeof data);
            console.log('📊 镜像列表API返回数据:', JSON.stringify(data, null, 2));

            imageSelect.innerHTML = '';

            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = '-- 无（默认） --';
            imageSelect.appendChild(defaultOption);

            let images = [];
            if (Array.isArray(data)) {
                console.log('📋 直接获取到镜像数组，数量:', data.length);
                images = data;
            } else if (data && typeof data === 'object') {
                if ('results' in data) {
                    console.log('📋 获取到分页结果，results数量:', data.results.length);
                    images = data.results;
                } else {
                    console.log('📋 获取到对象数据，但没有results字段，尝试直接使用');
                    images = [data];
                }
            } else {
                console.warn('⚠️ 获取到未知数据类型，无法处理');
                images = [];
            }

            if (images.length > 0) {
                console.log('✅ 找到', images.length, '个镜像，开始添加选项');
                images.forEach((image, index) => {
                    const ver = image.version || image.tag || '';
                    const archLabel = Array.isArray(image.target_architectures) && image.target_architectures.length
                        ? ` · 架构 ${image.target_architectures.join(', ')}`
                        : '';
                    const label = ver ? `${image.name}:${ver}${archLabel}` : `${image.name}${archLabel}`;
                    console.log(`   ${index + 1}. ${label} (ID: ${image.id})`);
                    const option = document.createElement('option');
                    option.value = image.id;
                    option.textContent = label;
                    imageSelect.appendChild(option);
                });
                console.log('✅ 镜像选项添加完成');
            } else {
                console.log('ℹ️ 没有找到可用镜像');
                const noImagesOption = document.createElement('option');
                noImagesOption.value = '';
                noImagesOption.textContent = '-- 没有可用镜像 --';
                noImagesOption.disabled = true;
                imageSelect.appendChild(noImagesOption);
            }
        })
        .catch((error) => {
            console.error('❌ 加载镜像失败:', error);
            console.error('❌ 错误详情:', error.message);
            console.error('❌ 错误堆栈:', error.stack);
            imageSelect.innerHTML = '';

            const errorOption = document.createElement('option');
            errorOption.value = '';
            errorOption.textContent = '-- 加载失败，点击重试 --';
            errorOption.disabled = false;
            imageSelect.appendChild(errorOption);

            imageSelect.addEventListener('click', function retryHandler() {
                imageSelect.removeEventListener('click', retryHandler);
                console.log('🔄 点击重试，重新加载镜像数据');
                loadDetailImages();
            });
        });
}

function startServer(taskId) {
    console.log('🔍 开始执行 startServer 函数，任务ID:', taskId);

    const imageSelect = document.getElementById('detail-image');
    const startBtn = document.getElementById('start-server-btn');

    if (!imageSelect) {
        console.warn('⚠️ 未找到详情镜像选择元素，跳过启动服务器操作');
        return;
    }

    if (!startBtn) {
        console.warn('⚠️ 未找到启动服务器按钮，跳过启动服务器操作');
        return;
    }

    const selectedImageId = imageSelect.value;

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin',
        body: JSON.stringify({
            task_id: taskId,
            container_image_id: selectedImageId || null,
        }),
    };

    stopLegacyStartupStatusPoll();
    startBtn.textContent = '启动中...';
    startBtn.disabled = true;
    applyLegacyStartupStatusDom({
        status: 'processing',
        message: '正在提交启动请求...',
        progress: 5,
    });

    window
        .apiFetch('/api/cloud/start-server/', requestOptions)
        .then((response) => {
            if (!response.ok) {
                const err = new Error(`HTTP错误！状态：${response.status}`);
                err.traceId = response.traceId || '';
                throw err;
            }
            return response.json();
        })
        .then((data) => {
            console.log('业务日志 - 启动服务器成功，返回数据:', data);
            const message =
                (typeof data?.message === 'string' && data.message) ||
                '服务器启动请求已提交，请稍后查看状态';
            startBtn.textContent = '启动中...';
            startBtn.disabled = true;
            applyLegacyStartupStatusDom({
                status: 'processing',
                message,
                progress: 5,
            });

            const tenantId = getTenantIdFromPath();
            const workspaceId = getWorkspaceIdFromLegacyContext();
            if (tenantId && workspaceId && taskId) {
                startLegacyStartupStatusPoll({
                    tenantId,
                    workspaceId,
                    taskId,
                    eventId: data?.event_id,
                    onUpdate: (polled) => {
                        applyLegacyStartupStatusDom({
                            status: polled.status || 'processing',
                            message: polled.message || message,
                            progress: polled.progress ?? 5,
                        });
                    },
                    onDone: (polled) => {
                        if (polled.status === 'success') {
                            startBtn.textContent = '服务器运行中';
                            startBtn.disabled = true;
                        } else if (polled.status === 'error') {
                            startBtn.textContent = '启动服务器';
                            startBtn.disabled = false;
                        }
                    },
                });
            }
        })
        .catch((error) => {
            console.error('业务日志 - 启动服务器失败:', error);
            stopLegacyStartupStatusPoll();
            applyLegacyStartupStatusDom({
                status: 'error',
                message: '启动服务器失败，请重试',
                progress: 0,
            });
            showRequestError('启动服务器失败，请重试', error);
            startBtn.textContent = '启动服务器';
            startBtn.disabled = false;
        });
}

function buildLegacyTaskDetailFullUrl(taskId) {
    const tenantId = getTenantIdFromPath();
    let workspaceId = '';
    if (window.currentUser && window.currentUser.current_workspace) {
        workspaceId = window.currentUser.current_workspace.id;
    }
    const basePath = `/tenant/${tenantId}/workspace/${workspaceId}/task-detail/${taskId}/`;
    const q = new URLSearchParams();
    try {
        const cur = new URL(window.location.href);
        for (const k of ['accessCode', 'github']) {
            const v = cur.searchParams.get(k);
            if (v != null && v !== '') q.set(k, v);
        }
    } catch (_) {
        /* ignore */
    }
    const qs = q.toString();
    const pathWithQuery = qs ? `${basePath}?${qs}` : basePath;
    return `${window.location.origin}${pathWithQuery}`;
}

export function openTaskDetailModal(taskId) {
    console.log('🔍 开始执行 openTaskDetailModal 函数，任务ID:', taskId);
    const taskDetailModal = document.getElementById('task-detail-modal');

    if (!taskDetailModal) {
        console.error('❌ 任务详情模态框未找到');
        return;
    }

    console.log('✅ 找到任务详情模态框:', taskDetailModal);

    const taskDetailContentEl = document.getElementById('task-detail-content');
    const vueManagedDetailLink =
        !!taskDetailContentEl?.querySelector('[data-vue-component="TaskDetailContent"]');

    console.log('🔗 第一步：遗留 DOM 看板下设置新标签页链接（Vue 工作面板由 :href 绑定，勿覆盖）');
    const openInNewTabLink = document.getElementById('open-task-in-new-tab');
    if (openInNewTabLink && !vueManagedDetailLink) {
        console.log('✅ 找到 open-task-in-new-tab 元素:', openInNewTabLink);
        console.log('✅ 当前链接 href 为:', openInNewTabLink.href);

        if (taskId) {
            const fullUrl = buildLegacyTaskDetailFullUrl(taskId);

            console.log('✅ 生成的任务详情URL:', fullUrl);

            openInNewTabLink.onclick = null;

            openInNewTabLink.href = fullUrl;
            console.log('✅ 直接设置后链接 href 为:', openInNewTabLink.href);

            openInNewTabLink.setAttribute('href', fullUrl);
            console.log('✅ setAttribute设置后链接 href 为:', openInNewTabLink.href);
        } else {
            console.error('❌ taskId 无效，无法设置链接 href');
        }
    } else if (!openInNewTabLink) {
        console.error('❌ 未找到 open-task-in-new-tab 元素');
        console.log('📋 任务详情模态框完整HTML:', taskDetailModal.innerHTML);
    } else if (vueManagedDetailLink) {
        console.log('ℹ️ TaskDetailContent 管理「新标签打开」链接，跳过遗留脚本改写 href');
    }

    const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
    if (!taskCard) {
        console.error('❌ 未找到任务卡片:', taskId);
        return;
    }

    console.log('✅ 找到任务卡片:', taskCard);

    const title = taskCard.querySelector('h4').textContent;
    const description = taskCard.querySelector('.text-sm.text-text-light').textContent;
    const priorityElement = taskCard.querySelector('span.text-xs.px-2.py-1.rounded');
    const priority = priorityElement.textContent;
    const createdBy = taskCard.querySelector('img').alt;
    const createdAt = taskCard.querySelector('.text-xs.text-gray-500').textContent;

    console.log('📋 提取的任务信息:', {
        title,
        description,
        priority,
        createdBy,
        createdAt,
    });

    console.log('📋 开始填充任务详情内容');

    const detailElements = {
        'detail-title': title,
        'detail-task-id': taskId,
        'detail-description': description,
        'detail-priority': priority,
        'detail-created-by': createdBy,
        'detail-created-at': createdAt,
    };

    for (const [id, content] of Object.entries(detailElements)) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = content;
            console.log(`✅ 填充元素 ${id}: ${content}`);
        } else {
            console.warn(`⚠️ 元素 ${id} 不存在，跳过填充`);
        }
    }

    console.log('📋 任务详情内容填充完成');

    const taskDetailContent = document.getElementById('task-detail-content');
    if (
        taskDetailContent &&
        taskDetailContent.querySelector('TaskDetailView, [data-vue-component="TaskDetailView"]')
    ) {
        console.log('ℹ️ 检测到使用Vue组件显示任务详情，跳过传统DOM操作');
    }

    console.log('🔄 调用 loadDetailImages() 加载镜像列表');
    loadDetailImages();

    const startBtn = document.getElementById('start-server-btn');
    if (startBtn) {
        console.log('✅ 找到启动服务器按钮:', startBtn);
        startBtn.onclick = null;
        startBtn.onclick = () => startServer(taskId);
        console.log('🔄 为启动服务器按钮添加事件监听');
    } else {
        console.error('❌ 未找到启动服务器按钮');
        console.log('📋 任务详情模态框完整HTML:', taskDetailModal.innerHTML);
    }

    console.log('🔄 显示任务详情模态框');
    taskDetailModal.classList.remove('hidden');
    document.body.classList.add('modal-open');

    const tenantIdForComments = getTenantIdFromPath();
    if (window.commentsModule) {
        window.commentsModule.renderComments(taskId, tenantIdForComments);
        window.commentsModule.addDetailCommentEventListeners(taskId, tenantIdForComments);
        console.log('🔄 初始化评论列表');
    }

    console.log('✅ 任务详情模态框打开完成');
}

export function closeTaskDetailModal() {
    console.log('关闭任务详情模态框...');
    stopLegacyStartupStatusPoll();
    const taskDetailModal = document.getElementById('task-detail-modal');
    if (taskDetailModal) {
        taskDetailModal.classList.add('hidden');
        document.body.classList.remove('modal-open');
    }
}

export function toggleTaskDetailMaximize() {
    console.log('切换任务详情模态框最大化状态...');
    const taskDetailModal = document.getElementById('task-detail-modal');
    if (!taskDetailModal) {
        console.error('未找到任务详情模态框！');
        return;
    }

    const modalContent = taskDetailModal.querySelector('div');
    if (!modalContent) {
        console.error('未找到模态框内容容器！');
        return;
    }

    const maximizeBtn = document.getElementById('maximize-task-detail-modal');
    if (!maximizeBtn) {
        console.error('未找到最大化按钮！');
        return;
    }
    const maxIcon = maximizeBtn.querySelector('.max-icon');
    const restoreIcon = maximizeBtn.querySelector('.restore-icon');

    if (modalContent.classList.contains('maximized-modal')) {
        modalContent.classList.remove('maximized-modal');
        modalContent.style.width = '';
        modalContent.style.maxWidth = '';
        modalContent.style.height = '';
        modalContent.style.maxHeight = '';
        modalContent.style.borderRadius = '';
        maxIcon.style.display = 'block';
        restoreIcon.style.display = 'none';
        maximizeBtn.title = '最大化';
        console.log('恢复模态框正常大小');
    } else {
        modalContent.classList.add('maximized-modal');
        modalContent.style.width = '100%';
        modalContent.style.maxWidth = '100%';
        modalContent.style.height = '100%';
        modalContent.style.maxHeight = '100%';
        modalContent.style.borderRadius = '0';
        maxIcon.style.display = 'none';
        restoreIcon.style.display = 'block';
        maximizeBtn.title = '恢复';
        console.log('最大化模态框');
    }
}
