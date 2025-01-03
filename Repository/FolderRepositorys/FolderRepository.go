package FolderRepositorys

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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

func InsertFolder(db *gorm.DB, folderName string) error {
	folder := tbFolder.Folder{Name: folderName}
	return db.Create(&folder).Error
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

// ============================================

func GetDataFromId(table, id string) model.BaseResponseModel {
	// var query string
	var result model.BaseResponseModel
	var listData []tbFolder.Folder
	// db := connection.DB
	tempResult := connection.DB

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

func GetRowFromId(table, id string) ([]tbFolder.Folder, error) {
	var query string
	var listData []tbFolder.Folder
	var errorMessage error
	db := connection.DB
	tempResult := connection.DB

	if id != "" {
		intId, err := strconv.Atoi(id)
		if err != nil {
			errorMessage = errors.New("invalid paremeter input for 'id'")
			return nil, errorMessage
		}

		tempResult = db.Where("id = ?", intId).Find(&listData)
		fmt.Println("id query " + id)
	} else {
		query = fmt.Sprintf("SELECT * FROM %s", table)
		tempResult = db.Raw(query).Find(&listData)
		fmt.Println("tampilkan semua data dari tabel")
	}

	if tempResult.Error != nil {
		return nil, tempResult.Error
	}

	return listData, nil
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
			// 	// 1. Ambil data dari tabel sumber
			strId := strconv.Itoa(id)
			rows, err := GetRowFromId(sourceTable, strId)

			if err != nil {
				return err
			}

			if len(rows) == 0 {
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

			// Buat slice baru hanya dengan field Name
			// newRows := make([]tbFolder.Folder, len(rows))
			newRows := make([]NewFolder.NewFolder, len(rows))
			for i, row := range rows {
				// newRows[i] = tbFolder.Folder{
				// 	Name: row.Name, // Hanya ambil Name
				// }
				newRows[i] = NewFolder.NewFolder{
					Name:        row.Name, // Hanya ambil Name
					IsCompleted: false,    // Isi nilai devault true
				}
			}

			// 	// 2. Masukkan data ke tabel tujuan
			if err := tx.Table(targetTable).Create(&newRows).Error; err != nil {
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
				defaultData := dto.InputDataRes{
					// ID:     id,
					Title:  "Success",
					Status: true,
				}
				// COPY DATA
				for _, data := range newRows {

					source := "../folder0/folder1/" + data.Name + "/"
					// destination := "../folder0/folder3/"
					destination := "../folder0/folder3/" + data.Name + "/"

					if err := copyPaste(source, destination); err != nil {
						fmt.Println("Error:", err)
					} else {
						fmt.Println("Semua file dan folder berhasil disalin.")
					}
					fmt.Println(destination)
				}

				response = append(response, defaultData)
				fmt.Println("masuk response success")
			}

			// 	// 3. Hapus data dari tabel sumber
			// 	var indexs []int
			// 	for _, row := range rows {
			// 		indexs = append(indexs, int(row.ID))
			// 	}

			// 	// listDataSuccess = ids

			// 	if err := tx.Table(sourceTable).Where("id IN ?", indexs).Delete(nil).Error; err != nil {
			// 		return fmt.Errorf("failed to delete rows from source table: %w", err)
			// 	} else {
			// 		defaultData := dto.InputDataRes{
			// 			ID:     id,
			// 			Title:  "Success",
			// 			Status: true,
			// 		}
			// 		response = append(response, defaultData)
			// 		fmt.Println("masuk response success")
			// 	}
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

func FailedResponse(message error) model.BaseResponseModel {
	result := model.BaseResponseModel{
		CodeResponse:  400,
		HeaderMessage: "Error",
		Message:       message.Error(),
		Data:          nil,
	}
	return result
}
