package FolderControllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	// "strconv"
	"strings"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	messageStatus "web_backend/Model/MessageStatus"

	// newFolder "web_backend/Model/NewFolder"
	tbFolder "web_backend/Model/TbFolder"
	"web_backend/Repository/DuplicateRepositorys"
	"web_backend/Repository/FolderRepositorys"
)

// @Summary Cari manga (new_folders) berdasarkan nama
// @Tags manga
// @Produce json
// @Param q query string true "Keyword"
// @Param page query int false "Halaman (20 per halaman)" default(1)
// @Success 200 {object} object{page=int,per_page=int,total_data=int,data=[]object}
// @Failure 400 {object} object{error=string}
// @Security BearerAuth
// @Router /search [get]
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

// @Summary Scan SRC_DIR dan sinkronkan tabel folders
// @Description Insert folder baru, hapus row yatim, dan antrikan nama yang sudah pernah di-approve ke /duplicates.
// @Tags folders
// @Produce json
// @Success 200 {object} object{messages=[]messageStatus.Message}
// @Security BearerAuth
// @Router /update [get]
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

	// Sama seperti existingNames di atas, tapi untuk new_folders (folder
	// yang sudah di-approve) — dipakai supaya nama yang muncul lagi di
	// SRC_DIR tapi sudah pernah selesai tidak langsung di-insert ulang
	// seolah baru, melainkan diarahkan ke antrian review manual. Lihat
	// zunks/feat/patch_handle_duplication-2.md.
	existingNewFolders, err := FolderRepositorys.ExistingNewFolderNames(db)
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

		if existingNewFolderID, ok := existingNewFolders[finalFolderName]; ok {
			if err := DuplicateRepositorys.QueueCandidate(finalFolderName, folder, existingNewFolderID); err != nil {
				messages = append(messages, messageStatus.Message{
					FolderName: finalFolderName,
					Status:     "error",
					Error:      "gagal memasukkan ke antrian duplikat: " + err.Error(),
				})
			} else {
				messages = append(messages, messageStatus.Message{
					FolderName: finalFolderName,
					Status:     "duplicate_pending_review",
					Error:      "nama sudah pernah di-approve — perlu direview manual di /duplicates",
				})
			}
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

	// Let every open /status tab re-fetch its page (new rows inserted or
	// orphan rows removed above). Cheap even when nothing changed.
	FolderRepositorys.PublishFoldersChanged()

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
	})
}

// @Summary List folder di staging (folders)
// @Tags folders
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Item per halaman" default(10)
// @Success 200 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /folders [get]
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

// @Summary List manga selesai (new_folders)
// @Tags manga
// @Produce json
// @Param page query int false "Halaman (20 per halaman)" default(1)
// @Success 200 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /newFolders [get]
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

// @Summary Detail manga + daftar halaman
// @Tags manga
// @Produce json
// @Param id path int true "new_folders.id"
// @Success 200 {object} model.BaseResponseModel{Data=dto.NewFolderResponse}
// @Failure 404 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /id/{id} [get]
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

// @Summary Rename manga
// @Tags manga
// @Accept json
// @Produce json
// @Param id path int true "new_folders.id"
// @Param body body dto.RenameNewFolderRequest true "Nama baru"
// @Success 200 {object} model.BaseResponseModel
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /id/{id} [patch]
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

// @Summary Hapus manga
// @Tags manga
// @Accept json
// @Produce json
// @Param id path int true "new_folders.id"
// @Param body body dto.DeleteNewFolderRequest true "apply_to_disk=true juga menghapus folder di DST_DIR"
// @Success 200 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /id/{id} [delete]
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

// @Summary Perbaiki thumbnail satu manga
// @Description Hitung ulang URL thumbnail dari isi folder di disk (halaman pertama, urutan sama dengan reader) dan simpan kalau berbeda dari yang di DB.
// @Description status: ok (sudah benar) | fixed (diperbarui). 409 kalau folder tidak ada di disk (folder_missing) atau tanpa gambar (no_image); DB tidak diubah.
// @Tags manga
// @Produce json
// @Param id path int true "new_folders.id"
// @Success 200 {object} model.BaseResponseModel{Data=object{status=string,old=string,new=string}}
// @Failure 404 {object} model.BaseResponseModel
// @Failure 409 {object} model.BaseResponseModel{Data=object{status=string,old=string}}
// @Security BearerAuth
// @Router /id/{id}/thumbnail [post]
func RepairThumbnail(c *gin.Context) {
	response := FolderRepositorys.RepairNewFolderThumbnail(c.Param("id"))
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

// @Summary Antrikan folder staging untuk dipindah ke DST_DIR
// @Description Folder yang sedang antri/diproses (oleh siapa pun) dilewati dan dikembalikan di `skipped`.
// @Description Status per folder dipantau lewat SSE global GET /folders/events.
// @Tags folders
// @Accept json
// @Produce json
// @Param body body dto.InputDataReq true "ID folders yang dipindah"
// @Success 200 {object} model.BaseResponseModel{Data=object{claimed=[]int,skipped=[]FolderRepositorys.SkippedFolder}}
// @Failure 400 {object} model.BaseResponseModel
// @Failure 409 {object} model.BaseResponseModel{Data=object{claimed=[]int,skipped=[]FolderRepositorys.SkippedFolder}} "Semua id sudah diproses/tidak ditemukan"
// @Security BearerAuth
// @Router /folders [post]
func MoveRowAndTrack(c *gin.Context) {
	enqueueFolders(c, "move")
}

// DeleteRowsAndTrack queues selected "folders" rows (and their SRC_DIR
// directory) for deletion — the delete counterpart to MoveRowAndTrack,
// sharing the same claim, queue and SSE stream.
// @Summary Antrikan folder staging untuk dihapus
// @Tags folders
// @Accept json
// @Produce json
// @Param body body dto.InputDataReq true "ID folders yang dihapus"
// @Success 200 {object} model.BaseResponseModel{Data=object{claimed=[]int,skipped=[]FolderRepositorys.SkippedFolder}}
// @Failure 400 {object} model.BaseResponseModel
// @Failure 409 {object} model.BaseResponseModel{Data=object{claimed=[]int,skipped=[]FolderRepositorys.SkippedFolder}}
// @Security BearerAuth
// @Router /folders/delete [post]
func DeleteRowsAndTrack(c *gin.Context) {
	enqueueFolders(c, "delete")
}

func enqueueFolders(c *gin.Context, op string) {
	var request dto.InputDataReq
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: err.Error(), Data: nil})
		return
	}
	if len(request.IDS) == 0 {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "no ids provided", Data: nil})
		return
	}

	claimed, skipped, err := FolderRepositorys.EnqueueFolders(request.IDS, op)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil})
		return
	}

	data := gin.H{"claimed": claimed, "skipped": skipped}
	if len(claimed) == 0 {
		c.JSON(http.StatusConflict, model.BaseResponseModel{CodeResponse: 409, HeaderMessage: "Conflict", Message: "tidak ada folder yang bisa diproses", Data: data})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Accepted", Message: fmt.Sprintf("%d folder masuk antrian", len(claimed)), Data: data})
}

// FolderEvents is the global SSE stream behind /status: every connection
// first gets a `snapshot` of all queued/processing folders, then live
// `item` / `progress` / `changed` events. A reconnect (EventSource retries
// on its own) just gets a fresh snapshot, so there is nothing to resume.
// @Summary Stream status folder (SSE global)
// @Description `snapshot` {items:[{id,name,op,status,percent}]} dikirim pertama di setiap koneksi.
// @Description `item` {id,name,op,status: queued|processing|moved|deleted|failed, error?}.
// @Description `progress` {id,percent}. `changed` {} = list berubah (mis. setelah /update), ambil ulang.
// @Description Komentar `: ping` tiap 20 detik menjaga koneksi tetap hidup.
// @Tags folders
// @Produce text/event-stream
// @Success 200 {string} string "event stream"
// @Security BearerAuth
// @Router /folders/events [get]
func FolderEvents(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	snapshot, events := FolderRepositorys.SubscribeFolderEvents()
	defer FolderRepositorys.UnsubscribeFolderEvents(events)

	send := func(event string, data any) error {
		payload, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	fmt.Fprint(c.Writer, "retry: 3000\n\n")
	if send("snapshot", gin.H{"items": snapshot}) != nil {
		return
	}

	// Cloudflare Tunnel drops HTTP connections idle for ~100s.
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ping.C:
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case evt, ok := <-events:
			if !ok {
				return // fell behind and was dropped; the browser reconnects and resyncs
			}
			if send(evt.Event, evt.Data) != nil {
				return
			}
		}
	}
}

// @Summary Folder staging yang belum selesai
// @Tags folders
// @Produce json
// @Success 200 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /filteredDatas [get]
func GetFilteredData(c *gin.Context) {
	response := FolderRepositorys.FilteredData("folders", "new_folder")
	if response.CodeResponse != 200 {
		c.JSON(response.CodeResponse, response)
		return
	}
	c.JSON(http.StatusOK, response)
}
