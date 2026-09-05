import { apiFetch } from '../utils/apiUtils.js';
import { initialsAvatarDataUri } from '../utils/initialsAvatarDataUri.js';
import { showRequestError } from '../utils/requestErrorDisplay.js';
import { getTenantIdFromPath } from './modal-helpers.js';

function resolveTenantId(tenantId) {
    let tid = tenantId != null ? String(tenantId).trim() : '';
    if (!tid) {
        tid = getTenantIdFromPath() || '';
    }
    return tid;
}

// 渲染任务评论
function renderComments(taskId, tenantId) {
    console.log('渲染任务评论，任务ID:', taskId);
    const tid = resolveTenantId(tenantId);
    if (!tid) {
        console.error('加载评论失败: 缺少租户 ID');
        const commentsContainer = document.getElementById('comments-container');
        if (commentsContainer) {
            commentsContainer.innerHTML = '';
            const errorMsg = document.createElement('p');
            errorMsg.className = 'text-xs text-red-500 text-center py-2';
            errorMsg.textContent = '无法加载评论（缺少租户上下文）';
            commentsContainer.appendChild(errorMsg);
        }
        return;
    }
    // 清空评论容器
    const commentsContainer = document.getElementById('comments-container');
    commentsContainer.innerHTML = '';
    
    // 创建加载提示
    const loadingMsg = document.createElement('p');
    loadingMsg.className = 'text-xs text-gray-500 text-center py-2';
    loadingMsg.textContent = '加载评论中...';
    commentsContainer.appendChild(loadingMsg);
    
    // 调用API获取评论列表
    fetch(`/api/tasks/${encodeURIComponent(String(taskId))}/comments/tenant_id/${encodeURIComponent(tid)}`, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin'
    })
    .then(response => {
        if (!response.ok) {
            const err = new Error(`HTTP错误！状态：${response.status}`);
            err.traceId = response.traceId || '';
            throw err;
        }
        return response.json();
    })
    .then(data => {
        console.log('获取到的评论数据:', data);
        // 清空评论容器
        commentsContainer.innerHTML = '';
        
        // 如果没有评论，显示暂无评论提示
        if (data.length === 0) {
            const noCommentMsg = document.createElement('p');
            noCommentMsg.className = 'text-xs text-gray-500 text-center py-2';
            noCommentMsg.textContent = '暂无评论';
            commentsContainer.appendChild(noCommentMsg);
        } else {
            // 渲染评论列表
            data.forEach(comment => {
                // 创建评论元素
                const commentItem = document.createElement('div');
                commentItem.className = 'comment-item mb-3 pb-3 border-b border-gray-100 last:border-b-0 last:mb-0 last:pb-0';
                
                // 创建评论内容结构
                const commentContent = document.createElement('div');
                commentContent.className = 'flex items-start';
                
                // 创建头像
                const avatar = document.createElement('img');
                avatar.className = 'w-5 h-5 rounded-full border-2 border-white mr-2';
                // 本地首字母占位头像
                const username = comment.created_by.username || '用户';
                avatar.src = initialsAvatarDataUri(username);
                avatar.alt = username;
                commentContent.appendChild(avatar);
                
                // 创建评论信息容器
                const commentInfo = document.createElement('div');
                commentInfo.className = 'flex-1';
                
                // 创建评论头部
                const commentHeader = document.createElement('div');
                commentHeader.className = 'flex justify-between items-center mb-1';
                
                // 创建用户名
                const usernameSpan = document.createElement('span');
                usernameSpan.className = 'text-xs font-medium text-gray-800';
                usernameSpan.textContent = username;
                commentHeader.appendChild(usernameSpan);
                
                // 创建评论时间
                const commentTime = document.createElement('span');
                commentTime.className = 'text-xs text-gray-500';
                commentTime.textContent = new Date(comment.created_at).toLocaleString('zh-CN', {
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit'
                });
                commentHeader.appendChild(commentTime);
                
                // 添加评论头部到评论信息
                commentInfo.appendChild(commentHeader);
                
                // 创建评论内容
                const commentText = document.createElement('p');
                commentText.className = 'text-xs text-gray-700';
                commentText.textContent = comment.content;
                commentInfo.appendChild(commentText);
                
                // 添加评论信息到评论内容
                commentContent.appendChild(commentInfo);
                
                // 添加评论内容到评论元素
                commentItem.appendChild(commentContent);
                
                // 添加评论到评论容器
                commentsContainer.appendChild(commentItem);
            });
        }
    })
    .catch(error => {
        console.error('加载评论失败:', error);
        // 清空评论容器
        commentsContainer.innerHTML = '';
        // 显示加载失败提示
        const errorMsg = document.createElement('p');
        errorMsg.className = 'text-xs text-red-500 text-center py-2';
        errorMsg.textContent = '加载评论失败，请重试';
        commentsContainer.appendChild(errorMsg);
    });
}

// 为任务详情模态框中的评论功能添加事件监听器
function addDetailCommentEventListeners(taskId, tenantId) {
    console.log('添加评论事件监听器...');
    const tid = tenantId != null ? String(tenantId).trim() : '';
    
    // 直接获取元素，不需要等待DOM更新
    const commentList = document.getElementById('detail-comment-list');
    const addCommentForm = document.getElementById('detail-add-comment-form');
    
    console.log('获取到的元素:', {
        commentList: commentList ? '找到' : '未找到',
        addCommentForm: addCommentForm ? '找到' : '未找到'
    });
    
    if (commentList && addCommentForm) {
        
        // 提交评论按钮事件
        const submitBtn = document.getElementById('detail-comment-submit-btn');
        const cancelBtn = document.getElementById('detail-comment-cancel-btn');
        const textarea = document.getElementById('detail-comment-content');
        
        if (submitBtn && cancelBtn && textarea) {
            // 移除旧的事件监听器
            submitBtn.replaceWith(submitBtn.cloneNode(true));
            const newSubmitBtn = document.getElementById('detail-comment-submit-btn');
            
            cancelBtn.replaceWith(cancelBtn.cloneNode(true));
            const newCancelBtn = document.getElementById('detail-comment-cancel-btn');
            
            newSubmitBtn.addEventListener('click', function() {
                console.log('点击了提交评论按钮');
                const content = textarea.value.trim();
                if (content) {
                    submitDetailComment(taskId, content, tid);
                }
            });
            
            // 取消评论按钮事件
            newCancelBtn.addEventListener('click', function() {
                console.log('点击了取消评论按钮');
                textarea.value = '';
            });
        }
    }
}

// 提交任务详情模态框中的评论
function submitDetailComment(taskId, content, tenantId) {
    const tid = resolveTenantId(tenantId);
    if (!tid) {
        console.error('提交评论失败: 缺少租户 ID');
        showRequestError('提交评论失败：缺少租户上下文');
        return;
    }
    // 准备请求数据
    const commentData = {
        content: content
    };
    
    // 发送POST请求创建评论（任务 ID 在 URL 中）
    apiFetch(`/api/tasks/${encodeURIComponent(String(taskId))}/comments/tenant_id/${encodeURIComponent(tid)}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'same-origin',
        body: JSON.stringify(commentData)
    })
    .then(response => {
        if (!response.ok) {
            const err = new Error(`HTTP错误！状态：${response.status}`);
            err.traceId = response.traceId || '';
            throw err;
        }
        return response.json();
    })
    .then(comment => {
        // 更新任务详情模态框中的评论UI
        updateDetailCommentUI(comment);
        // 同时更新任务卡片上的评论UI
        const taskCard = document.querySelector(`[data-task-id="${taskId}"]`);
        if (taskCard) {
            updateCommentUI(comment, taskCard);
        }
    })
    .catch(error => {
        console.error('提交评论失败:', error);
        showRequestError('提交评论失败，请重试', error);
    });
}

// 更新任务详情模态框中的评论UI
function updateDetailCommentUI(comment) {
    const commentsContainer = document.getElementById('comments-container');
    const textarea = document.getElementById('detail-comment-content');
    
    // 清空文本框
    textarea.value = '';
    
    // 移除暂无评论提示（如果存在）
    const noCommentMsg = commentsContainer.querySelector('.text-center');
    if (noCommentMsg) {
        noCommentMsg.remove();
    }
    
    // 创建新评论元素
    const newComment = document.createElement('div');
    newComment.className = 'comment-item mb-3 pb-3 border-b border-gray-100 last:border-b-0 last:mb-0 last:pb-0';
    
    // 创建评论内容结构
    const commentContent = document.createElement('div');
    commentContent.className = 'flex items-start';
    
    // 创建头像
    const avatar = document.createElement('img');
    avatar.className = 'w-5 h-5 rounded-full border-2 border-white mr-2';
    // 本地首字母占位头像
    const username = comment.created_by.username;
    avatar.src = initialsAvatarDataUri(username);
    avatar.alt = username;
    commentContent.appendChild(avatar);
    
    // 创建评论信息容器
    const commentInfo = document.createElement('div');
    commentInfo.className = 'flex-1';
    
    // 创建评论头部
    const commentHeader = document.createElement('div');
    commentHeader.className = 'flex justify-between items-center mb-1';
    
    // 创建用户名
    const usernameSpan = document.createElement('span');
    usernameSpan.className = 'text-xs font-medium text-gray-800';
    usernameSpan.textContent = comment.created_by.username;
    commentHeader.appendChild(usernameSpan);
    
    // 创建评论时间
    const commentTime = document.createElement('span');
    commentTime.className = 'text-xs text-gray-500';
    commentTime.textContent = new Date(comment.created_at).toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
    commentHeader.appendChild(commentTime);
    
    // 创建评论内容
    const commentText = document.createElement('p');
    commentText.className = 'text-xs text-gray-700';
    commentText.textContent = comment.content;
    
    // 组装评论信息
    commentInfo.appendChild(commentHeader);
    commentInfo.appendChild(commentText);
    
    // 组装评论
    commentContent.appendChild(commentInfo);
    newComment.appendChild(commentContent);
    
    // 将新评论添加到评论容器末尾
    commentsContainer.appendChild(newComment);
}

// 更新任务卡片上的评论UI
function updateCommentUI(comment, cardElement) {
    const commentList = cardElement.querySelector('.comment-list');
    const addCommentForm = cardElement.querySelector('.add-comment-form');
    const textarea = cardElement.querySelector('textarea');
    const commentCount = cardElement.querySelector('.comment-toggle-btn span');
    
    // 清空文本框
    textarea.value = '';
    // 隐藏添加评论表单
    if (addCommentForm) {
        addCommentForm.classList.add('hidden');
    }
    
    // 移除暂无评论提示（如果存在）
    if (commentList) {
        const noCommentMsg = commentList.querySelector('.text-center');
        if (noCommentMsg) {
            noCommentMsg.remove();
        }
        
        // 创建新评论元素
        const newComment = document.createElement('div');
        newComment.className = 'comment-item mb-3 pb-3 border-b border-gray-100 last:border-b-0 last:mb-0 last:pb-0';
        
        // 创建评论内容结构
        const commentContent = document.createElement('div');
        commentContent.className = 'flex items-start';
        
        // 创建头像
        const avatar = document.createElement('img');
        avatar.className = 'w-5 h-5 rounded-full border-2 border-white mr-2';
        // 本地首字母占位头像
        const username = comment.created_by.username;
        avatar.src = initialsAvatarDataUri(username);
        avatar.alt = username;
        commentContent.appendChild(avatar);
        
        // 创建评论信息容器
        const commentInfo = document.createElement('div');
        commentInfo.className = 'flex-1';
        
        // 创建评论头部
        const commentHeader = document.createElement('div');
        commentHeader.className = 'flex justify-between items-center mb-1';
        
        // 创建用户名
        const usernameSpan = document.createElement('span');
        usernameSpan.className = 'text-xs font-medium text-gray-800';
        usernameSpan.textContent = comment.created_by.username;
        commentHeader.appendChild(usernameSpan);
        
        // 创建评论时间
        const commentTime = document.createElement('span');
        commentTime.className = 'text-xs text-gray-500';
        commentTime.textContent = new Date(comment.created_at).toLocaleString('zh-CN', {
            month: '2-digit',
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit'
        });
        commentHeader.appendChild(commentTime);
        
        // 创建评论内容
        const commentText = document.createElement('p');
        commentText.className = 'text-xs text-gray-700';
        commentText.textContent = comment.content;
        
        // 组装评论信息
        commentInfo.appendChild(commentHeader);
        commentInfo.appendChild(commentText);
        
        // 组装评论
        commentContent.appendChild(commentInfo);
        newComment.appendChild(commentContent);
        
        // 添加到评论列表
        commentList.appendChild(newComment);
        
        // 更新评论数量
        if (commentCount) {
            const currentCount = parseInt(commentCount.textContent) || 0;
            commentCount.textContent = currentCount + 1;
        }
        
        // 更新评论按钮文本
        const toggleBtn = cardElement.querySelector('.comment-toggle-btn');
        if (toggleBtn) {
            const commentCount = cardElement.querySelectorAll('.comment-item').length;
            if (commentCount > 0) {
                // 移除现有内容
                toggleBtn.innerHTML = '';
                
                // 创建SVG图标
                const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
                svg.className = 'w-3 h-3 mr-1';
                svg.setAttribute('fill', 'none');
                svg.setAttribute('stroke', 'currentColor');
                svg.setAttribute('viewBox', '0 0 24 24');
                
                // 创建SVG路径
                const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
                path.setAttribute('stroke-linecap', 'round');
                path.setAttribute('stroke-linejoin', 'round');
                path.setAttribute('stroke-width', '2');
                path.setAttribute('d', 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z');
                svg.appendChild(path);
                
                // 添加SVG到按钮
                toggleBtn.appendChild(svg);
                
                // 创建文本节点
                const text = document.createTextNode(` 评论 (${commentCount})`);
                toggleBtn.appendChild(text);
            }
        }
    }
}

// 导出函数，供其他模块使用
window.commentsModule = {
    renderComments,
    addDetailCommentEventListeners,
    submitDetailComment
};