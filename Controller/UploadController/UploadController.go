package UploadController

import (
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	model "web_backend/Model"
	"web_backend/Repository/UploadRepositorys"
)

// @Summary Upload folder manga ke SRC_DIR
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param folder_name formData string true "Nama folder"
// @Param files formData []file true "Halaman-halaman (boleh banyak)" collectionFormat(multi)
// @Success 200 {object} model.BaseResponseModel{Data=object{folder_name=string,files_written=int,files_skipped=int}}
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /upload/folder [post]
func UploadFolder(c *gin.Context) {
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

	folderName := c.PostForm("folder_name")
	if folderName == "" {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       "folder_name is required",
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

	written, skipped, err := UploadRepositorys.SaveFolderFiles(os.Getenv("SRC_DIR"), folderName, files)
	if err != nil {
		if errors.Is(err, UploadRepositorys.ErrInvalidName) {
			c.JSON(http.StatusBadRequest, model.BaseResponseModel{
				CodeResponse:  400,
				HeaderMessage: "Bad Request",
				Message:       err.Error(),
				Data:          nil,
			})
			return
		}
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
		Message:       "folder uploaded successfully",
		Data: gin.H{
			"folder_name":   folderName,
			"files_written": written,
			"files_skipped": skipped,
		},
	})
}
