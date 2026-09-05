import DOMPurify from 'dompurify'

/** 步骤卡片内 AI HTML 允许的标签（禁止 script/iframe/form/input 等） */
const STEP_HTML_ALLOWED_TAGS = [
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'p',
  'div',
  'span',
  'strong',
  'b',
  'em',
  'i',
  'ul',
  'ol',
  'li',
  'blockquote',
  'table',
  'thead',
  'tbody',
  'tr',
  'th',
  'td',
  'br',
  'hr',
  'code',
  'pre',
]

const STEP_HTML_ALLOWED_ATTR = ['class', 'colspan', 'rowspan', 'data-value', 'data-action', 'data-param', 'data-group']

const STEP_HTML_FORBID_TAGS = ['script', 'iframe', 'form', 'input', 'textarea', 'select', 'option', 'button', 'object', 'embed', 'link', 'meta', 'base', 'style']

/**
 * 净化容器/模型下发的步骤 HTML，仅保留交互约定 class 与 data-*。
 * @param {string} dirty
 * @returns {string}
 */
export function purifyStepInteractiveHtml(dirty) {
  const s = typeof dirty === 'string' ? dirty : ''
  if (!s.trim()) return ''
  return DOMPurify.sanitize(s, {
    ALLOWED_TAGS: STEP_HTML_ALLOWED_TAGS,
    ALLOWED_ATTR: STEP_HTML_ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
    FORBID_TAGS: STEP_HTML_FORBID_TAGS,
    FORBID_ATTR: ['style', 'id'],
  })
}

const IFRAME_BASE_CSS = `
html, body { height: 100%; margin: 0; overflow: auto; }
:root { color: #1f2937; font-family: ui-sans-serif, system-ui, sans-serif; font-size: 12px; line-height: 1.45; }
body.root { margin: 0; padding: 8px; box-sizing: border-box; }
.opt-radio, .opt-checkbox, .action-btn {
  cursor: pointer;
  display: inline-block;
  margin: 2px 4px 2px 0;
  padding: 2px 8px;
  border-radius: 6px;
  border: 1px solid #c7d2fe;
  background: #eef2ff;
  user-select: none;
}
.action-btn { background: #f0fdf4; border-color: #86efac; }
.opt-radio.is-selected, .opt-checkbox.is-selected {
  background: #4f46e5;
  color: #fff;
  border-color: #4338ca;
}
table { border-collapse: collapse; width: 100%; margin: 6px 0; }
th, td { border: 1px solid #e5e7eb; padding: 4px 6px; text-align: left; }
blockquote { margin: 6px 0; padding-left: 8px; border-left: 3px solid #a5b4fc; color: #4b5563; }
pre, code { font-family: ui-monospace, monospace; }
pre { background: #f9fafb; padding: 6px; border-radius: 4px; overflow: auto; }
`

/** 受信任的内联脚本（非 AI 输出）：将点击转为 postMessage，供父窗口校验 event.source 后处理 */
const IFRAME_TRUSTED_CLICK_BRIDGE = `(function(){
function lbl(el){return (el.textContent||'').trim()}
function clearSibling(except){
  var g=except.getAttribute('data-group');
  if(g){
    document.querySelectorAll('.opt-radio[data-group="'+String(g).replace(/"/g,'')+'"]').forEach(function(n){if(n!==except)n.classList.remove('is-selected')});
  }else{
    var p=except.parentElement;if(!p)return;
    p.querySelectorAll(':scope > .opt-radio').forEach(function(n){if(n!==except)n.classList.remove('is-selected')});
  }
}
document.addEventListener('click',function(e){
  var t=e.target;if(!t||!t.closest)return;
  var el=t.closest('.opt-radio, .opt-checkbox, .action-btn');if(!el)return;
  e.preventDefault();e.stopPropagation();
  if(el.classList.contains('opt-radio')){
    clearSibling(el);el.classList.add('is-selected');
    parent.postMessage({__secureStepRich:true,kind:'radio',value:el.getAttribute('data-value')||'',group:el.getAttribute('data-group')||'',label:lbl(el)},'*');
  }else if(el.classList.contains('opt-checkbox')){
    el.classList.toggle('is-selected');
    parent.postMessage({__secureStepRich:true,kind:'checkbox',value:el.getAttribute('data-value')||'',selected:el.classList.contains('is-selected'),label:lbl(el)},'*');
  }else if(el.classList.contains('action-btn')){
    parent.postMessage({__secureStepRich:true,kind:'action',action:el.getAttribute('data-action')||'',param:el.getAttribute('data-param')||'',label:lbl(el)},'*');
  }
},true);
})();`

/**
 * 组装 iframe srcdoc：内联样式 + 受信任点击桥接脚本 + 净化后的 body。
 * iframe 需使用 sandbox 含 allow-scripts allow-same-origin。
 * @param {string} purifiedBodyInnerHtml
 * @returns {string}
 */
export function buildStepIframeSrcDoc(purifiedBodyInnerHtml) {
  const body = purifiedBodyInnerHtml || ''
  return `<!DOCTYPE html><html><head><meta charset="utf-8"><style>${IFRAME_BASE_CSS}</style><script>${IFRAME_TRUSTED_CLICK_BRIDGE}<\/script></head><body class="root">${body}</body></html>`
}
