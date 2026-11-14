<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import type { UploadFileInfo } from 'naive-ui'
import { NAlert, NButton, NCheckbox, NCheckboxGroup, NDivider, NInput, NRadio, NRadioGroup, NSpace, NUpload, useMessage } from 'naive-ui'
import { RoundCardModal, SvgIcon } from '@/components/common'
import type { IconGroup, ImportJsonResult } from '@/utils/jsonImportExport'
import { ConfigVersionLowError, FormatError, exportJson, importJsonString } from '@/utils/jsonImportExport'
import { get as getAbout } from '@/api/system/about'
import { edit as addGroup, getList as getGroupList } from '@/api/panel/itemIconGroup'
import { addMultiple as addMultipleIcons, getListByGroupId } from '@/api/panel/itemIcon'
import type { DataExportOptions, DataExportSummary } from '@/api/panel/dataExport'
import { exportData, getSummary as getExportSummary } from '@/api/panel/dataExport'
import type { DataImportPreview } from '@/api/panel/dataImport'
import { applyImport, previewBackup } from '@/api/panel/dataImport'

import { t } from '@/locales'

interface ItemGroup extends Panel.ItemIconGroup {
  items?: Panel.ItemInfo[]
}

const ms = useMessage()

const jsonData = ref<string | null>(null)
const importWarning = ref<string[]>([])
const importRoundModalShow = ref(false)
const exportRoundModalShow = ref(false)
const loading = ref(false)
const uploadLoading = ref(false)
const version = ref('') // 当前软件版本
const debug = ref(false)

// 完整数据备份
const backupLoading = ref(false)
const backupSummary = ref<DataExportSummary | null>(null)
const backupOptions = ref<DataExportOptions>({
  includeIcons: true,
  includeStyle: true,
  includeFiles: true,
})

const backupSummaryText = computed(() => {
  if (!backupSummary.value)
    return t('apps.exportImport.backupLoadingSummary')

  return t('apps.exportImport.backupSummary', {
    groups: backupSummary.value.groupCount,
    icons: backupSummary.value.iconCount,
    files: backupSummary.value.fileCount,
    size: formatFileSize(backupSummary.value.fileSize),
  })
})

const backupOptionsEmpty = computed(() => !backupOptions.value.includeIcons && !backupOptions.value.includeStyle && !backupOptions.value.includeFiles)

// 从备份恢复
const restoreShow = ref(false)
const restorePreviewing = ref(false)
const restoreLoading = ref(false)
const restorePreview = ref<DataImportPreview | null>(null)
const restoreMode = ref<'merge' | 'replace'>('merge')
const restoreOptions = ref({
  includeIcons: true,
  includeStyle: true,
  includeFiles: true,
})

const restoreOptionsEmpty = computed(() => !restoreOptions.value.includeIcons && !restoreOptions.value.includeStyle && !restoreOptions.value.includeFiles)

const restoreMetaText = computed(() => {
  const preview = restorePreview.value
  if (!preview)
    return ''

  return t('apps.exportImport.restoreMeta', {
    user: preview.sourceUsername || '-',
    time: preview.exportTime || '-',
    version: preview.appVersion || '-',
  })
})

const restoreStatsText = computed(() => {
  const preview = restorePreview.value
  if (!preview)
    return ''

  return t('apps.exportImport.backupSummary', {
    groups: preview.groupCount,
    icons: preview.iconCount,
    files: preview.fileCount,
    size: formatFileSize(preview.fileSize),
  })
})

const restoreWarnings = computed(() => {
  const warnings = restorePreview.value?.warnings ?? []
  return warnings.map(item => t(`apps.exportImport.restoreWarn.${item.code}`, item.params ?? {}))
})

const importObj = ref<ImportJsonResult | null> (null)

const importItems = ref<string[]>(['icons']) // 当前软件版本支持导入导出的项目
const checkedItems = ref<string[]>(['icons']) // 当前准备导入的项目

// 导入图标
async function importIcons(): Promise<string | null> {
  const groups = importObj.value?.geticons()
  const batchSize = 50

  if (!groups)
    return null

  try {
    for (let i = 0; i < groups.length; i++) {
      const element = groups[i]

      // 创建组得到组id
      const createGroupResponse = await addGroup<Panel.ItemIconGroup>({
        title: element.title,
        sort: element.sort,
      })

      if (createGroupResponse.code === 0) {
        const groupId = createGroupResponse.data?.id

        if (groupId) {
          let addIcons: Panel.ItemInfo[] = []

          // 批量添加子项
          for (let iconI = 0; iconI < element.children.length; iconI++) {
            const iconElement = element.children[iconI]

            addIcons.push({
              title: iconElement.title,
              sort: iconElement.sort,
              icon: iconElement.icon,
              url: iconElement.url,
              lanUrl: iconElement.lanUrl,
              description: iconElement.description,
              openMethod: iconElement.openMethod,
              itemIconGroupId: groupId,
            })

            // 每 batchSize 个添加一次
            if (addIcons.length === batchSize || iconI === element.children.length - 1) {
              const response = await addMultipleIcons(addIcons)

              if (response.code !== 0)
                return response.msg

              addIcons = []
            }
          }
        }
      }
      else {
        return createGroupResponse.msg
      }
    }

    return null
  }
  catch (error) {
    if (error instanceof Error)
      return `${t('common.failed')}: ${error.message}`
    else
      return t('common.unknownError')
  }
}

// 导出图标
async function exportIcons(): Promise<IconGroup[]> {
  const iconGroups: IconGroup[] = []

  // 获取组数据
  const { code, data } = await getGroupList<Common.ListResponse<ItemGroup[]>>()

  if (code === 0) {
    // 使用 Promise.all 等待所有异步操作完成
    await Promise.all(data.list.map(async (element) => {
      const group: IconGroup = {
        title: element.title as string,
        sort: element.sort as 0,
        children: [],
      }

      const res = await getListByGroupId<Common.ListResponse<Panel.ItemInfo[]>>(element.id)

      if (res.code === 0) {
        for (const iconElement of res.data.list) {
          group.children.push({
            icon: iconElement.icon,
            sort: iconElement.sort || 99999,
            title: iconElement.title,
            url: iconElement.url,
            lanUrl: iconElement.lanUrl || '',
            description: iconElement.description || '',
            openMethod: iconElement.openMethod || 1,
          })
        }
      }

      iconGroups.push(group)
    }))

    return iconGroups
  }
  else {
    return []
  }
}

// 文件大小格式化
function formatFileSize(size: number): string {
  if (!size || size <= 0)
    return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = size
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value = value / 1024
    unitIndex++
  }

  return `${unitIndex === 0 ? value : value.toFixed(1)} ${units[unitIndex]}`
}

// 获取可导出的数据概览
async function loadBackupSummary() {
  try {
    const res = await getExportSummary()
    if (res.data.code === 0)
      backupSummary.value = res.data.data
  }
  catch (error) {
    console.error('Failed to load export summary', error)
  }
}

// 从响应头解析下载文件名
function parseDownloadFileName(disposition?: string): string {
  if (!disposition)
    return ''

  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match)
    return decodeURIComponent(utf8Match[1])

  const match = disposition.match(/filename="?([^";]+)"?/i)
  return match ? match[1] : ''
}

// 触发浏览器下载
function downloadBlob(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

function defaultBackupFileName(): string {
  const now = new Date()
  const pad = (value: number) => `${value}`.padStart(2, '0')
  const stamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  return `SunPanel-Backup-${backupSummary.value?.username || 'user'}-${stamp}.zip`
}

// 导出完整数据备份
async function handleExportBackup() {
  if (backupOptionsEmpty.value) {
    ms.warning(t('apps.exportImport.backupSelectAtLeastOne'))
    return
  }

  backupLoading.value = true
  try {
    const response = await exportData({ ...backupOptions.value })
    const blob = response.data

    // 后端异常时返回的是 JSON
    if (blob.type.includes('application/json')) {
      let errMsg = t('common.failed')
      try {
        const result = JSON.parse(await blob.text())
        if (result?.msg)
          errMsg = result.msg
      }
      catch (error) {
        console.error('Failed to parse export error response', error)
      }
      ms.error(errMsg)
      return
    }

    const fileName = parseDownloadFileName(response.headers['content-disposition'] as string | undefined) || defaultBackupFileName()
    downloadBlob(blob, fileName)
    ms.success(t('common.success'))
  }
  catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    ms.error(`${t('common.failed')}: ${message}`)
  }
  finally {
    backupLoading.value = false
  }
}

// 恢复失败时的错误文案（后端返回固定错误码）
function importErrorText(code: number, fallback?: string): string {
  const messages: Record<number, string> = {
    1400: t('apps.exportImport.errorImportFileFormat'),
    1401: t('apps.exportImport.errorImportVersion'),
    1402: t('apps.exportImport.errorImportSession'),
    1403: t('apps.exportImport.errorImportEmpty'),
  }
  return messages[code] ?? (fallback || t('common.failed'))
}

// 选择备份文件后先解析预览，确认真实写入前不改动任何数据
async function handleRestoreFileChange(options: { file: UploadFileInfo; fileList: Array<UploadFileInfo> }) {
  const file = options.file.file
  if (!file)
    return

  restorePreviewing.value = true
  try {
    const res = await previewBackup(file)
    if (res.data.code !== 0) {
      ms.error(importErrorText(res.data.code, res.data.msg))
      return
    }

    const preview = res.data.data
    restorePreview.value = preview
    restoreOptions.value = {
      includeIcons: preview.hasIcons,
      includeStyle: preview.hasStyle,
      includeFiles: preview.canRestoreFiles,
    }
    restoreMode.value = 'merge'
    restoreShow.value = true
  }
  catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    ms.error(`${t('common.failed')}: ${message}`)
  }
  finally {
    restorePreviewing.value = false
  }
}

// 执行恢复
async function handleRestoreApply() {
  const preview = restorePreview.value
  if (!preview)
    return

  if (restoreOptionsEmpty.value) {
    ms.warning(t('apps.exportImport.restoreSelectAtLeastOne'))
    return
  }

  restoreLoading.value = true
  try {
    const res = await applyImport({
      importId: preview.importId,
      mode: restoreMode.value,
      ...restoreOptions.value,
    })

    if (res.data.code !== 0) {
      ms.error(importErrorText(res.data.code, res.data.msg))
      return
    }

    const result = res.data.data
    const fileCount = result.fileRestored + result.fileSkipped
    ms.success(t('apps.exportImport.restoreSuccess', {
      groups: result.groupCount,
      icons: result.iconCount,
      files: fileCount,
    }), { duration: 6000 })
    ms.info(t('apps.exportImport.restoreDoneRefresh'), { duration: 6000 })

    if (result.warnings?.length)
      result.warnings.forEach(item => ms.warning(t(`apps.exportImport.restoreWarn.${item.code}`, item.params ?? {})))

    restoreShow.value = false
    restorePreview.value = null
    loadBackupSummary()
  }
  catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    ms.error(`${t('common.failed')}: ${message}`)
  }
  finally {
    restoreLoading.value = false
  }
}

onMounted(() => {
  interface Version {
    versionName: string
    versionCode: number
  }

  getAbout<Version>().then((res) => {
    if (res.code === 0)
      version.value = res.data.versionName
  })

  loadBackupSummary()
})

function handleFileChange(options: { file: UploadFileInfo; fileList: Array<UploadFileInfo> }) {
  uploadLoading.value = true
  console.log(options.file.file)
  if (options.file.file) {
    const reader = new FileReader()
    reader.onload = () => {
      if (reader.result) {
        jsonData.value = reader.result as string
        importCheck()
      }
      else {
        ms.error(`${t('common.failed')}: ${t('common.repeatLater')}`)
      }
      uploadLoading.value = false
    }
    reader.readAsText(options.file.file)
  }
}

// 验证导入文件
function importCheck() {
  importWarning.value = []
  if (jsonData.value) {
    try {
      importObj.value = importJsonString(jsonData.value)
      if (importObj.value) {
        if (!importObj.value.isPassCheckMd5())
          importWarning.value.push(t('apps.exportImport.fileModified'))

        if (!importObj.value.isPassCheckConfigVersionOld())
          importWarning.value.push(t('apps.exportImport.warnConfigFileLow'))

        if (!importObj.value.isPassCheckConfigVersionNew())
          importWarning.value.push(t('apps.exportImport.softwareVersionLow'))

        // （暂时不做）此处可以判断，当前的配置文件是否存在的导入项目（不存在隐藏importItems里面的值）操作变量：importItems

        // 通过了验证,打开弹窗
        importRoundModalShow.value = !importRoundModalShow.value

        // console.log(importObj.value.geticons())
      }
    }
    catch (error) {
      if (error instanceof ConfigVersionLowError) {
        ms.error(t('apps.exportImport.errorConfigFileLow'))
        console.error('The configuration file version is too low to be compatible')
      }
      else if (error instanceof FormatError) {
        ms.error(t('apps.exportImport.errorConfigFileFormat'))
        console.error('The format is incorrect and cannot be imported')
      }
    }
  }
  else {
    ms.error(t('apps.exportImport.errorConfigFileFormat'))
  }
}

// 开始导出
async function handleStartExport() {
  loading.value = true
  // console.log('要导出的项目', checkedItems.value)
  // 获取软件版本号
  const exportResult = exportJson(version.value)
  if (checkedItems.value.includes('icons')) {
    console.log('export icons ...')
    const iconGroups = await exportIcons()
    exportResult.addIconsData(iconGroups)
    console.log('export icons finish', iconGroups)
  }

  // console.log('导出结果')

  jsonData.value = exportResult.string()
  exportResult.exportFile()
  loading.value = false
  exportRoundModalShow.value = false
  // ms.success(t('common.success'))
}

// 开始导入
async function handleStartImport() {
  loading.value = true
  if (checkedItems.value.includes('icons')) {
    console.log('export icons ...')
    const errMsg = await importIcons()
    if (errMsg !== null)
      ms.success(`${t('common.failed')}:${errMsg}`)
  }

  loading.value = false
  importRoundModalShow.value = false
  ms.success(`${t('common.success')}, ${t('common.refreshPage')}`)
}
</script>

<template>
  <div class="pt-2">
    <NDivider title-placement="left">
      {{ $t('apps.exportImport.backupTitle') }}
    </NDivider>

    <NAlert type="default" :bordered="false">
      <p>{{ $t('apps.exportImport.backupTip') }}</p>
    </NAlert>

    <NSpace justify="center" :size="18" class="mt-4">
      <NCheckbox v-model:checked="backupOptions.includeIcons" :disabled="backupLoading">
        {{ $t('apps.exportImport.backupIncludeIcons') }}
      </NCheckbox>
      <NCheckbox v-model:checked="backupOptions.includeStyle" :disabled="backupLoading">
        {{ $t('apps.exportImport.backupIncludeStyle') }}
      </NCheckbox>
      <NCheckbox v-model:checked="backupOptions.includeFiles" :disabled="backupLoading">
        {{ $t('apps.exportImport.backupIncludeFiles') }}
      </NCheckbox>
    </NSpace>

    <div class="mt-2 text-center text-[13px] opacity-70">
      {{ backupSummaryText }}
    </div>

    <div class="flex flex-wrap justify-center items-center gap-3 mt-4">
      <NButton type="primary" size="large" :loading="backupLoading" :disabled="backupOptionsEmpty" @click="handleExportBackup">
        <template #icon>
          <SvgIcon icon="fa6:solid-file-export" />
        </template>
        {{ $t('apps.exportImport.backupExport') }}
      </NButton>

      <NUpload
        accept=".zip,.json"
        :default-upload="false"
        :show-file-list="false"
        @change="handleRestoreFileChange"
      >
        <NButton type="info" size="large" :loading="restorePreviewing">
          <template #icon>
            <SvgIcon icon="fa6:solid-file-import" />
          </template>
          {{ $t('apps.exportImport.restore') }}
        </NButton>
      </NUpload>
    </div>

    <NDivider title-placement="left">
      {{ $t('apps.exportImport.iconConfigTitle') }}
    </NDivider>

    <NAlert type="info" :bordered="false">
      <p>{{ $t('apps.exportImport.tip') }}</p>
    </NAlert>
    <div class="flex justify-center m-[30px]">
      <div class="m-[10px]">
        <NUpload
          accept=".sun-panel.json,.sunpanel.json"
          directory-dnd
          :default-upload="false"
          :show-file-list="false"
          @change="handleFileChange"
        >
          <NButton type="info" size="large" :loading="uploadLoading">
            <template #icon>
              <SvgIcon icon="fa6:solid-file-import" />
            </template>
            {{ $t('apps.exportImport.import') }}
          </NButton>
        </NUpload>
      </div>
      <div class="m-[10px]">
        <NButton type="info" size="large" @click="exportRoundModalShow = !exportRoundModalShow">
          <template #icon>
            <SvgIcon icon="fa6:solid-file-export" />
          </template>
          {{ $t('apps.exportImport.export') }}
        </NButton>
      </div>
    </div>

    <div class="flex justify-center">
      <a href="https://hslr-s.github.io/sun-panel-tool-page/#/" target="_blank">{{ $t('apps.exportImport.transmuteStandard') }}</a>
    </div>

    <!-- 调试模式 -->
    <div v-if="debug">
      <NButton @click="importCheck">
        检查导入
      </NButton>

      <!-- <NButton @click="exportJsonS">
      导出JSON
    </NButton> -->

      <NButton @click="jsonData = ''">
        清空导入数据
      </NButton>

      <NInput
        v-model:value="jsonData"
        type="textarea"
        placeholder="基本的 Textarea"
      />

      <div v-if="jsonData">
        <h2>JSON 数据</h2>
        <pre>{{ jsonData }}</pre>
      </div>
    </div>

    <!-- 从备份恢复 -->
    <RoundCardModal v-model:show="restoreShow" style="max-width: 460px;" :title="$t('apps.exportImport.restoreTitle')">
      <div v-if="restorePreview" class="max-h-[62vh] overflow-y-auto pr-1">
        <div class="text-center text-[13px] opacity-70">
          {{ restoreMetaText }}
        </div>
        <div class="text-center mt-1 font-medium">
          {{ restoreStatsText }}
        </div>

        <NAlert v-if="restoreWarnings.length > 0" class="mt-3" :title="$t('common.warning')" type="warning">
          <div v-for="(text, index) in restoreWarnings" :key="index">
            {{ text }}
          </div>
        </NAlert>

        <NDivider title-placement="left">
          {{ $t('apps.exportImport.restoreOptionsTitle') }}
        </NDivider>

        <NSpace justify="center">
          <NCheckbox v-model:checked="restoreOptions.includeIcons" :disabled="!restorePreview.hasIcons || restoreLoading">
            {{ $t('apps.exportImport.backupIncludeIcons') }}
          </NCheckbox>
          <NCheckbox v-model:checked="restoreOptions.includeStyle" :disabled="!restorePreview.hasStyle || restoreLoading">
            {{ $t('apps.exportImport.backupIncludeStyle') }}
          </NCheckbox>
          <NCheckbox v-model:checked="restoreOptions.includeFiles" :disabled="!restorePreview.canRestoreFiles || restoreLoading">
            {{ $t('apps.exportImport.backupIncludeFiles') }}
          </NCheckbox>
        </NSpace>

        <NDivider title-placement="left">
          {{ $t('apps.exportImport.restoreModeTitle') }}
        </NDivider>

        <NRadioGroup v-model:value="restoreMode" :disabled="restoreLoading">
          <div class="flex flex-col gap-2">
            <NRadio value="merge">
              {{ $t('apps.exportImport.restoreModeMerge') }}
            </NRadio>
            <div class="text-[12px] opacity-60 pl-6">
              {{ $t('apps.exportImport.restoreModeMergeDesc') }}
            </div>
            <NRadio value="replace">
              {{ $t('apps.exportImport.restoreModeReplace') }}
            </NRadio>
            <div class="text-[12px] opacity-60 pl-6">
              {{ $t('apps.exportImport.restoreModeReplaceDesc') }}
            </div>
          </div>
        </NRadioGroup>

        <NAlert v-if="restoreMode === 'replace' && restoreOptions.includeIcons" class="mt-3" type="error" :bordered="false">
          {{ $t('apps.exportImport.restoreReplaceWarning') }}
        </NAlert>

        <NSpace justify="center" class="mt-5">
          <NButton :disabled="restoreLoading" @click="restoreShow = false">
            {{ $t('common.cancel') }}
          </NButton>
          <NButton
            type="success"
            :loading="restoreLoading"
            :disabled="restoreOptionsEmpty"
            @click="handleRestoreApply"
          >
            {{ $t('apps.exportImport.restoreStart') }}
          </NButton>
        </NSpace>
      </div>
    </RoundCardModal>

    <RoundCardModal v-model:show="importRoundModalShow" style="max-width: 400px;" :title=" $t('apps.exportImport.import')">
      <div v-if="importWarning.length > 0">
        <NAlert :title="$t('common.warning')" type="warning">
          <div v-for="(text, index) in importWarning " :key="index">
            {{ text }}
          </div>
        </NAlert>
      </div>
      <NDivider title-placement="left">
        {{ $t('apps.exportImport.selectImportData') }}
      </NDivider>

      <NSpace justify="center" style="margin-top: 20px;">
        <NCheckboxGroup v-model:value="checkedItems">
          <NCheckbox v-if="importItems.includes('icons')" value="icons" :label="$t('apps.exportImport.moduleIcon')" />
          <NCheckbox v-if="importItems.includes('style')" value="style" :label="$t('apps.exportImport.moduleStyle')" />
        </NCheckboxGroup>
      </NSpace>
      <NSpace justify="center">
        <div class="mt-[50px]">
          <NButton type="success" :disabled="checkedItems.length === 0" :loading="loading" @click="handleStartImport">
            {{ $t('common.continue') }}
          </NButton>
        </div>
      </NSpace>
    </RoundCardModal>

    <RoundCardModal v-model:show="exportRoundModalShow" style="max-width: 400px;" :title=" $t('apps.exportImport.export')">
      <NDivider title-placement="left">
        {{ $t('apps.exportImport.selectExportData') }}
      </NDivider>

      <NSpace justify="center" style="margin-top: 20px;">
        <NCheckboxGroup v-model:value="checkedItems">
          <NCheckbox v-if="importItems.includes('icons')" value="icons" :label="$t('apps.exportImport.moduleIcon')" />
          <NCheckbox v-if="importItems.includes('style')" value="style" :label="$t('apps.exportImport.moduleStyle')" />
        </NCheckboxGroup>
      </NSpace>
      <NSpace justify="center">
        <div class="mt-[50px]">
          <NButton type="success" :disabled="checkedItems.length === 0" :loading="loading" @click="handleStartExport">
            {{ $t('common.continue') }}
          </NButton>
        </div>
      </NSpace>
    </RoundCardModal>
  </div>
</template>
