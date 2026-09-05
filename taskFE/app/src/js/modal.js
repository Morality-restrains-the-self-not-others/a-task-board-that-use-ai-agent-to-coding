// 遗留看板：window.modalModule。依赖 modal-init（事件）、modal-create-task、modal-task-detail、modal-helpers
import { openModal, closeModal } from './modal-create-task.js';
import {
    openTaskDetailModal,
    closeTaskDetailModal,
    toggleTaskDetailMaximize,
} from './modal-task-detail.js';
import { initModals } from './modal-init.js';

window.modalModule = {
    openModal,
    closeModal,
    openTaskDetailModal,
    closeTaskDetailModal,
    toggleTaskDetailMaximize,
    initModals,
};
