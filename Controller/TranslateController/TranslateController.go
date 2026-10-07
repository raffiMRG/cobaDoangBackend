package TranslateController

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	"web_backend/Repository/TranslateRepositorys"
)

// @Summary Masukkan manga ke antrian translate
// @Tags translate
// @Produce json
// @Param id path int true "new_folders.id"
// @Success 200 {object} model.BaseResponseModel{Data=object{status=string}}
// @Failure 404 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /translate/{id}/request [post]
func RequestTranslation(c *gin.Context) {
	id := c.Param("id")
	response := TranslateRepositorys.RequestTranslation(id)
	c.JSON(response.CodeResponse, response)
}

// @Summary Keluarkan manga dari antrian translate
// @Tags translate
// @Produce json
// @Param id path int true "new_folders.id"
// @Success 200 {object} model.BaseResponseModel
// @Failure 404 {object} model.BaseResponseModel
// @Failure 409 {object} model.BaseResponseModel "Sedang diproses"
// @Security BearerAuth
// @Router /translate/{id} [delete]
func CancelTranslation(c *gin.Context) {
	id := c.Param("id")
	response := TranslateRepositorys.CancelTranslation(id)
	c.JSON(response.CodeResponse, response)
}

// @Summary List antrian translate (pending/processing/failed)
// @Tags translate
// @Produce json
// @Success 200 {object} model.BaseResponseModel{Data=[]dto.PendingTranslationItem}
// @Security BearerAuth
// @Router /translate/pending [get]
func ListPending(c *gin.Context) {
	items, err := TranslateRepositorys.ListPendingTranslations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{
			CodeResponse:  500,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}
	c.JSON(http.StatusOK, model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "ok",
		Data:          items,
	})
}

// @Summary Update status translate (dipakai worker)
// @Tags translate
// @Accept json
// @Produce json
// @Param id path int true "new_folders.id"
// @Param body body dto.UpdateTranslationStatusRequest true "status: pending|processing|completed|failed"
// @Success 200 {object} model.BaseResponseModel
// @Failure 400 {object} model.BaseResponseModel
// @Failure 404 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /translate/{id}/status [patch]
func UpdateStatus(c *gin.Context) {
	var request dto.UpdateTranslationStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	id := c.Param("id")
	response := TranslateRepositorys.UpdateTranslationStatus(id, request.Status, request.ErrorMessage)
	c.JSON(response.CodeResponse, response)
}

// @Summary Upload hasil translate (dipakai worker)
// @Description Simpan halaman hasil translate sebagai folder baru di DST_DIR dan tandai request completed.
// @Tags translate
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "new_folders.id asal"
// @Param new_folder_name formData string true "Nama folder hasil"
// @Param files formData []file true "Halaman hasil translate" collectionFormat(multi)
// @Success 200 {object} model.BaseResponseModel{Data=object{new_folder_id=int,name=string,files_written=int,files_skipped=int}}
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /translate/{id}/complete [post]
func CompleteTranslation(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	newFolderName := c.PostForm("new_folder_name")
	if newFolderName == "" {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       "new_folder_name is required",
			Data:          nil,
		})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       "no files uploaded",
			Data:          nil,
		})
		return
	}

	id := c.Param("id")
	response := TranslateRepositorys.CompleteTranslation(id, newFolderName, files)
	c.JSON(response.CodeResponse, response)
}

// @Summary Ambil setting translator (tanpa API key)
// @Tags translate
// @Produce json
// @Success 200 {object} model.BaseResponseModel{Data=dto.TranslatorSettingsResponse}
// @Security BearerAuth
// @Router /translate/settings [get]
func GetSettings(c *gin.Context) {
	response := TranslateRepositorys.GetSettings()
	c.JSON(response.CodeResponse, response)
}

// @Summary Ambil setting translator untuk worker (termasuk API key)
// @Tags translate
// @Produce json
// @Success 200 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /translate/settings/worker [get]
func GetWorkerSettings(c *gin.Context) {
	response := TranslateRepositorys.GetWorkerSettings()
	c.JSON(response.CodeResponse, response)
}

// @Summary Simpan setting translator
// @Tags translate
// @Accept json
// @Produce json
// @Param body body dto.TranslatorSettingsRequest true "Setting"
// @Success 200 {object} model.BaseResponseModel
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /translate/settings [put]
func SaveSettings(c *gin.Context) {
	var request dto.TranslatorSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	response := TranslateRepositorys.SaveSettings(request)
	c.JSON(response.CodeResponse, response)
}

// @Summary Reset setting translator ke default
// @Tags translate
// @Produce json
// @Success 200 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /translate/settings [delete]
func ResetSettings(c *gin.Context) {
	response := TranslateRepositorys.ResetSettings()
	c.JSON(response.CodeResponse, response)
}
