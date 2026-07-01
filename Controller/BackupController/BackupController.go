package BackupController

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	"web_backend/Repository/BackupRepositorys"
)

func Export(c *gin.Context) {
	mode := c.DefaultQuery("mode", "full")
	if mode != "full" && mode != "data" {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       "mode must be 'full' or 'data'",
			Data:          nil,
		})
		return
	}

	sqlContent, err := BackupRepositorys.ExportDatabase(connection.DB, mode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.BaseResponseModel{
			CodeResponse:  500,
			HeaderMessage: "Error",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	filename := fmt.Sprintf("manga-backup-%s-%s.sql", mode, time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/sql", []byte(sqlContent))
}

func Import(c *gin.Context) {
	var content []byte

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		// No multipart file provided — fall back to raw request body.
		content, err = io.ReadAll(c.Request.Body)
		if err != nil || len(content) == 0 {
			c.JSON(http.StatusBadRequest, model.BaseResponseModel{
				CodeResponse:  400,
				HeaderMessage: "Bad Request",
				Message:       "no file uploaded and no request body",
				Data:          nil,
			})
			return
		}
	} else {
		defer file.Close()
		content, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.BaseResponseModel{
				CodeResponse:  400,
				HeaderMessage: "Bad Request",
				Message:       err.Error(),
				Data:          nil,
			})
			return
		}
	}

	if err := BackupRepositorys.ImportDatabase(connection.DB, string(content)); err != nil {
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
		Message:       "database imported successfully",
		Data:          nil,
	})
}
