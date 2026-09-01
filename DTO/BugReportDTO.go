package dto

type CreateBugReportRequest struct {
	FolderID    uint   `json:"folder_id" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type UpdateBugReportStatusRequest struct {
	Status string `json:"status" binding:"required"` // "open" | "fixed"
}

// BugReportItem is the list-page shape: bug_reports joined with new_folders
// for the manga name/thumbnail, same pattern as PendingTranslationItem.
type BugReportItem struct {
	ID          uint   `gorm:"column:id" json:"id"`
	FolderID    uint   `gorm:"column:folder_id" json:"folder_id"`
	FolderName  string `gorm:"column:folder_name" json:"folder_name"`
	Thumbnail   string `gorm:"column:thumbnail" json:"thumbnail"`
	Description string `gorm:"column:description" json:"description"`
	Status      string `gorm:"column:status" json:"status"`
	CreatedAt   string `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   string `gorm:"column:updated_at" json:"updated_at"`
}
