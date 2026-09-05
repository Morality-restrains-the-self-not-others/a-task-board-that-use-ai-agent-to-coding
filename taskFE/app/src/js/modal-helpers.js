/**
 * 遗留 DOM 看板模态：网络与租户路径小工具（运行依赖 window.apiFetch）。
 */

export function getTenantIdFromPath(pathname = window.location.pathname) {
    const m = pathname.match(/\/tenant\/([^/]+)\//);
    return m ? m[1] : '';
}

export const fetchWithTimeout = (url, options, timeout) => {
    return Promise.race([
        window.apiFetch(url, options),
        new Promise((_, reject) =>
            setTimeout(() => reject(new Error('请求超时')), timeout)
        ),
    ]);
};
