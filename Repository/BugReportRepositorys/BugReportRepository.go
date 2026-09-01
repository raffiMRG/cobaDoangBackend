package BugReportRepositorys

import (
	"strconv"

	dto "web_backend/DTO"
	model "web_backend/Model"
	BugReport "web_backend/Model/BugReport"
	connection "web_backend/Model/Connection"
	"web_backend/Model/NewFolder"
)

var validStatuses = map[string]bool{
	"open":  true,
	"fixed": true,
}

// CreateBugReport records a report tied to an existing manga folder. Unlike
// translation requests, a manga can have any number of open bug reports at
// once (e.g. different broken pages reported separately), so this always
// inserts a new row rather than upserting an existing one.
func CreateBugReport(folderID uint, description string) model.BaseResponseModel {
	var folder NewFolder.NewFolder
	if err := connection.DB.Where("id = ?", folderID).First(&folder).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: "manga not found", Data: nil}
	}

	report := BugReport.BugReport{FolderID: folderID, Description: description, Status: "open"}
	if err := connection.DB.Create(&report).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	return model.BaseResponseModel{
		CodeResponse:  200,
		HeaderMessage: "Success",
		Message:       "bug report submitted",
		Data:          map[string]interface{}{"id": report.ID, "status": report.Status},
	}
}

// ListBugReports backs the /bug-reports admin page: every report, optionally
// filtered by status and sorted by report date.
func ListBugReports(status string, sort string) ([]dto.BugReportItem, error) {
	var items []dto.BugReportItem

	query := connection.DB.Table("bug_reports br").
		Select("br.id, br.folder_id, nf.name AS folder_name, nf.thumbnail, br.description, br.status, br.created_at, br.updated_at").
		Joins("JOIN new_folders nf ON nf.id = br.folder_id")

	if status != "" && status != "all" {
		query = query.Where("br.status = ?", status)
	}

	if sort == "oldest" {
		query = query.Order("br.created_at ASC")
	} else {
		query = query.Order("br.created_at DESC")
	}

	err := query.Scan(&items).Error
	return items, err
}

// UpdateBugReportStatus is the manual "mark fixed" / "reopen" toggle on the
// admin page — there's no automatic status transition for bug reports.
func UpdateBugReportStatus(idStr string, status string) model.BaseResponseModel {
	if !validStatuses[status] {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid status: " + status, Data: nil}
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: "invalid id", Data: nil}
	}

	result := connection.DB.Model(&BugReport.BugReport{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: result.Error.Error(), Data: nil}
	}
	if result.RowsAffected == 0 {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: "bug report not found", Data: nil}
	}

	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "status updated", Data: nil}
}
