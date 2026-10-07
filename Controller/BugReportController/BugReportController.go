package BugReportController

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "web_backend/DTO"
	model "web_backend/Model"
	"web_backend/Repository/BugReportRepositorys"
)

// @Summary Laporkan bug pada manga
// @Tags bug-reports
// @Accept json
// @Produce json
// @Param body body dto.CreateBugReportRequest true "Laporan"
// @Success 200 {object} model.BaseResponseModel
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /bug-reports [post]
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

// @Summary List bug report
// @Tags bug-reports
// @Produce json
// @Param status query string false "Filter status" Enums(all, open, fixed) default(all)
// @Param sort query string false "Urutan" Enums(newest, oldest) default(newest)
// @Success 200 {object} model.BaseResponseModel{Data=[]dto.BugReportItem}
// @Security BearerAuth
// @Router /bug-reports [get]
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

// @Summary Ubah status bug report
// @Tags bug-reports
// @Accept json
// @Produce json
// @Param id path int true "bug_reports.id"
// @Param body body dto.UpdateBugReportStatusRequest true "status: open|fixed"
// @Success 200 {object} model.BaseResponseModel
// @Failure 400 {object} model.BaseResponseModel
// @Security BearerAuth
// @Router /bug-reports/{id}/status [patch]
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
