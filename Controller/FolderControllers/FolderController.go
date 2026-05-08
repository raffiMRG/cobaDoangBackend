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
	// tbFolder "web_backend/Model/TbFolder"
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
	// Process folders and insert into the database
	var messages []messageStatus.Message
	for _, folder := range folders {
		unixStylePath := strings.ReplaceAll(folder, "\\", "/")
		finalFolderName := filepath.Base(unixStylePath)

		// Cek apakah folder sudah ada di database
		exists, err := FolderRepositorys.IsFolderExist(db, finalFolderName)
		if err != nil {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "error",
				Error:      "check failed: " + err.Error(),
			})
			continue
		}

		if exists {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "skipped",
				Error:      "folder already exists",
			})
			continue
		}

		// Insert into database and collect status
		err = FolderRepositorys.InsertFolder(db, finalFolderName, folder)
		if err != nil {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "error",
				Error:      err.Error(),
			})
		} else {
			messages = append(messages, messageStatus.Message{
				FolderName: finalFolderName,
				Status:     "success",
			})
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

	// === 3️⃣ Loop kirim progres ke client ===
	for {
		val, ok := FolderRepositorys.ProgressMap.Load(taskID)
		if ok {
			progress := val.(float64)

			// gunakan format event standar SSE
			fmt.Fprintf(c.Writer, "event: progress\ndata: %.2f\n\n", progress)
			flusher.Flush()

			if progress >= 100 {
				fmt.Fprintf(c.Writer, "event: done\ndata: Completed\n\n")
				flusher.Flush()
				FolderRepositorys.ProgressMap.Delete(taskID)
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	// === 4️⃣ Pastikan koneksi ditutup dengan rapi ===
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
