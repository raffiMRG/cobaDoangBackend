package FolderRepositorys

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"web_backend/Model/NewFolder"
	tbFolder "web_backend/Model/TbFolder"

	// folder "web_backend/Model/Folder"
	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	"web_backend/Repository/UploadRepositorys"

	"sync"

	"gorm.io/gorm"
)

// ProgressChannels maps a move taskID to a chan float64 that receives a new
// percentage the instant a file finishes copying — push-based rather than
// polled, so progress shows up in real time regardless of how fast the copy
// is. Only one reader is expected per taskID (a single SSE connection); if
// multiple listeners ever connect to the same taskID concurrently, updates
// would be split between them rather than broadcast, since this is a
// personal single-user app and that scenario isn't expected in practice.
var ProgressChannels sync.Map

// func SearchFolders(keyword string, page int) ([]NewFolder.NewFolder, int64, error) {
func SearchFolders(keyword string, page int) ([]dto.NewFolderQuery, int64, error) {
	// var folders []NewFolder.NewFolder
	// var folders []dto.NewFolderResponse
	var foldersQuery []dto.NewFolderQuery
	// var total int64

	limit := 20
	offset := (page - 1) * limit

	// ==== Query lama start ====
	// query := connection.DB.Model(&NewFolder.NewFolder{})
	// if keyword != "" {
	// 	query = query.Where("name LIKE ?", "%"+keyword+"%")
	// }

	// // Hitung total data sebelum paginasi
	// if err := query.Count(&total).Error; err != nil {
	// 	return nil, 0, err
	// }

	// // Ambil data dengan pagination
	// if err := query.Limit(limit).Offset(offset).Find(&folders).Error; err != nil {
	// 	return nil, 0, err
	// }
	// ==== Query lama end ====

	// ==== Query data baru start ====
	// Ambil data dengan limit & offset
	err := connection.DB.Table("new_folders nf").
		Select(`
			nf.id,
			nf.name,
			nf.thumbnail,
			nf.is_completed,
			nf.create_at,
			EXISTS (SELECT 1 FROM bookmarks b WHERE b.folder_id = nf.id) AS is_bookmarked,
			EXISTS (SELECT 1 FROM translations t WHERE t.folder_id = nf.id AND t.status = 'completed') AS is_translated
		`).
		Where("nf.name LIKE ?", "%"+keyword+"%").
		Order("nf.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&foldersQuery).Error
	// Find(&folders).Error
	// ==== Query data baru end ====

	if err != nil {
		return nil, 0, err
	}

	// return folders, total, nil
	// fmt.Println("total data:", int64(len(folders)))
	return foldersQuery, int64(len(foldersQuery)), nil
}

func ScanFolders(root string) ([]string, error) {
	var folders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != root {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if filepath.Dir(relPath) == "." {
				folders = append(folders, path)
			}
			return filepath.SkipDir
		}
		return nil
	})
	return folders, err
}

func ScanFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, filepath.Base(path))
		}
		return nil
	})
	return files, err
}

// ScanDestinationFolderNames lists DST_DIR one level deep and returns just
// the folder names — used to compare what's actually on disk against old DB
// backups, independent of the new_folders table. Uses os.ReadDir directly
// (not ScanFolders/filepath.Walk) since Walk calls os.Lstat per entry to
// build a full os.FileInfo; for ~8000 entries on slower storage that extra
// stat syscall per entry was enough to blow past the frontend's HTTP
// timeout. ReadDir gets IsDir() from the raw directory read (d_type on most
// Linux filesystems) with no extra syscall, and already returns entries
// sorted by filename.
func ScanDestinationFolderNames() ([]string, error) {
	destPath := os.Getenv("DST_DIR")
	entries, err := os.ReadDir(destPath)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// buildFileURL reads folderPath's first file and builds a URL for it under
// the given static-serving path segment (e.g. "sementara" or "new").
func buildFileURL(staticSegment, folderName, folderPath string) (string, error) {
	apiBaseUrl := os.Getenv("API_BASEURL")

	if folderName == "" {
		return "", errors.New("folder name cannot be empty")
	}

	files, err := os.ReadDir(folderPath)
	if err != nil {
		return "", errors.New("folder does not exist")
	}
	if len(files) == 0 {
		return "", errors.New("folder is empty")
	}

	fileURL, _ := url.Parse(apiBaseUrl + "/" + staticSegment + "/")
	fileURL.Path = path.Join(fileURL.Path, folderName, files[0].Name())

	return fileURL.String(), nil
}

// BuildThumbnailURL reads folderPath's first file and builds the thumbnail
// URL for it, without touching the DB — split out from the old InsertFolder
// so UpdateAndInsert can batch the actual INSERT across all new folders
// instead of one INSERT per folder.
func BuildThumbnailURL(folderName, folderPath string) (string, error) {
	return buildFileURL("sementara", folderName, folderPath)
}

// BuildNewFolderThumbnailURL is BuildThumbnailURL's DST_DIR counterpart —
// used wherever a thumbnail is (re)computed for a folder that already lives
// under DST_DIR (e.g. a freshly-uploaded translated manga).
func BuildNewFolderThumbnailURL(folderName, folderPath string) (string, error) {
	return buildFileURL("new", folderName, folderPath)
}

func GetAllData(table string, page, limit int) model.BaseResponseModel {
	var result model.BaseResponseModel
	var ListData []tbFolder.Folder
	db := connection.DB

	allowedTables := map[string]bool{
		"folders": true,
	}

	// Validasi table
	if !allowedTables[table] {
		return model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       "Invalid table name",
			Data:          nil,
		}
	}

	// Hitung offset
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Hitung total rows
	var total int64
	db.Table(table).Count(&total)

	// Ambil data dengan limit & offset
	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", table)
	tempResult := db.Raw(query, limit, offset).Scan(&ListData)

	if tempResult.Error != nil {
		result = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       tempResult.Error.Error(),
			Data:          nil,
		}
	} else {
		// Bungkus data + pagination
		result = model.BaseResponseModel{
			CodeResponse:  200,
			HeaderMessage: "Success",
			Message:       "Data retrieved successfully",
			Data: map[string]interface{}{
				"items": ListData,
				"pagination": map[string]interface{}{
					"total": total,
					"page":  page,
					"limit": limit,
					"pages": int((total + int64(limit) - 1) / int64(limit)), // ceiling
				},
			},
		}
	}

	return result
}

func GetAllDataNewfolders(page, limit int) model.BaseResponseModel {
	var result model.BaseResponseModel
	// var listData []NewFolder.NewFolder
	var listData []dto.NewFolderResponse
	db := connection.DB

	// Hitung offset untuk pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Ambil total data (untuk info pagination)
	var total int64
	db.Model(&NewFolder.NewFolder{}).Count(&total)

	// Ambil data dengan limit & offset
	err := db.Table("new_folders nf").
		Select(`
			nf.id,
			nf.name,
			nf.thumbnail,
			nf.is_completed,
			nf.create_at,
			EXISTS (SELECT 1 FROM bookmarks b WHERE b.folder_id = nf.id) AS is_bookmarked,
			EXISTS (SELECT 1 FROM translations t WHERE t.folder_id = nf.id AND t.status = 'completed') AS is_translated
		`).
		Order("nf.id DESC").
		Limit(limit).
		Offset(offset).
		Find(&listData).Error

	if err != nil {
		result = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		}
		return result
	}

	// Bungkus data + info pagination
	data := map[string]interface{}{
		"items": listData,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": int((total + int64(limit) - 1) / int64(limit)), // total halaman
	}

	result = model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "Data retrieved successfully",
		Data:          data,
	}

	return result
}

// ============================================

func GetDataFromId(table, id string) model.BaseResponseModel {
	// var query string
	var result model.BaseResponseModel
	// var listData []tbFolder.Folder
	var listData *tbFolder.Folder
	// db := connection.DB
	tempResult := connection.DB

	allowedTables := map[string]bool{
		"folders":     true,
		"new_folders": true,
	}

	// Periksa apakah tabel valid
	if !allowedTables[table] {
		return model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       "Invalid table name",
			Data:          nil,
		}
	}

	listData, err := GetRowFromId(table, id)

	if err != nil {
		result := model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		}
		return result
	}

	if tempResult.Error != nil {
		result = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       tempResult.Error.Error(),
			Data:          nil,
		}
	} else if listData == nil {
		result = model.BaseResponseModel{
			CodeResponse:  404,
			HeaderMessage: "Error",
			Message:       "Data Not Found",
			Data:          nil,
		}
	} else {
		result = model.BaseResponseModel{
			CodeResponse:  200,
			HeaderMessage: "Success",
			Message:       "Data retrieved successfully :)",
			Data:          listData,
		}
	}

	return result
}

func GetNewfolderDataFromId(id string) model.BaseResponseModel {
	dstPath := os.Getenv("DST_DIR")

	// var query string
	var result model.BaseResponseModel
	// var listData []tbFolder.Folder
	var listData *NewFolder.NewFolder
	// db := connection.DB
	tempResult := connection.DB
	var err error
	var pages []string

	if id == "" {
		result := model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       "Invalid ID",
			Data:          nil,
		}
		return result
	}

	// listData, err := GetNewfolderRowFromId(id)

	// if err != nil {
	if listData, err = GetNewfolderRowFromId(id); err != nil {
		result := model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		}
		return result
	}

	scanPath := dstPath + "/" + listData.Name
	fmt.Println("Scanning path:", scanPath)

	// scan folder dan ambil page
	if pages, err = ScanFiles(scanPath); err != nil {
		result := model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		}
		return result
	}

	translationStatus, translationError := getTranslationStatus(listData.ID)

	// Masukkan ke struct response
	response := dto.NewFolderResponse{
		ID:                listData.ID,
		Name:              listData.Name,
		Thumbnail:         listData.Thumbnail,
		IsCompleted:       listData.IsCompleted,
		CreateAt:          listData.CreateAt,
		Page:              pages,
		TranslationStatus: translationStatus,
		TranslationError:  translationError,
	}

	if tempResult.Error != nil {
		result = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       tempResult.Error.Error(),
			Data:          nil,
		}
	} else if listData == nil {
		result = model.BaseResponseModel{
			CodeResponse:  404,
			HeaderMessage: "Error",
			Message:       "Data Not Found",
			Data:          nil,
		}
	} else {
		result = model.BaseResponseModel{
			CodeResponse:  200,
			HeaderMessage: "Success",
			Message:       "Data retrieved successfully :)",
			Data:          response,
		}
	}

	return result
}

func GetRowFromId(table, id string) (*tbFolder.Folder, error) {
	var query string
	var data tbFolder.Folder
	var errorMessage error
	db := connection.DB

	if id != "" {
		intId, err := strconv.Atoi(id)
		if err != nil {
			errorMessage = errors.New("invalid parameter input for 'id'")
			return nil, errorMessage
		}

		// Ambil 1 data berdasarkan ID
		result := db.Where("id = ?", intId).First(&data)
		if result.Error != nil {
			return nil, result.Error
		}
		fmt.Println("id query " + id)
	} else {
		// Ambil 1 data pertama dari tabel jika id tidak diberikan
		query = fmt.Sprintf("SELECT * FROM %s LIMIT 1", table)
		result := db.Raw(query).Scan(&data)
		if result.Error != nil {
			return nil, result.Error
		}
		fmt.Println("tampilkan satu data pertama dari tabel")
	}

	return &data, nil
}

func GetNewfolderRowFromId(id string) (*NewFolder.NewFolder, error) {
	var data NewFolder.NewFolder
	var errorMessage error
	db := connection.DB

	intId, err := strconv.Atoi(id)
	if err != nil {
		errorMessage = errors.New("invalid parameter input for 'id'")
		return nil, errorMessage
	}

	// Ambil 1 data berdasarkan ID
	result := db.Where("id = ?", intId).First(&data)
	if result.Error != nil {
		return nil, result.Error
	}
	fmt.Println("id query " + id)

	return &data, nil
}

// getTranslationStatus looks up folderId's own translations row, returning
// ("none", "") if no request has ever been made for it.
func getTranslationStatus(folderId int) (status string, errorMessage string) {
	var row struct {
		Status       string
		ErrorMessage string
	}

	connection.DB.Table("translations").
		Select("status, error_message").
		Where("folder_id = ?", folderId).
		Scan(&row)

	if row.Status == "" {
		return "none", ""
	}
	return row.Status, row.ErrorMessage
}

// RenameNewFolder updates a new_folders row's name, and optionally the
// matching directory in DST_DIR to keep disk and DB in sync. When
// applyToDisk is false, only the DB row changes — useful for correcting the
// catalog to match a folder that was already renamed by hand on disk (or
// just fixing a display typo without touching real files).
func RenameNewFolder(id, newName string, applyToDisk bool) model.BaseResponseModel {
	db := connection.DB

	row, err := GetNewfolderRowFromId(id)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	safeName, err := UploadRepositorys.SanitizeName(newName)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: err.Error(), Data: nil}
	}

	updates := map[string]interface{}{"name": safeName}

	if applyToDisk {
		dstPath := os.Getenv("DST_DIR")
		oldPath := dstPath + "/" + row.Name
		newPath := dstPath + "/" + safeName

		if _, statErr := os.Stat(oldPath); statErr != nil {
			return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "original folder not found on disk: " + statErr.Error(), Data: nil}
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
		}

		// The stored thumbnail URL was baked in at move-time and still
		// points at the old folder name — recompute it so listings don't
		// end up with a broken image. Non-fatal if it fails (e.g. the
		// folder happens to be empty): the rename itself already succeeded.
		if newThumbnail, thumbErr := buildFileURL("new", safeName, newPath); thumbErr == nil {
			updates["thumbnail"] = newThumbnail
		}
	}

	if err := db.Model(&NewFolder.NewFolder{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "renamed successfully", Data: updates}
}

// DeleteNewFolder removes a new_folders row, and optionally its matching
// directory in DST_DIR. Bookmarks on this folder are cleaned up
// automatically via the bookmarks.folder_id ON DELETE CASCADE FK.
func DeleteNewFolder(id string, applyToDisk bool) model.BaseResponseModel {
	db := connection.DB

	row, err := GetNewfolderRowFromId(id)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	if applyToDisk {
		dstPath := os.Getenv("DST_DIR")
		if err := os.RemoveAll(dstPath + "/" + row.Name); err != nil {
			return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
		}
	}

	if err := db.Delete(&NewFolder.NewFolder{}, row.ID).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "deleted successfully", Data: nil}
}

// =======================================================

// func moveRows(sourceTable string, targetTable string, limit int) model.BaseResponseModel {
func MoveRows(ids []int, sourceTable, targetTable string) model.BaseResponseModel {
	var srcPath string = os.Getenv("SRC_DIR")
	var destPath string = os.Getenv("DST_DIR")
	// var result error
	db := connection.DB
	// var listDataSuccess []int
	var response []dto.InputDataRes
	// // Inisialisasi transaksi
	result := db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			// filter id pastikan tidak minus
			if id < 0 {
				return fmt.Errorf("invalid id: %d cannot be negative", id)
			}

			// 	// 1. Ambil data dari tabel sumber
			strId := strconv.Itoa(id)
			row, err := GetRowFromId(sourceTable, strId)

			if err != nil {
				return err
			}

			// if len(rows) == 0 {
			if row == nil {
				// Append default data to the response
				defaultData := dto.InputDataRes{
					// ID:     id,
					Title:  "No Data Available",
					Status: false,
				}
				response = append(response, defaultData)
				fmt.Println("masuk response false 1")
				continue
			}
			// newRows := NewFolder.NewFolder{
			newRow := NewFolder.NewFolder{
				Name:        row.Name, // Hanya ambil Name
				Thumbnail:   strings.Replace(row.Thumbnail, "/sementara/", "/new/", 1),
				IsCompleted: false, // Isi nilai devault true
			}

			// COPY DATA
			// copyFailed := false
			// for _, data := range newRows {

			source := srcPath + "/" + newRow.Name + "/"
			destination := destPath + "/" + newRow.Name + "/"

			if err := copyPaste(source, destination, nil); err != nil {
				fmt.Println("Error:", err)
				// copyFailed = true
				response = append(response, dto.InputDataRes{
					ID:     uint(id),
					Title:  "Copy failed",
					Status: false,
				})
				break
				// continue
			} else {
				fmt.Println("file dan folder berhasil disalin.")
			}
			// }

			// 	// 2. Masukkan data ke tabel tujuan
			if err := tx.Table(targetTable).Create(&newRow).Error; err != nil {
				// return err
				defaultData := dto.InputDataRes{
					// ID:     id,
					Title:  err.Error(),
					Status: false,
				}
				response = append(response, defaultData)
				fmt.Println("masuk response false 2")
				continue
			} else {
				// defaultData := dto.InputDataRes{
				// 	// ID:     id,
				// 	Title:  "Success",
				// 	Status: true,
				// }

				// response = append(response, defaultData)
				fmt.Println("masuk response success pindah data")
			}

			// // 	// 3. Hapus data dari folder sumber
			if err := os.RemoveAll(source); err != nil {
				fmt.Println("Error:", err)
				response = append(response, dto.InputDataRes{
					ID:     uint(id),
					Title:  "Delete failed",
					Status: false,
				})
				continue
			} else {
				fmt.Println("folder sumber berhasil dihapus.")
			}
			// =============================

			if err := tx.Table(sourceTable).Where("id = ?", row.ID).Delete(nil).Error; err != nil {
				return fmt.Errorf("failed to delete rows from source table: %w", err)
			} else {
				defaultData := dto.InputDataRes{
					ID:     uint(id),
					Title:  "Success",
					Status: true,
				}
				response = append(response, defaultData)
				fmt.Println("masuk response success hapus data")
			}
		}
		// return fmt.Errorf("rows moved")
		return nil
	})

	if result != nil {
		return FailedResponse(result)
	} else {
		successResult := model.BaseResponseModel{
			CodeResponse:  200,
			HeaderMessage: "Success",
			Message:       "Data retrieved successfully",
			Data:          response,
		}
		return successResult
	}
}

func MoveRowsWithProgress(taskID string, ids []int, sourceTable, targetTable string) {
	var srcPath = os.Getenv("SRC_DIR")
	var destPath = os.Getenv("DST_DIR")

	db := connection.DB
	total := len(ids)

	// Hitung total file di semua folder yang dipilih lebih dulu, supaya
	// progress bisa dilaporkan per-file, bukan cuma per-folder. Tanpa ini
	// progress diam di 0% sepanjang waktu penyalinan satu folder (bisa berisi
	// puluhan halaman manga) lalu lompat langsung ke persentase berikutnya
	// begitu folder itu selesai total.
	totalFiles := 0
	for _, id := range ids {
		if id < 0 {
			continue
		}
		row, err := GetRowFromId(sourceTable, strconv.Itoa(id))
		if err != nil || row == nil {
			continue
		}
		source := srcPath + "/" + row.Name + "/"
		if files, err := ScanFiles(source); err == nil {
			totalFiles += len(files)
		}
	}

	// Buffered sebesar totalFiles+1 supaya penyalinan file TIDAK PERNAH
	// menunggu SSE handler membaca — kalau browser belum sempat connect
	// (atau tidak connect sama sekali), update tetap antre di channel dan
	// langsung "mengejar" begitu listener terhubung, bukan hilang/nge-block.
	progressChan := make(chan float64, totalFiles+1)
	ProgressChannels.Store(taskID, progressChan)
	defer close(progressChan)

	if total == 0 || totalFiles == 0 {
		progressChan <- 100.0
		return
	}

	filesCopied := 0
	onFileDone := func() {
		filesCopied++
		progressChan <- (float64(filesCopied) / float64(totalFiles)) * 100
	}

	_ = db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			// ===== PROSES PINDAH SAMA SEPERTI MoveRows =====
			if id < 0 {
				continue
			}

			strId := strconv.Itoa(id)
			row, err := GetRowFromId(sourceTable, strId)
			if err != nil || row == nil {
				continue
			}

			newRow := NewFolder.NewFolder{
				Name:        row.Name,
				Thumbnail:   strings.Replace(row.Thumbnail, "/sementara/", "/new/", 1),
				IsCompleted: false,
			}

			source := srcPath + "/" + newRow.Name + "/"
			destination := destPath + "/" + newRow.Name + "/"

			_ = copyPaste(source, destination, onFileDone)
			_ = tx.Table(targetTable).Create(&newRow).Error
			_ = os.RemoveAll(source)
			_ = tx.Table(sourceTable).Where("id = ?", row.ID).Delete(nil).Error
		}

		// Pastikan selesai 100%
		progressChan <- 100.0
		return nil
	})
}

func FilteredData(table, table2 string) model.BaseResponseModel {
	var result model.BaseResponseModel
	var listData []tbFolder.Folder
	db := connection.DB
	tempResult := connection.DB

	allowedTables := map[string]bool{
		"folders":    true,
		"new_folder": true,
	}

	// Periksa apakah tabel valid
	if !allowedTables[table] {
		return model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       "Invalid table name",
			Data:          nil,
		}
	}

	// query := fmt.Sprintf("SELECT * FROM %s", table)
	// query := fmt.Sprintf("SELECT id, name FROM %s WHERE name NOT IN (SELECT name FROM %s)", table, table2)
	// query := fmt.Sprintf("SELECT t1.id, t1.name FROM %s t1 JOIN %s t2 ON t1.name = t2.name WHERE t2.is_completed = 0", table, table2)
	// query := fmt.Sprintf("SELECT t1.id, t1.name FROM %s t1 LEFT JOIN %s t2 ON t1.name = t2.name WHERE t2.is_completed != 1", table, table2)
	query := fmt.Sprintf("SELECT t1.id, t1.name FROM %s t1 LEFT JOIN %s t2 ON t1.name = t2.name AND t2.is_completed = 1 WHERE t2.name IS NULL;", table, table2)
	tempResult = db.Raw(query).Find(&listData)
	fmt.Println("tampilkan semua data dari tabel")

	if tempResult.Error != nil {
		result := model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       tempResult.Error.Error(),
			Data:          nil,
		}
		return result
	}

	if tempResult.Error != nil {
		result = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       tempResult.Error.Error(),
			Data:          nil,
		}
	} else if len(listData) == 0 {
		result = model.BaseResponseModel{
			CodeResponse:  404,
			HeaderMessage: "Error",
			Message:       "Data Not Found",
			Data:          nil,
		}
	} else {
		result = model.BaseResponseModel{
			CodeResponse:  200,
			HeaderMessage: "Success",
			Message:       "Data retrieved successfully :)",
			Data:          listData,
		}
	}

	return result
}

// ExistingFolderNames returns every name currently in `folders` as a set, in
// one query — used by UpdateAndInsert instead of one exists-check query per
// scanned folder, which got slow once SRC_DIR had more than a handful of
// pending folders.
func ExistingFolderNames(db *gorm.DB) (map[string]bool, error) {
	var names []string
	if err := db.Model(&tbFolder.Folder{}).Pluck("name", &names).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set, nil
}

func FailedResponse(message error) model.BaseResponseModel {
	result := model.BaseResponseModel{
		CodeResponse:  400,
		HeaderMessage: "Error",
		Message:       message.Error(),
		Data:          nil,
	}
	return result
}
