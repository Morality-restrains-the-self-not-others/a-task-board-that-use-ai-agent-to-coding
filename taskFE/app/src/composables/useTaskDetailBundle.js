import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const COMPOSABLES = dirname(fileURLToPath(import.meta.url))

/** Manifest order: facade last so `export function useTaskDetail` stays the entry. */
export const USE_TASK_DETAIL_SPLIT_FILES = [
  'taskDetail/useTaskDetailCore.js',
  'taskDetail/useTaskDetailFetches.js',
  'taskDetail/useTaskDetailContainer.js',
  'taskDetail/useTaskDetailActions.js',
  'taskDetail/useTaskDetailApi.js',
  'useTaskDetail.js',
]

export function readUseTaskDetailBundle() {
  return USE_TASK_DETAIL_SPLIT_FILES
    .map((rel) => join(COMPOSABLES, rel))
    .filter((abs) => existsSync(abs))
    .map((abs) => readFileSync(abs, 'utf8'))
    .join('\n')
}

export function useTaskDetailFilesForLineLimit() {
  return USE_TASK_DETAIL_SPLIT_FILES
    .map((rel) => join(COMPOSABLES, rel))
    .filter((abs) => existsSync(abs))
}
