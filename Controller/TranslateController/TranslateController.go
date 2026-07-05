package TranslateController

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	"web_backend/Repository/TranslateRepositorys"
)

func RequestTranslation(c *gin.Context) {
	id := c.Param("id")
	response := TranslateRepositorys.RequestTranslation(id)
	c.JSON(response.CodeResponse, response)
}

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
