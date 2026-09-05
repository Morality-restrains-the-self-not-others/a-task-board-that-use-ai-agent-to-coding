import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js'
import DOMPurify from 'dompurify'

function escapeHtml(s) {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    const code = String(str || '')
    if (lang && hljs.getLanguage(lang)) {
      try {
        return hljs.highlight(code, { language: lang, ignoreIllegals: true }).value
      } catch {
        /* fall through */
      }
    }
    try {
      return hljs.highlightAuto(code).value
    } catch {
      return escapeHtml(code)
    }
  },
})

const MD_ALLOWED_TAGS = [
  'p',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'ul',
  'ol',
  'li',
  'blockquote',
  'code',
  'pre',
  'a',
  'strong',
  'em',
  'b',
  'i',
  'br',
  'hr',
  'table',
  'thead',
  'tbody',
  'tr',
  'th',
  'td',
  'span',
]

const MD_ALLOWED_ATTR = ['href', 'title', 'class', 'colspan', 'rowspan']

/**
 * Markdown → HTML → DOMPurify，用于助手正文展示。
 * @param {string} source
 * @returns {string}
 */
export function renderSafeMarkdown(source) {
  const raw = typeof source === 'string' ? source : ''
  if (!raw.trim()) return ''
  const rendered = md.render(raw)
  return DOMPurify.sanitize(rendered, {
    ALLOWED_TAGS: MD_ALLOWED_TAGS,
    ALLOWED_ATTR: MD_ALLOWED_ATTR,
    ALLOW_DATA_ATTR: false,
    ALLOWED_URI_REGEXP: /^(?:(?:https?|mailto):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i,
  })
}
