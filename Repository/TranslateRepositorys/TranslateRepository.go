package TranslateRepositorys

import (
	"errors"
	"mime/multipart"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"

	dto "web_backend/DTO"
	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	"web_backend/Model/NewFolder"
	Translation "web_backend/Model/Translation"
	"web_backend/Repository/FolderRepositorys"
	"web_backend/Repository/UploadRepositorys"
)

var validStatuses = map[string]bool{
	"pending":    true,
	"processing": true,
	"completed":  true,
	"failed":     true,
}

// RequestTranslation enqueues folderId for translation. A second request
// while already pending is a no-op (returns the current status unchanged);
// a request while failed/processing/completed resets it to pending — the
// failed/processing case doubles as manual recovery if a worker ever
// crashes mid-job (no automatic timeout/retry yet), and completed is
// allowed to be re-queued on purpose so a manga can be re-translated
// (e.g. after a bad run) without any separate "retranslate" action.
func RequestTranslation(folderIdStr string) model.BaseResponseModel {
	db := connection.DB

	folderId, err := strconv.Atoi(folderIdStr)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid id", Data: nil}
	}

	var folder NewFolder.NewFolder
	if err := db.Where("id = ?", folderId).First(&folder).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: "manga not found", Data: nil}
	}

	var existing Translation.Translation
	err = db.Where("folder_id = ?", folderId).First(&existing).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		newRow := Translation.Translation{FolderID: uint(folderId), Status: "pending"}
		if err := db.Create(&newRow).Error; err != nil {
			return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
		}
		return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "translation requested", Data: map[string]string{"status": "pending"}}

	case err != nil:
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}

	case existing.Status == "pending":
		return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "already " + existing.Status, Data: map[string]string{"status": existing.Status}}

	default: // completed, failed, processing
		if err := db.Model(&existing).Updates(map[string]interface{}{"status": "pending", "error_message": ""}).Error; err != nil {
			return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
		}
		return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "translation requested", Data: map[string]string{"status": "pending"}}
	}
}

// CancelTranslation removes a translation request from the queue (used by
// the "keluarkan dari antrian" button on the /translate batch page, which
// only ever lists pending items). Refuses to touch a row the worker is
// actively processing, since the daemon would otherwise keep calling
// /translate/:id/status or /translate/:id/complete against a row that no
// longer exists partway through a job.
func CancelTranslation(folderIdStr string) model.BaseResponseModel {
	folderId, err := strconv.Atoi(folderIdStr)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid id", Data: nil}
	}

	var existing Translation.Translation
	if err := connection.DB.Where("folder_id = ?", folderId).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: "no translation request found", Data: nil}
		}
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	if existing.Status == "processing" {
		return model.BaseResponseModel{CodeResponse: 409, HeaderMessage: "Error", Message: "cannot cancel: translation is currently processing", Data: nil}
	}

	if err := connection.DB.Delete(&existing).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "translation request removed", Data: nil}
}

// ListPendingTranslations backs the /translate batch page — every manga
// currently queued (not yet picked up by a worker).
func ListPendingTranslations() ([]dto.PendingTranslationItem, error) {
	var items []dto.PendingTranslationItem
	err := connection.DB.Table("translations t").
		Select("t.folder_id, nf.name, nf.thumbnail").
		Joins("JOIN new_folders nf ON nf.id = t.folder_id").
		Where("t.status = ?", "pending").
		Order("t.created_at ASC").
		Scan(&items).Error
	return items, err
}

// UpdateTranslationStatus is called by the translate worker to mark a job
// as processing/failed (and as a fallback path for completed, though the
// happy path goes through CompleteTranslation instead).
func UpdateTranslationStatus(folderIdStr, status, errorMessage string) model.BaseResponseModel {
	if !validStatuses[status] {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid status: " + status, Data: nil}
	}

	folderId, err := strconv.Atoi(folderIdStr)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid id", Data: nil}
	}

	result := connection.DB.Model(&Translation.Translation{}).
		Where("folder_id = ?", folderId).
		Updates(map[string]interface{}{"status": status, "error_message": errorMessage})
	if result.Error != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: result.Error.Error(), Data: nil}
	}
	if result.RowsAffected == 0 {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: "no translation request found for this manga", Data: nil}
	}

	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "status updated", Data: nil}
}

// CompleteTranslation is the worker's single "I'm done" call: it writes the
// translated pages into DST_DIR as a brand-new folder, creates a matching
// new_folders row (exactly like a manga that just got approved via
// /status), and marks the original folder's translation request completed.
func CompleteTranslation(folderIdStr, newFolderName string, files []*multipart.FileHeader) model.BaseResponseModel {
	folderId, err := strconv.Atoi(folderIdStr)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid id", Data: nil}
	}

	dstDir := os.Getenv("DST_DIR")
	written, skipped, err := UploadRepositorys.SaveFolderFiles(dstDir, newFolderName, files)
	if err != nil {
		if errors.Is(err, UploadRepositorys.ErrInvalidName) {
			return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: err.Error(), Data: nil}
		}
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	safeName, _ := UploadRepositorys.SanitizeName(newFolderName) // already proven valid by SaveFolderFiles succeeding
	thumbnail, thumbErr := FolderRepositorys.BuildNewFolderThumbnailURL(safeName, dstDir+"/"+safeName)
	if thumbErr != nil {
		thumbnail = ""
	}

	newRow := NewFolder.NewFolder{Name: safeName, Thumbnail: thumbnail, IsCompleted: false, CreateAt: time.Now()}
	if err := connection.DB.Create(&newRow).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	if err := connection.DB.Model(&Translation.Translation{}).Where("folder_id = ?", folderId).
		Updates(map[string]interface{}{"status": "completed", "error_message": ""}).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	return model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "translation completed",
		Data: map[string]interface{}{
			"new_folder_id": newRow.ID,
			"name":          safeName,
			"files_written": written,
			"files_skipped": skipped,
		},
	}
}
