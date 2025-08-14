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

	"gorm.io/gorm"
)

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

func InsertFolder(db *gorm.DB, folderName, folderPath string) error {
	// check jika nama folder kosong
	if folderName == "" {
		return errors.New("folder name cannot be empty")
	}

	// check apakah directory ada
	// fullPath := filepath.Join(path, folderName)
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return errors.New("folder does not exist")
	}

	// fmt.Println("path : ", path)
	// fmt.Println("folder name : ", folderName)
	// fmt.Println("filse name : ", files[0].Name())

	// thumbnailPath := filepath.Join("http://192.168.1.133:8080/sementara/", folderName, files[0].Name())

	thumbnailPath, _ := url.Parse("http://192.168.1.133:8080/sementara/")
	thumbnailPath.Path = path.Join(thumbnailPath.Path, folderName, files[0].Name())
	fmt.Println(thumbnailPath.String())

	fmt.Println("thumbnail files:", thumbnailPath)

	folder := tbFolder.Folder{Name: folderName, Thumbnail: thumbnailPath.String()}
	return db.Create(&folder).Error
	// return nil
}

func GetAllData(table string) model.BaseResponseModel {
	var query string
	var result model.BaseResponseModel
	var ListData []tbFolder.Folder
	db := connection.DB

	allowedTables := map[string]bool{
		"folders": true,
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

	// Buat query dengan nama tabel
	// query = fmt.Sprintf("SELECT * FROM %s", table)

	query = "SELECT * FROM folders"
	// fmt.Println(query)
	// // tempResult = db.Raw(query).Find(&ListMahasiswaList)
	// tempResult := db.Raw(query)
	tempResult := db.Raw(query).Scan(&ListData)
	// fmt.Println(tempResult)

	if tempResult.Error != nil {
		result = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Error",
			Message:       tempResult.Error.Error(),
			Data:          nil,
		}
	} else {
		result = model.BaseResponseModel{
			CodeResponse:  200,
			HeaderMessage: "Success",
			Message:       "Data retrieved successfully",
			Data:          ListData,
		}
	}

	return result
}

func GetAllDataNewfolders(page, limit int) model.BaseResponseModel {
	var result model.BaseResponseModel
	var listData []NewFolder.NewFolder
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
	err := db.Table("new_folders").
		Limit(limit).
		Offset(offset).
		Order("id DESC").
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

	scanPath := "../new/" + listData.Name
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

	// Masukkan ke struct response
	response := dto.NewFolderResponse{
		ID:          listData.ID,
		Name:        listData.Name,
		Thumbnail:   listData.Thumbnail,
		IsCompleted: listData.IsCompleted,
		CreateAt:    listData.CreateAt,
		Page:        pages,
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

// =======================================================

// func moveRows(sourceTable string, targetTable string, limit int) model.BaseResponseModel {
func MoveRows(ids []int, sourceTable, targetTable string) model.BaseResponseModel {
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

			source := "../sementara/" + newRow.Name + "/"
			// destination := "../folder0/folder3/"
			destination := "../new/" + newRow.Name + "/"

			if err := copyPaste(source, destination); err != nil {
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

			// // 	// 3. Hapus data dari tabel sumber
			// var indexs []int
			// for _, row := range rows {
			// 	indexs = append(indexs, int(row.ID))
			// }

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

func IsFolderExist(db *gorm.DB, folderName string) (bool, error) {
	var count int64
	err := db.Model(&tbFolder.Folder{}).Where("name = ?", folderName).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
