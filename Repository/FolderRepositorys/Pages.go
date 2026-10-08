package FolderRepositorys

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	model "web_backend/Model"
	connection "web_backend/Model/Connection"
	"web_backend/Model/NewFolder"
)

// Page order and thumbnails share one definition: the reader's page list
// (GET /id/:id) and every stored thumbnail URL both come from PageFiles,
// so "thumbnail = page 1 in the reader" holds by construction. Before
// this, the reader sorted with PHP strnatcmp while thumbnails took
// os.ReadDir()[0], and the two disagreed on folders like 7675 (01.png vs
// 001.webp).

var (
	ErrFolderMissing = errors.New("folder tidak ditemukan di disk")
	ErrNoImage       = errors.New("folder tidak berisi file gambar")
)

var pageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".gif": true, ".avif": true, ".bmp": true,
}

// PageFiles lists the image files directly inside dir (no subfolders, no
// Thumbs.db/.txt and the like), in natural order: 1, 2, 10.
func PageFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrFolderMissing
		}
		return nil, err
	}

	pages := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && pageExtensions[strings.ToLower(filepath.Ext(e.Name()))] {
			pages = append(pages, e.Name())
		}
	}
	sort.Slice(pages, func(i, j int) bool { return naturalLess(pages[i], pages[j]) })
	return pages, nil
}

func firstPageFile(dir string) (string, error) {
	pages, err := PageFiles(dir)
	if err != nil {
		return "", err
	}
	if len(pages) == 0 {
		return "", ErrNoImage
	}
	return pages[0], nil
}

// naturalLess orders digit runs by numeric value and everything else
// case-insensitively. Names that compare equal that way ("01.png" vs
// "1.png") fall back to a plain string compare so the order is total and
// stable.
func naturalLess(a, b string) bool {
	if c := naturalCompare(a, b); c != 0 {
		return c < 0
	}
	return a < b
}

func naturalCompare(a, b string) int {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if isDigit(a[i]) && isDigit(b[j]) {
			si := i
			for i < len(a) && isDigit(a[i]) {
				i++
			}
			sj := j
			for j < len(b) && isDigit(b[j]) {
				j++
			}
			na := strings.TrimLeft(a[si:i], "0")
			nb := strings.TrimLeft(b[sj:j], "0")
			if len(na) != len(nb) {
				return sign(len(na) - len(nb))
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			continue
		}
		ca, cb := lower(a[i]), lower(b[j])
		if ca != cb {
			return sign(int(ca) - int(cb))
		}
		i++
		j++
	}
	return sign((len(a) - i) - (len(b) - j))
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// RepairNewFolderThumbnail recomputes one manga's thumbnail URL from what is
// actually on disk and stores it if the DB value differs in any way
// (encoding, wrong/missing file, truncated URL, old host). Never touches
// files on disk. Status: "ok" (already correct) or "fixed"; a missing
// folder or a folder without images is a 409 and leaves the DB unchanged.
func RepairNewFolderThumbnail(id string) model.BaseResponseModel {
	row, err := GetNewfolderRowFromId(id)
	if err != nil {
		return model.BaseResponseModel{CodeResponse: 404, HeaderMessage: "Error", Message: "manga tidak ditemukan", Data: nil}
	}
	if err := checkFolderName(row.Name); err != nil {
		return model.BaseResponseModel{CodeResponse: 400, HeaderMessage: "Bad Request", Message: err.Error(), Data: nil}
	}

	canonical, err := BuildNewFolderThumbnailURL(row.Name, filepath.Join(os.Getenv("DST_DIR"), row.Name))
	switch {
	case errors.Is(err, ErrFolderMissing):
		return model.BaseResponseModel{CodeResponse: 409, HeaderMessage: "Conflict",
			Message: "Folder \"" + row.Name + "\" tidak ditemukan di disk. Kalau folder di-rename di luar aplikasi, samakan namanya lewat Edit (tanpa 'apply to disk').",
			Data:    map[string]any{"status": "folder_missing", "old": row.Thumbnail}}
	case errors.Is(err, ErrNoImage):
		return model.BaseResponseModel{CodeResponse: 409, HeaderMessage: "Conflict",
			Message: "Folder \"" + row.Name + "\" tidak berisi file gambar.",
			Data:    map[string]any{"status": "no_image", "old": row.Thumbnail}}
	case err != nil:
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}

	data := map[string]any{"status": "ok", "old": row.Thumbnail, "new": canonical}
	if canonical == row.Thumbnail {
		return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "thumbnail sudah benar", Data: data}
	}

	if err := connection.DB.Model(&NewFolder.NewFolder{}).Where("id = ?", row.ID).Update("thumbnail", canonical).Error; err != nil {
		return model.BaseResponseModel{CodeResponse: 500, HeaderMessage: "Error", Message: err.Error(), Data: nil}
	}
	data["status"] = "fixed"
	return model.BaseResponseModel{CodeResponse: 200, HeaderMessage: "Success", Message: "thumbnail diperbaiki", Data: data}
}
