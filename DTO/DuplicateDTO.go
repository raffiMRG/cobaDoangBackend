package dto

// DuplicateCandidateItem backs the /duplicates list page — one row per
// pending candidate, with just enough from both sides to render without
// touching the filesystem (see the caching rationale in
// zunks/feat/patch_handle_duplication-2.md).
type DuplicateCandidateItem struct {
	ID                  uint   `json:"id"`
	Name                string `json:"name"`
	ExistingNewFolderID uint   `json:"existing_new_folder_id"`
	ExistingThumbnail   string `json:"existing_thumbnail"`
	ExistingPageCount   int    `json:"existing_page_count"`
	IncomingPageCount   int    `json:"incoming_page_count"`
	CreatedAt           string `json:"created_at"`
}

// DuplicateComparePage is one page/image on either side of the compare view.
type DuplicateComparePage struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// DuplicateCompareResponse backs the /duplicates/:id/compare page — full
// page-by-page listing for both sides, read live off disk at request time
// (unlike the cached counts used for the list view).
type DuplicateCompareResponse struct {
	ID            uint                   `json:"id"`
	Name          string                 `json:"name"`
	ExistingPages []DuplicateComparePage `json:"existing_pages"`
	IncomingPages []DuplicateComparePage `json:"incoming_pages"`
}

// ResolveDuplicateRequest is the body for POST /duplicates/:id/resolve.
// Action is "new_title" (needs NewTitle) or "merge" (needs MergeMode,
// "replace" or "append").
type ResolveDuplicateRequest struct {
	Action    string `json:"action" binding:"required"`
	NewTitle  string `json:"new_title"`
	MergeMode string `json:"merge_mode"`
}

// BackfillResult backs POST /duplicates/backfill.
type BackfillResult struct {
	Converted int      `json:"converted"`
	Errors    []string `json:"errors"`
}
