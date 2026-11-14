import type { AxiosResponse } from 'axios'
import service from '@/utils/request/axios'
import { useAppStore, useAuthStore } from '@/store'

export interface DataExportSummary {
  username: string
  appVersion: string
  groupCount: number
  iconCount: number
  fileCount: number
  fileSize: number
  hasStyle: boolean
}

export interface DataExportOptions {
  includeIcons: boolean
  includeStyle: boolean
  includeFiles: boolean
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

// 获取可导出的数据概览
export function getSummary() {
  return service.post('/panel/dataExport/getSummary', {}, {
    headers: buildHeaders(),
  }) as unknown as Promise<AxiosResponse<ApiResponse<DataExportSummary>>>
}

// 导出完整数据，返回 ZIP 二进制
export function exportData(options: DataExportOptions) {
  return service.post('/panel/dataExport/export', options, {
    headers: buildHeaders(),
    responseType: 'blob',
  }) as unknown as Promise<AxiosResponse<Blob>>
}
