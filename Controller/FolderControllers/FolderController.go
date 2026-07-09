package FolderControllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// "strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	messageStatus "web_backend/Model/MessageStatus"

	// newFolder "web_backend/Model/NewFolder"
	tbFolder "web_backend/Model/TbFolder"
	"web_backend/Repository/FolderRepositorys"
)

func SearchFolders(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Keyword is required"})
		return
	}

	// Ambil parameter page (default: 1)
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	results, total, err := FolderRepositorys.SearchFolders(keyword, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"page":       page,
		"per_page":   20,
		"total_data": total,
		"data":       results,
	})
}

func UpdateAndInsert(c *gin.Context) {

	root := os.Getenv("SRC_DIR") // Change to your desired root directory

	folders, err := FolderRepositorys.ScanFolders(root)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	db := connection.DB

	// Satu query untuk semua nama yang sudah ada, bukan satu query
	// exists-check per folder — ini yang bikin lambat begitu SRC_DIR
	// punya banyak folder pending.
	existingNames, err := FolderRepositorys.ExistingFolderNames(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var messages []messageStatus.Message
	var toInsert []tbFolder.Folder
	var toInsertNames []string
	scannedNames := make(map[string]bool, len(folders))

	for _, folder := range folders {
		unixStylePath := strings.ReplaceAll(folder, "\\", "/")
		finalFolderName := filepath.Base(unixStylePath)
		scannedNames[finalFolderName] = true

		if existingNames[finalFolderName] {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "skipped",
				Error:      "folder already exists",
			})
			continue
		}

		thumbnail, err := FolderRepositorys.BuildThumbnailURL(finalFolderName, folder)
		if err != nil {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "error",
				Error:      err.Error(),
			})
			continue
		}

		toInsert = append(toInsert, tbFolder.Folder{Name: finalFolderName, Thumbnail: thumbnail})
		toInsertNames = append(toInsertNames, finalFolderName)
	}

	// Row folders yang namanya sudah tercatat di DB tapi tidak lagi muncul
	// di hasil scan SRC_DIR sekarang — foldernya sudah dihapus/dipindah
	// manual di luar aplikasi, dan tanpa ini thumbnail-nya nyangkut 404
	// selamanya (lihat zunks/why 404.md, kasus pertama). Guard di
	// len(scannedNames) == 0 supaya SRC_DIR yang gagal ke-mount / kosong
	// sesaat tidak disalahartikan sebagai "semua folder hilang" dan
	// nge-wipe seluruh tabel.
	if len(scannedNames) > 0 {
		var orphanNames []string
		for name := range existingNames {
			if !scannedNames[name] {
				orphanNames = append(orphanNames, name)
			}
		}

		if len(orphanNames) > 0 {
			if err := db.Where("name IN ?", orphanNames).Delete(&tbFolder.Folder{}).Error; err != nil {
				for _, name := range orphanNames {
					messages = append(messages, messageStatus.Message{
						FolderName: name,
						Status:     "error",
						Error:      "gagal hapus row yatim: " + err.Error(),
					})
				}
			} else {
				for _, name := range orphanNames {
					messages = append(messages, messageStatus.Message{
						FolderName: name,
						Status:     "deleted",
						Error:      "folder tidak ditemukan lagi di SRC_DIR",
					})
				}
			}
		}
	}

	// Satu batch insert untuk semua folder baru, bukan satu INSERT per folder.
	if len(toInsert) > 0 {
		if err := db.CreateInBatches(&toInsert, 200).Error; err != nil {
			for _, name := range toInsertNames {
				messages = append(messages, messageStatus.Message{
					FolderName: name,
					Status:     "error",
					Error:      "batch insert failed: " + err.Error(),
				})
			}
		} else {
			for _, name := range toInsertNames {
				messages = append(messages, messageStatus.Message{
					FolderName: name,
					Status:     "success",
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}

func DisplayAllDataFolder(c *gin.Context) {
	// var response model.BaseResponseModel
	// Baca query params: ?page=2&limit=20
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	response := FolderRepositorys.GetAllData("folders", page, limit)
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

func DisplayDataNewfolder(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	// limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// var response model.BaseResponseModel

	response := FolderRepositorys.GetAllDataNewfolders(page, 20)
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}

	// use the parse data
	// fmt.Printf("Recived data : %+v\n", )
	c.JSON(http.StatusOK, response)
}

func GetDataById(c *gin.Context) {
	var response model.BaseResponseModel

	strId := c.Param("id")
	fmt.Println("strId:", strId)

	response = FolderRepositorys.GetNewfolderDataFromId(strId)
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	c.JSON(http.StatusOK, response)
}

func RenameNewFolder(c *gin.Context) {
	var request dto.RenameNewFolderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	strId := c.Param("id")
	response := FolderRepositorys.RenameNewFolder(strId, request.NewName, request.ApplyToDisk)
	c.JSON(response.CodeResponse, response)
}

func DeleteNewFolder(c *gin.Context) {
	var request dto.DeleteNewFolderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	strId := c.Param("id")
	response := FolderRepositorys.DeleteNewFolder(strId, request.ApplyToDisk)
	c.JSON(response.CodeResponse, response)
}

func MoveRow(c *gin.Context) {
	var request dto.InputDataReq
	var response model.BaseResponseModel

	if err := c.ShouldBindJSON(&request); err != nil {
		response = model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		}
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response = FolderRepositorys.MoveRows(request.IDS, "folders", "new_folders")
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	c.JSON(http.StatusOK, response)
}

func MoveRowAndTrack(c *gin.Context) {
	var request dto.InputDataReq
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	// buat ID unik untuk tracking progress
	taskID := uuid.New().String()

	// Jalankan proses pemindahan di background
	go FolderRepositorys.MoveRowsWithProgress(taskID, request.IDS, "folders", "new_folders")

	// kirim response taskID ke frontend
	c.JSON(http.StatusOK, model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Accepted",
		Message:       "Proses pemindahan dimulai",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}

func FolderProgress(c *gin.Context) {
	taskID := c.Param("taskID")

	// === 1️⃣ Set header SSE lengkap ===
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*") // penting kalau beda domain/port

	// === 2️⃣ Pastikan flusher tersedia ===
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// === 3️⃣ Ambil channel progress untuk taskID ini ===
	// MoveRowsWithProgress mendaftarkan channel-nya di goroutine terpisah,
	// jadi ada kemungkinan kecil browser connect sebelum itu sempat jalan —
	// retry singkat (maks ~500ms) alih-alih langsung 404.
	var progressChan chan float64
	found := false
	for i := 0; i < 50; i++ {
		if val, ok := FolderRepositorys.ProgressChannels.Load(taskID); ok {
			progressChan = val.(chan float64)
			found = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !found {
		http.Error(c.Writer, "Task not found", http.StatusNotFound)
		return
	}

	// === 4️⃣ Kirim setiap update PERSIS saat diterima — push, bukan polling,
	// jadi progress muncul real-time apapun kecepatan penyalinannya ===
	for progress := range progressChan {
		fmt.Fprintf(c.Writer, "event: progress\ndata: %.2f\n\n", progress)
		flusher.Flush()
	}

	// Channel ditutup oleh MoveRowsWithProgress setelah selesai
	FolderRepositorys.ProgressChannels.Delete(taskID)
	fmt.Fprintf(c.Writer, "event: done\ndata: Completed\n\n")
	flusher.Flush()

	// === 5️⃣ Pastikan koneksi ditutup dengan rapi ===
	c.Writer.Write([]byte("event: close\ndata: Connection closed\n\n"))
	flusher.Flush()
}
func GetFilteredData(c *gin.Context) {
	response := FolderRepositorys.FilteredData("folders", "new_folder")
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	c.JSON(http.StatusOK, response)
}
