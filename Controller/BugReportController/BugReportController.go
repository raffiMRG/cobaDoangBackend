package BugReportController

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	"web_backend/Repository/BugReportRepositorys"
)

func CreateBugReport(c *gin.Context) {
	var request dto.CreateBugReportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, model.BaseResponseModel{
			CodeResponse:  400,
			HeaderMessage: "Bad Request",
			Message:       err.Error(),
			Data:          nil,
		})
		return
	}

	response := BugReportRepositorys.CreateBugReport(request.FolderID, request.Description)
	c.JSON(response.CodeResponse, response)
}

func ListBugReports(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	sort := c.DefaultQuery("sort", "newest")

	items, err := BugReportRepositorys.ListBugReports(status, sort)
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

func UpdateBugReportStatus(c *gin.Context) {
	var request dto.UpdateBugReportStatusRequest
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
	response := BugReportRepositorys.UpdateBugReportStatus(id, request.Status)
	c.JSON(response.CodeResponse, response)
}
