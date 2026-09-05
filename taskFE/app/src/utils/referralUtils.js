// 推荐码处理工具函数

// 从localStorage获取推荐码
export const getReferralCode = () => {
  return localStorage.getItem('referralCode') || '';
};

// 存储推荐码到localStorage
export const setReferralCode = (code) => {
  if (code) {
    localStorage.setItem('referralCode', code);
  } else {
    localStorage.removeItem('referralCode');
  }
};

// 从URL中提取推荐码
export const extractReferralCodeFromUrl = () => {
  const urlParams = new URLSearchParams(window.location.search);
  return urlParams.get('accessCode') || '';
};

// 处理URL中的推荐码，存储到localStorage
export const handleUrlReferralCode = () => {
  const code = extractReferralCodeFromUrl();
  if (code) {
    setReferralCode(code);
  }
};

// 为URL添加推荐码
export const addReferralCodeToUrl = (url) => {
  const code = getReferralCode();
  if (!code) return url;
  
  const urlObj = new URL(url, window.location.origin);
  urlObj.searchParams.set('accessCode', code);
  return urlObj.toString();
};

const isSameOriginHttpLink = (anchor) => {
  if (!anchor?.href || anchor.href.startsWith('javascript:')) return false;
  if (anchor.hasAttribute('download')) return false;
  try {
    return new URL(anchor.href).origin === window.location.origin;
  } catch {
    return false;
  }
};

const navigateWithReferral = (href, newTab) => {
  const url = addReferralCodeToUrl(href);
  if (newTab) {
    window.open(url, '_blank', 'noopener,noreferrer');
  } else {
    window.location.href = url;
  }
};

// 初始化推荐码处理
export const initReferralCode = () => {
  // 处理URL中的推荐码
  handleUrlReferralCode();

  // 同源链接：在 URL 上附带推荐码。须保留「新标签页打开」类默认行为（Cmd/Ctrl/Shift+点击、target="_blank"、中键）。
  document.addEventListener('click', (event) => {
    if (event.defaultPrevented || event.button !== 0) return;
    const anchor = event.target.closest('a');
    if (!isSameOriginHttpLink(anchor)) return;

    const modifierOpensNewTab =
      event.metaKey || event.ctrlKey || event.shiftKey;
    const blankTarget =
      anchor.target && anchor.target.toLowerCase() === '_blank';

    if (modifierOpensNewTab || blankTarget) {
      event.preventDefault();
      navigateWithReferral(anchor.href, true);
      return;
    }

    event.preventDefault();
    navigateWithReferral(anchor.href, false);
  });

  document.addEventListener('auxclick', (event) => {
    if (event.button !== 1) return;
    const anchor = event.target.closest('a');
    if (!isSameOriginHttpLink(anchor)) return;
    event.preventDefault();
    navigateWithReferral(anchor.href, true);
  });
};
