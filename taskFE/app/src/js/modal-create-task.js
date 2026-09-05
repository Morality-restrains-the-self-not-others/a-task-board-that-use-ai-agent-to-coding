/**
 * 遗留「创建任务」模态：项目/镜像下拉与开关。
 */
import { fetchWithTimeout, getTenantIdFromPath } from './modal-helpers.js';

export function loadImages(selectedProjectId, defaultImageId) {
    const imageSelect = document.getElementById('task-image');

    if (!imageSelect) {
        // WorkPanel 等页面由 Vue CreateTaskModal 管理下拉，无遗留 id="task-image"
        return;
    }

    imageSelect.innerHTML = '';
    const loadingOption = document.createElement('option');
    loadingOption.value = '';
    loadingOption.textContent = '-- 加载中... --';
    imageSelect.appendChild(loadingOption);

    const requestOptions = {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin',
    };

    const tenantId = getTenantIdFromPath();
    console.log('从URL获取到的tenant_id:', tenantId);

    fetchWithTimeout(`/api/cloud/installed-images/tenant_id/${tenantId}`, requestOptions, 10000)
        .then((response) => {
            console.log('创建任务镜像列表API响应状态:', response.status);
            if (!response.ok) {
                throw new Error(`HTTP错误！状态：${response.status}`);
            }
            return response.json();
        })
        .then((data) => {
            console.log('创建任务镜像列表API返回数据:', data);
            imageSelect.innerHTML = '';

            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = '-- 无（默认） --';
            imageSelect.appendChild(defaultOption);

            let images = [];
            if (Array.isArray(data)) {
                images = data;
            } else if (data && typeof data === 'object' && 'results' in data) {
                images = data.results;
            }

            if (images.length > 0) {
                console.log('找到', images.length, '个镜像');
                images.forEach((image) => {
                    const option = document.createElement('option');
                    option.value = image.id;
                    const ver = image.version || image.tag || '';
                    option.textContent = ver ? `${image.name}:${ver}` : image.name;
                    if (defaultImageId && image.id == defaultImageId) {
                        option.selected = true;
                    }
                    imageSelect.appendChild(option);
                });
            } else {
                console.log('没有找到可用镜像');
                const noImagesOption = document.createElement('option');
                noImagesOption.value = '';
                noImagesOption.textContent = '-- 没有可用镜像 --';
                noImagesOption.disabled = true;
                imageSelect.appendChild(noImagesOption);
            }
        })
        .catch((error) => {
            console.error('加载镜像失败:', error);
            imageSelect.innerHTML = '';

            const errorOption = document.createElement('option');
            errorOption.value = '';
            errorOption.textContent = '-- 加载失败 --';
            imageSelect.appendChild(errorOption);

            const retryOption = document.createElement('option');
            retryOption.value = '';
            retryOption.textContent = '点击重试';
            retryOption.disabled = true;
            imageSelect.appendChild(retryOption);

            imageSelect.addEventListener('click', function retryHandler() {
                imageSelect.removeEventListener('click', retryHandler);
                loadImages(selectedProjectId, defaultImageId);
            });
        });
}

function handleProjectChange(event) {
    const selectedProjectId = event.target.value;
    const selectedOption = event.target.options[event.target.selectedIndex];
    const defaultImageId = selectedOption.getAttribute('data-container-image');

    loadImages(selectedProjectId, defaultImageId);
}

export function loadProjects() {
    const projectSelect = document.getElementById('task-project');

    if (!projectSelect) {
        return;
    }

    projectSelect.innerHTML = '';
    const loadingOption = document.createElement('option');
    loadingOption.value = '';
    loadingOption.textContent = '-- 加载中... --';
    projectSelect.appendChild(loadingOption);

    const requestOptions = {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin',
    };

    const currentWorkspaceId = window.currentWorkspaceId || null;

    const tenantId = getTenantIdFromPath();
    console.log('从URL获取到的tenant_id:', tenantId);
    let apiUrl = `/api/projects/tenant_id/${tenantId}`;
    if (currentWorkspaceId) {
        apiUrl += `?workspace_id=${currentWorkspaceId}`;
    }

    fetchWithTimeout(apiUrl, requestOptions, 10000)
        .then((response) => {
            if (!response.ok) {
                throw new Error(`HTTP错误！状态：${response.status}`);
            }
            return response.json();
        })
        .then((data) => {
            projectSelect.innerHTML = '';

            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = '-- 无（默认） --';
            projectSelect.appendChild(defaultOption);

            if (Array.isArray(data) && data.length > 0) {
                data.forEach((project) => {
                    const option = document.createElement('option');
                    option.value = project.id;
                    option.textContent = project.name;
                    option.setAttribute(
                        'data-container-image',
                        project.container_image ? project.container_image : ''
                    );
                    projectSelect.appendChild(option);
                });
            } else {
                const noProjectsOption = document.createElement('option');
                noProjectsOption.value = '';
                noProjectsOption.textContent = '-- 当前工作空间下没有项目 --';
                noProjectsOption.disabled = true;
                projectSelect.appendChild(noProjectsOption);
            }

            projectSelect.addEventListener('change', handleProjectChange);
        })
        .catch((error) => {
            console.error('加载项目失败:', error);
            projectSelect.innerHTML = '';

            const errorOption = document.createElement('option');
            errorOption.value = '';
            errorOption.textContent = '-- 加载失败 --';
            projectSelect.appendChild(errorOption);

            const retryOption = document.createElement('option');
            retryOption.value = '';
            retryOption.textContent = '点击重试';
            retryOption.disabled = true;
            projectSelect.appendChild(retryOption);

            projectSelect.addEventListener('click', function retryHandler() {
                projectSelect.removeEventListener('click', retryHandler);
                loadProjects();
            });
        });
}

export function openModal() {
    console.log('打开模态框...');
    const modal = document.getElementById('create-task-modal');
    if (modal) {
        modal.classList.remove('hidden');
        document.body.classList.add('modal-open');
        if (document.getElementById('task-project')) {
            loadProjects();
        }
        if (document.getElementById('task-image')) {
            loadImages(null, null);
        }
    }
}

export function closeModal() {
    console.log('关闭模态框...');
    const modal = document.getElementById('create-task-modal');
    const form = document.getElementById('create-task-form');
    if (modal) {
        modal.classList.add('hidden');
        document.body.classList.remove('modal-open');
        if (form) {
            form.reset();
        }
    }
}
