package panelApiStructs

// DataExportRequest 数据导出请求
// 字段为 nil 时表示使用默认值（默认全部导出），方便后续扩展
type DataExportRequest struct {
	IncludeIcons *bool `json:"includeIcons"` // 是否导出图标与分组
	IncludeStyle *bool `json:"includeStyle"` // 是否导出界面样式配置（面板样式、搜索引擎）
	IncludeFiles *bool `json:"includeFiles"` // 是否导出已上传的文件
}

// DataExportSummary 导出前的数据概览
type DataExportSummary struct {
	Username   string `json:"username"`
	AppVersion string `json:"appVersion"`
	GroupCount int64  `json:"groupCount"`
	IconCount  int64  `json:"iconCount"`
	FileCount  int64  `json:"fileCount"`
	FileSize   int64  `json:"fileSize"`
	HasStyle   bool   `json:"hasStyle"`
}
