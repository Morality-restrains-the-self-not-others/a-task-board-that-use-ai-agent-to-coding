/**
 * 项目详情 GET 失败时的用户可见文案。
 * 404 / project not found 表示目录行已不在，不是「页面空白」。
 */
export function projectDetailFetchErrorMessage(status, detail) {
  const d = detail != null ? String(detail).trim() : ''
  if (Number(status) === 404 || d === 'project not found') {
    return '项目不存在或已删除'
  }
  return d || '获取项目详情失败'
}
