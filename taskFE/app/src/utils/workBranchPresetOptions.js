/** 工作分支默认模版下拉项（${任务创建日期} 在生成/提交时替换为 YYYY-MM-DD） */
export const WORK_BRANCH_PRESET_OPTIONS = [
  { value: 'feature', label: 'feature/${任务创建日期}_${company_user_name}_daydaymoney${taskId}_${taskTitle}' },
  { value: 'bugfix', label: 'bugfix/${任务创建日期}_${company_user_name}_daydaymoney${taskId}_${taskTitle}' },
  { value: 'hotfix', label: 'hotfix/${任务创建日期}_${company_user_name}_daydaymoney${taskId}_${taskTitle}' },
  { value: 'release', label: 'release/${任务创建日期}_daydaymoney${taskId}' },
  { value: 'custom', label: '自定义' },
]
