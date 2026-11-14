package panel

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sun-panel/api/api_v1/common/apiData/panelApiStructs"
	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/cmn"
	"sun-panel/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
)

// 导入相关错误码（前端按 code 显示对应文案）
const (
	importErrorFileFormat = 1400 // 备份文件格式不正确
	importErrorVersion    = 1401 // 备份文件版本不兼容
	importErrorSession    = 1402 // 导入会话不存在或已过期
	importErrorEmpty      = 1403 // 备份中没有可导入的数据
)

const (
	importModeMerge   = "merge"
	importModeReplace = "replace"

	maxImportUploadSize   = 512 << 20 // 上传的备份文件上限 512MB
	maxImportManifestSize = 64 << 20  // data.json 上限 64MB
	maxImportEntryCount   = 20000     // 压缩包内文件数量上限
	maxImportUnzipSize    = 2 << 30   // 解压总大小上限 2GB
	importTempLifetime    = 3 * time.Hour
	importBatchSize       = 200
)

// 导入会话 ID：import-<userId>-<md5>
var importIdPattern = regexp.MustCompile(`^import-([0-9]+)-([0-9a-f]{32})$`)

type DataImport struct{}

// Preview 上传备份文件并解析出可导入的内容
func (a *DataImport) Preview(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		apiReturn.ErrorCode(c, importErrorFileFormat, "backup file is required", nil)
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxImportUploadSize {
		apiReturn.ErrorCode(c, importErrorFileFormat, "backup file is empty or too large", nil)
		return
	}

	cleanImportTemp()

	importId, tempPath, err := saveImportUpload(c, userInfo.ID, fileHeader)
	if err != nil {
		apiReturn.Error(c, "failed to save backup file: "+err.Error())
		return
	}

	manifest, fileType, err := loadImportManifest(tempPath)
	if err != nil {
		_ = os.Remove(tempPath)
		switch {
		case errors.Is(err, errImportVersion):
			apiReturn.ErrorCode(c, importErrorVersion, "backup format version is not compatible", nil)
		case errors.Is(err, errImportEmpty):
			apiReturn.ErrorCode(c, importErrorEmpty, "no importable data in backup", nil)
		default:
			apiReturn.ErrorCode(c, importErrorFileFormat, "invalid backup file: "+err.Error(), nil)
		}
		return
	}

	groupCount, iconCount := manifestContentCounts(manifest)
	var fileSize int64
	for _, entry := range manifest.Files {
		fileSize += entry.Size
	}

	preview := panelApiStructs.DataImportPreview{
		ImportId:        importId,
		FileType:        fileType,
		AppVersion:      manifest.AppVersion,
		ExportTime:      manifest.ExportTime,
		SourceUsername:  manifest.User.Username,
		GroupCount:      groupCount,
		IconCount:       iconCount,
		FileCount:       len(manifest.Files),
		FileSize:        fileSize,
		HasIcons:        groupCount > 0 || iconCount > 0,
		HasStyle:        manifest.Panel != nil || manifest.SearchEngine != nil,
		HasFiles:        len(manifest.Files) > 0,
		CanRestoreFiles: fileType == "zip" && len(manifest.Files) > 0,
		Warnings:        buildImportWarnings(manifest, fileType, userInfo.Username),
	}

	if !preview.HasIcons && !preview.HasStyle && !preview.HasFiles {
		_ = os.Remove(tempPath)
		apiReturn.ErrorCode(c, importErrorEmpty, "no importable data in backup", nil)
		return
	}

	apiReturn.SuccessData(c, preview)
}

// Apply 按备份内容恢复数据
func (a *DataImport) Apply(c *gin.Context) {
	userInfo, _ := base.GetCurrentUserInfo(c)

	req := panelApiStructs.DataImportRequest{}
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		apiReturn.ErrorParamFomat(c, err.Error())
		return
	}

	tempPath, err := resolveImportTempPath(userInfo.ID, req.ImportId)
	if err != nil {
		apiReturn.ErrorCode(c, importErrorSession, "import session not found or expired", nil)
		return
	}
	if _, err := os.Stat(tempPath); err != nil {
		apiReturn.ErrorCode(c, importErrorSession, "import session not found or expired", nil)
		return
	}
	defer os.Remove(tempPath)

	manifest, fileType, err := loadImportManifest(tempPath)
	if err != nil {
		switch {
		case errors.Is(err, errImportVersion):
			apiReturn.ErrorCode(c, importErrorVersion, "backup format version is not compatible", nil)
		case errors.Is(err, errImportEmpty):
			apiReturn.ErrorCode(c, importErrorEmpty, "no importable data in backup", nil)
		default:
			apiReturn.ErrorCode(c, importErrorFileFormat, "invalid backup file: "+err.Error(), nil)
		}
		return
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != importModeReplace {
		mode = importModeMerge
	}

	includeIcons := boolOrDefault(req.IncludeIcons, true)
	includeStyle := boolOrDefault(req.IncludeStyle, true)
	includeFiles := boolOrDefault(req.IncludeFiles, true)

	result := panelApiStructs.DataImportResult{Mode: mode, Warnings: []panelApiStructs.DataImportWarning{}}

	// 先落盘文件，再写数据库：即使事务失败也只会留下无害的孤儿文件
	fileOutcome := importFileOutcome{}
	if includeFiles && fileType == "zip" && len(manifest.Files) > 0 {
		fileOutcome = restoreImportFiles(userInfo.ID, tempPath, manifest.Files)
		result.FileRestored = fileOutcome.restored
		result.FileSkipped = fileOutcome.skipped
		result.FileFailed = fileOutcome.failed
		if fileOutcome.failed > 0 {
			result.Warnings = append(result.Warnings, panelApiStructs.DataImportWarning{
				Code:   "filesFailed",
				Params: map[string]string{"count": strconv.Itoa(fileOutcome.failed)},
			})
		}
	} else if includeFiles && fileType != "zip" && len(manifest.Files) > 0 {
		result.FileSkipped = len(manifest.Files)
		result.Warnings = append(result.Warnings, panelApiStructs.DataImportWarning{Code: "filesNeedZip"})
	}

	existingFileSrc := map[string]bool{}
	if len(fileOutcome.records) > 0 {
		srcs := []string{}
		for _, record := range fileOutcome.records {
			srcs = append(srcs, record.Src)
		}
		files := []models.File{}
		if err := global.Db.Find(&files, "user_id=? AND src IN ?", userInfo.ID, srcs).Error; err != nil {
			apiReturn.ErrorDatabase(c, err.Error())
			return
		}
		for _, file := range files {
			existingFileSrc[file.Src] = true
		}
	}

	txErr := global.Db.Transaction(func(tx *gorm.DB) error {
		if mode == importModeReplace && includeIcons {
			if err := tx.Where("user_id=?", userInfo.ID).Delete(&models.ItemIcon{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id=?", userInfo.ID).Delete(&models.ItemIconGroup{}).Error; err != nil {
				return err
			}
		}

		if includeIcons && len(manifest.IconGroups) > 0 {
			groupCount, iconCount, err := importIconGroups(tx, userInfo.ID, manifest.IconGroups)
			if err != nil {
				return err
			}
			result.GroupCount = groupCount
			result.IconCount = iconCount
		}

		if includeStyle {
			restored, err := importUserStyle(tx, userInfo.ID, manifest.Panel, manifest.SearchEngine)
			if err != nil {
				return err
			}
			result.StyleRestored = restored
		}

		if includeFiles && len(fileOutcome.records) > 0 {
			newRecords := make([]models.File, 0, len(fileOutcome.records))
			for _, record := range fileOutcome.records {
				if existingFileSrc[record.Src] {
					continue
				}
				existingFileSrc[record.Src] = true
				newRecords = append(newRecords, record)
			}
			if len(newRecords) > 0 {
				if err := tx.CreateInBatches(newRecords, importBatchSize).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	if txErr != nil {
		apiReturn.ErrorDatabase(c, txErr.Error())
		return
	}

	apiReturn.SuccessData(c, result)
}

var (
	errImportVersion = errors.New("import format version not compatible")
	errImportEmpty   = errors.New("no importable data in backup")
)

// 保存上传的备份文件，返回导入会话 ID 与临时文件路径
func saveImportUpload(c *gin.Context, userId uint, fileHeader *multipart.FileHeader) (importId, tempPath string, err error) {
	dir := importTempDir()
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", "", err
	}

	sum := md5.Sum([]byte(fmt.Sprintf("%d-%d-%s", userId, time.Now().UnixNano(), fileHeader.Filename)))
	importId = fmt.Sprintf("import-%d-%x", userId, sum)
	tempPath = filepath.Join(dir, importId+".bin")

	if err = c.SaveUploadedFile(fileHeader, tempPath); err != nil {
		return "", "", err
	}
	return importId, tempPath, nil
}

// 校验导入会话并返回临时文件路径
func resolveImportTempPath(userId uint, importId string) (string, error) {
	match := importIdPattern.FindStringSubmatch(strings.TrimSpace(importId))
	if match == nil {
		return "", errors.New("invalid import id")
	}
	id, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil || uint(id) != userId {
		return "", errors.New("import session does not belong to current user")
	}
	return filepath.Join(importTempDir(), importId+".bin"), nil
}

func importTempDir() string {
	dir := global.Config.GetValueString("base", "source_temp_path")
	if dir == "" {
		dir = "./runtime/temp"
	}
	return filepath.Join(dir, "import")
}

// 清理过期的导入临时文件
func cleanImportTemp() {
	dir := importTempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	deadline := time.Now().Add(-importTempLifetime)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(deadline) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// 读取备份清单，fileType 为 zip 或 json
func loadImportManifest(tempPath string) (manifest *exportManifest, fileType string, err error) {
	raw, err := os.ReadFile(tempPath)
	if err != nil {
		return nil, "", err
	}
	if len(raw) == 0 {
		return nil, "", errors.New("backup file is empty")
	}

	if bytes.HasPrefix(raw, []byte("PK")) {
		manifest, err = readManifestFromZip(tempPath)
		fileType = "zip"
	} else {
		manifest, err = parseManifestBytes(raw)
		fileType = "json"
	}
	if err != nil {
		return nil, fileType, err
	}

	if err = validateImportManifest(manifest); err != nil {
		return nil, fileType, err
	}
	return manifest, fileType, nil
}

func readManifestFromZip(tempPath string) (*exportManifest, error) {
	reader, err := zip.OpenReader(tempPath)
	if err != nil {
		return nil, fmt.Errorf("not a valid zip archive: %w", err)
	}
	defer reader.Close()

	if len(reader.File) > maxImportEntryCount {
		return nil, errors.New("too many entries in archive")
	}

	var dataFile *zip.File
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := strings.ReplaceAll(file.Name, "\\", "/")
		if !strings.EqualFold(path.Base(name), "data.json") {
			continue
		}
		// 取层级最浅的 data.json
		if dataFile == nil || len(name) < len(dataFile.Name) {
			dataFile = file
		}
	}
	if dataFile == nil {
		return nil, errors.New("data.json not found in archive")
	}
	if dataFile.UncompressedSize64 > maxImportManifestSize {
		return nil, errors.New("data.json is too large")
	}

	rc, err := dataFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	raw, err := io.ReadAll(io.LimitReader(rc, maxImportManifestSize))
	if err != nil {
		return nil, err
	}

	return parseManifestBytes(raw)
}

func parseManifestBytes(raw []byte) (*exportManifest, error) {
	manifest := &exportManifest{}
	if err := json.Unmarshal(raw, manifest); err != nil {
		return nil, fmt.Errorf("cannot parse data.json: %w", err)
	}
	return manifest, nil
}

func validateImportManifest(manifest *exportManifest) error {
	if manifest == nil {
		return errors.New("empty manifest")
	}
	if manifest.AppName != exportAppName {
		return errors.New("this file is not a Sun-Panel backup")
	}
	if manifest.FormatVersion <= 0 {
		return errImportVersion
	}
	if manifest.FormatVersion > exportFormatVersion {
		return errImportVersion
	}
	return nil
}

func manifestContentCounts(manifest *exportManifest) (groupCount, iconCount int) {
	for _, group := range manifest.IconGroups {
		if strings.TrimSpace(group.Title) == "" && len(group.Children) == 0 {
			continue
		}
		groupCount++
		iconCount += len(group.Children)
	}
	return groupCount, iconCount
}

func buildImportWarnings(manifest *exportManifest, fileType, currentUsername string) []panelApiStructs.DataImportWarning {
	warnings := []panelApiStructs.DataImportWarning{}

	sourceUsername := strings.TrimSpace(manifest.User.Username)
	if sourceUsername != "" && !strings.EqualFold(sourceUsername, currentUsername) {
		warnings = append(warnings, panelApiStructs.DataImportWarning{
			Code:   "otherUser",
			Params: map[string]string{"username": sourceUsername},
		})
	}

	missing := 0
	for _, entry := range manifest.Files {
		if entry.Missing {
			missing++
		}
	}
	if missing > 0 {
		warnings = append(warnings, panelApiStructs.DataImportWarning{
			Code:   "filesMissing",
			Params: map[string]string{"count": strconv.Itoa(missing)},
		})
	}

	if fileType != "zip" && len(manifest.Files) > 0 {
		warnings = append(warnings, panelApiStructs.DataImportWarning{Code: "filesNeedZip"})
	}

	if manifest.AppVersion != "" && manifest.AppVersion != cmn.GetSysVersionInfo().Version {
		warnings = append(warnings, panelApiStructs.DataImportWarning{
			Code:   "versionDiff",
			Params: map[string]string{"version": manifest.AppVersion},
		})
	}

	return warnings
}

// 导入图标分组与图标，返回实际写入的数量
func importIconGroups(tx *gorm.DB, userId uint, groups []exportItemIconGroup) (groupCount, iconCount int, err error) {
	for _, groupItem := range groups {
		title := strings.TrimSpace(groupItem.Title)
		if title == "" && len(groupItem.Children) == 0 {
			continue
		}
		if title == "" {
			title = "Ungrouped"
		}

		group := models.ItemIconGroup{
			Icon:        groupItem.Icon,
			Title:       title,
			Description: groupItem.Description,
			Sort:        groupItem.Sort,
			UserId:      userId,
		}
		if err = tx.Create(&group).Error; err != nil {
			return groupCount, iconCount, err
		}
		groupCount++

		if len(groupItem.Children) == 0 {
			continue
		}

		icons := make([]models.ItemIcon, 0, len(groupItem.Children))
		for _, iconItem := range groupItem.Children {
			iconJson := "{}"
			if jb, marshalErr := json.Marshal(iconItem.Icon); marshalErr == nil {
				iconJson = string(jb)
			}
			icons = append(icons, models.ItemIcon{
				IconJson:        iconJson,
				Title:           iconItem.Title,
				Url:             iconItem.Url,
				LanUrl:          iconItem.LanUrl,
				Description:     iconItem.Description,
				OpenMethod:      iconItem.OpenMethod,
				Sort:            iconItem.Sort,
				ItemIconGroupId: int(group.ID),
				UserId:          userId,
			})
		}

		if err = tx.CreateInBatches(icons, importBatchSize).Error; err != nil {
			return groupCount, iconCount, err
		}
		iconCount += len(icons)
	}

	return groupCount, iconCount, nil
}

// 恢复界面样式与搜索引擎配置
func importUserStyle(tx *gorm.DB, userId uint, panelConfig, searchEngine map[string]interface{}) (restored bool, err error) {
	if panelConfig == nil && searchEngine == nil {
		return false, nil
	}

	panelJson := "{}"
	if panelConfig != nil {
		if jb, marshalErr := json.Marshal(panelConfig); marshalErr == nil {
			panelJson = string(jb)
		}
	}
	searchEngineJson := "{}"
	if searchEngine != nil {
		if jb, marshalErr := json.Marshal(searchEngine); marshalErr == nil {
			searchEngineJson = string(jb)
		}
	}

	var existing models.UserConfig
	findErr := tx.First(&existing, "user_id=?", userId).Error
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return false, findErr
	}

	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		cfg := models.UserConfig{
			UserId:           userId,
			PanelJson:        panelJson,
			SearchEngineJson: searchEngineJson,
		}
		if err = tx.Create(&cfg).Error; err != nil {
			return false, err
		}
		return true, nil
	}

	if err = tx.Model(&models.UserConfig{}).Where("user_id=?", userId).Updates(map[string]interface{}{
		"panel_json":         panelJson,
		"search_engine_json": searchEngineJson,
	}).Error; err != nil {
		return false, err
	}
	return true, nil
}

type importFileOutcome struct {
	records  []models.File
	restored int
	skipped  int
	failed   int
}

// 从压缩包中还原文件到上传目录，并生成待写入的文件记录
func restoreImportFiles(userId uint, tempPath string, entries []exportFileEntry) importFileOutcome {
	outcome := importFileOutcome{}

	reader, err := zip.OpenReader(tempPath)
	if err != nil {
		outcome.failed = len(entries)
		return outcome
	}
	defer reader.Close()

	zipFiles := map[string]*zip.File{}
	var totalSize int64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		totalSize += int64(file.UncompressedSize64)
		if totalSize > maxImportUnzipSize {
			outcome.failed = len(entries)
			return outcome
		}
		name := strings.ReplaceAll(file.Name, "\\", "/")
		// 含 .. 的条目直接忽略，避免路径逃逸
		if hasDotDotSegment(name) {
			continue
		}
		zipFiles[path.Clean(name)] = file
	}

	sourcePath := strings.TrimSuffix(strings.ReplaceAll(global.Config.GetValueString("base", "source_path"), "\\", "/"), "/")
	restoredTargets := map[string]bool{}

	for _, entry := range entries {
		if entry.Missing || entry.ArchivePath == "" {
			outcome.skipped++
			continue
		}

		archiveName := strings.ReplaceAll(entry.ArchivePath, "\\", "/")
		archiveName = strings.TrimLeft(archiveName, "/")

		relative, relErr := importRelativePath(archiveName)
		if relErr != nil {
			outcome.failed++
			continue
		}

		zipFile, ok := zipFiles[path.Clean(archiveName)]
		if !ok {
			outcome.failed++
			continue
		}

		target := sourcePath + "/" + relative
		// 同一个目标路径只处理一次
		if restoredTargets[target] {
			outcome.skipped++
			continue
		}
		restoredTargets[target] = true
		if dir := path.Dir(target); dir != "" && dir != "." {
			if mkErr := os.MkdirAll(dir, os.ModePerm); mkErr != nil {
				outcome.failed++
				continue
			}
		}

		// 目标已存在且大小一致时跳过写入，避免重复覆盖
		if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() && info.Size() == int64(zipFile.UncompressedSize64) {
			outcome.skipped++
		} else if extractErr := extractZipEntry(zipFile, target); extractErr != nil {
			outcome.failed++
			continue
		} else {
			outcome.restored++
		}

		fileName := entry.FileName
		if strings.TrimSpace(fileName) == "" {
			fileName = path.Base(relative)
		}
		ext := entry.Ext
		if strings.TrimSpace(ext) == "" {
			ext = path.Ext(relative)
		}

		outcome.records = append(outcome.records, models.File{
			UserId:   userId,
			FileName: fileName,
			Ext:      ext,
			Src:      target,
		})
	}

	return outcome
}

// 将压缩包内路径（files/2026/11/14/x.png）转换为相对上传目录的路径
// 必须在 path.Clean 之前完成校验，否则 "../" 会被折叠成合法路径
func importRelativePath(archiveName string) (string, error) {
	p := strings.ReplaceAll(archiveName, "\\", "/")
	p = strings.TrimLeft(p, "/")

	if !strings.HasPrefix(p, "files/") {
		return "", errors.New("path must be inside files/")
	}
	p = strings.TrimPrefix(p, "files/")

	if p == "" || hasDotDotSegment(p) || path.IsAbs(p) || strings.Contains(p, ":") {
		return "", errors.New("illegal path")
	}

	// 兼容早期备份把上传目录名也打进归档路径的情况（files/uploads/2026/...）
	sourceBase := path.Base(strings.TrimSuffix(strings.ReplaceAll(global.Config.GetValueString("base", "source_path"), "\\", "/"), "/"))
	if sourceBase != "" && sourceBase != "." && strings.HasPrefix(p, sourceBase+"/") {
		p = strings.TrimPrefix(p, sourceBase+"/")
	}

	p = path.Clean(p)
	if p == "" || p == "." {
		return "", errors.New("empty path")
	}
	return p, nil
}

// 判断路径中是否存在 ".." 段
func hasDotDotSegment(p string) bool {
	for _, segment := range strings.Split(p, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func extractZipEntry(file *zip.File, target string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, io.LimitReader(rc, maxImportUnzipSize)); err != nil {
		_ = os.Remove(target)
		return err
	}
	return nil
}
