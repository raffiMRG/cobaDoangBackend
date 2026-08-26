package DuplicateCandidate

import (
	"time"
	"web_backend/Model/NewFolder"
)

// DuplicateCandidate holds a folder found in SRC_DIR whose name already
// matches an approved new_folders row, until a human picks "buat judul
// baru" or "merge" on the /duplicates review page. See
// zunks/feat/patch_handle_duplication-2.md for the full design.
type DuplicateCandidate struct {
	ID                   uint                `json:"id" gorm:"primaryKey"`
	Name                 string              `json:"name"`
	ExistingNewFolderID  uint                `json:"existing_new_folder_id"`
	ExistingNewFolder    NewFolder.NewFolder `json:"existing_new_folder" gorm:"foreignKey:ExistingNewFolderID;constraint:OnDelete:CASCADE"`
	IncomingPath         string              `json:"incoming_path"`
	ExistingPageCount    int                 `json:"existing_page_count"`
	IncomingPageCount    int                 `json:"incoming_page_count"`
	Status               string              `json:"status"`
	ResolutionNote       string              `json:"resolution_note"`
	CreatedAt            time.Time           `json:"created_at"`
	ResolvedAt           *time.Time          `json:"resolved_at"`
}
