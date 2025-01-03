package MessageStatus

type Message struct {
	FolderName string `json:"folder_name"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}
