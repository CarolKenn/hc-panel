import type { AxiosResponse } from 'axios'
import service from '@/utils/request/axios'
import { useAppStore, useAuthStore } from '@/store'

export interface ImportWarning {
  code: string
  params?: Record<string, string>
}

export interface DataImportPreview {
  importId: string
  fileType: 'zip' | 'json' | string
  appVersion: string
  exportTime: string
  sourceUsername: string
  groupCount: number
  iconCount: number
  fileCount: number
  fileSize: number
  hasIcons: boolean
  hasStyle: boolean
  hasFiles: boolean
  canRestoreFiles: boolean
  warnings: ImportWarning[]
}

export interface DataImportOptions {
  importId: string
  mode: 'merge' | 'replace'
  includeIcons: boolean
  includeStyle: boolean
  includeFiles: boolean
}

export interface DataImportResult {
  mode: string
  groupCount: number
  iconCount: number
  fileRestored: number
  fileSkipped: number
  fileFailed: number
  styleRestored: boolean
  warnings: ImportWarning[]
}

interface ApiResponse<T> {
  code: number
  msg: string
  data: T
}

// 后端登录拦截器读取的是 token 头，此处手动补全
function buildHeaders() {
  const authStore = useAuthStore()
  const appStore = useAppStore()
  return {
    token: authStore.token,
    lang: appStore.language,
  }
}

// 上传备份文件并解析内容（不写入任何数据）
export function previewBackup(file: File) {
  const form = new FormData()
  form.append('file', file)
  return service.post('/panel/dataImport/preview', form, {
    headers: buildHeaders(),
  }) as unknown as Promise<AxiosResponse<ApiResponse<DataImportPreview>>>
}

// 按备份内容恢复数据
export function applyImport(options: DataImportOptions) {
  return service.post('/panel/dataImport/apply', options, {
    headers: buildHeaders(),
  }) as unknown as Promise<AxiosResponse<ApiResponse<DataImportResult>>>
}
