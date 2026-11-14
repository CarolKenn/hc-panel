package panelApiStructs

// DataImportWarning 导入过程中的提示信息，code 由前端翻译
type DataImportWarning struct {
	Code   string            `json:"code"`
	Params map[string]string `json:"params,omitempty"`
}

// DataImportPreview 备份文件的解析结果
type DataImportPreview struct {
	ImportId        string              `json:"importId"`
	FileType        string              `json:"fileType"` // zip | json
	AppVersion      string              `json:"appVersion"`
	ExportTime      string              `json:"exportTime"`
	SourceUsername  string              `json:"sourceUsername"`
	GroupCount      int                 `json:"groupCount"`
	IconCount       int                 `json:"iconCount"`
	FileCount       int                 `json:"fileCount"`
	FileSize        int64               `json:"fileSize"`
	HasIcons        bool                `json:"hasIcons"`
	HasStyle        bool                `json:"hasStyle"`
	HasFiles        bool                `json:"hasFiles"`
	CanRestoreFiles bool                `json:"canRestoreFiles"` // 仅 ZIP 备份可以还原文件
	Warnings        []DataImportWarning `json:"warnings"`
}

// DataImportRequest 执行导入
// Mode: merge（追加，保留现有数据，默认）| replace（替换，先清空现有图标与分组）
type DataImportRequest struct {
	ImportId     string `json:"importId"`
	Mode         string `json:"mode"`
	IncludeIcons *bool  `json:"includeIcons"`
	IncludeStyle *bool  `json:"includeStyle"`
	IncludeFiles *bool  `json:"includeFiles"`
}

// DataImportResult 导入结果
type DataImportResult struct {
	Mode          string              `json:"mode"`
	GroupCount    int                 `json:"groupCount"`
	IconCount     int                 `json:"iconCount"`
	FileRestored  int                 `json:"fileRestored"`
	FileSkipped   int                 `json:"fileSkipped"`
	FileFailed    int                 `json:"fileFailed"`
	StyleRestored bool                `json:"styleRestored"`
	Warnings      []DataImportWarning `json:"warnings"`
}
