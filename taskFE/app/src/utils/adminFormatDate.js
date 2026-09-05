/**
 * Format admin list timestamps for zh-CN display.
 * @param {string|Date|null|undefined} dateString
 * @returns {string}
 */
export function adminFormatDate(dateString) {
  if (!dateString) {
    return '-'
  }

  if (dateString instanceof Date) {
    return dateString.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const dateStr = String(dateString)
  let date = new Date(dateStr)

  if (isNaN(date.getTime())) {
    let processedDateStr = dateStr
    processedDateStr = processedDateStr.replace(/(\+\d{2}:\d{2}|Z)$/, '')
    processedDateStr = processedDateStr.replace('T', ' ')
    processedDateStr = processedDateStr.replace(/\.\d+/, '')
    date = new Date(processedDateStr)

    if (isNaN(date.getTime())) {
      const dateRegex = /(\d{4})[-/](\d{2})[-/](\d{2})\s*(\d{2}):(\d{2}):(\d{2})/
      const match = processedDateStr.match(dateRegex)
      if (match) {
        const [, year, month, day, hour, minute, second] = match
        date = new Date(year, month - 1, day, hour, minute, second)
      }
    }
  }

  if (isNaN(date.getTime())) {
    return dateStr
  }

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}
