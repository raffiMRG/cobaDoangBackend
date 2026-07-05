package dto

type RenameNewFolderRequest struct {
	NewName     string `json:"new_name" binding:"required"`
	ApplyToDisk bool   `json:"apply_to_disk"`
}

type DeleteNewFolderRequest struct {
	ApplyToDisk bool `json:"apply_to_disk"`
}
