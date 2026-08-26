// Package DuplicateRepositorys implements the manual duplicate-review
// feature described in zunks/feat/patch_handle_duplication-2.md: when a
// folder scanned from SRC_DIR shares its name with an already-approved
// new_folders row, it's queued here instead of being silently inserted (or
// silently left to overwrite something later), and a human resolves it via
// the /duplicates page as either a brand new title or a merge into the
// existing entry.
package DuplicateRepositorys

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	dto "web_backend/DTO"
	connection "web_backend/Model/Connection"
	"web_backend/Model/DuplicateCandidate"
	"web_backend/Model/NewFolder"
	tbFolder "web_backend/Model/TbFolder"
	"web_backend/Repository/FolderRepositorys"
	"web_backend/Repository/UploadRepositorys"
)

// QueueCandidate records a scanned SRC_DIR folder as needing manual review
// because its name already matches an approved new_folders row. A no-op if
// this exact name is already queued and pending (re-scanning SRC_DIR
// shouldn't pile up duplicate queue entries).
func QueueCandidate(name, incomingPath string, existingNewFolderID uint) error {
	return QueueCandidateTx(connection.DB, name, incomingPath, existingNewFolderID)
}

// QueueCandidateTx is QueueCandidate against an explicit db/tx handle, so
// BackfillExistingCollisions can enqueue-and-delete-the-folders-row in one
// transaction.
func QueueCandidateTx(db *gorm.DB, name, incomingPath string, existingNewFolderID uint) error {
	var count int64
	if err := db.Model(&DuplicateCandidate.DuplicateCandidate{}).
		Where("name = ? AND status = ?", name, "pending").
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	incomingPages, err := FolderRepositorys.ScanFiles(incomingPath)
	if err != nil {
		return fmt.Errorf("gagal membaca folder incoming %q: %w", incomingPath, err)
	}

	existingFolder, err := FolderRepositorys.GetNewfolderRowFromId(strconv.Itoa(int(existingNewFolderID)))
	if err != nil {
		return fmt.Errorf("gagal mengambil data new_folders id=%d: %w", existingNewFolderID, err)
	}
	existingPages, err := FolderRepositorys.ScanFiles(filepath.Join(os.Getenv("DST_DIR"), existingFolder.Name))
	if err != nil {
		return fmt.Errorf("gagal membaca folder existing %q: %w", existingFolder.Name, err)
	}

	candidate := DuplicateCandidate.DuplicateCandidate{
		Name:                name,
		ExistingNewFolderID: existingNewFolderID,
		IncomingPath:        incomingPath,
		ExistingPageCount:   len(existingPages),
		IncomingPageCount:   len(incomingPages),
		Status:              "pending",
	}
	return db.Create(&candidate).Error
}

// ListPending backs the /duplicates list page. Deliberately doesn't touch
// the filesystem — existing/incoming page counts come from the columns
// cached at QueueCandidate time, same rationale as ScanDestinationFolderNames
// vs ScanFiles documented in FolderRepository.go: a list page shouldn't
// depend on storage being responsive to render.
func ListPending(page, limit int) ([]dto.DuplicateCandidateItem, int64, error) {
	db := connection.DB
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	if err := db.Model(&DuplicateCandidate.DuplicateCandidate{}).Where("status = ?", "pending").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []DuplicateCandidate.DuplicateCandidate
	if err := db.Where("status = ?", "pending").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]dto.DuplicateCandidateItem, 0, len(rows))
	for _, r := range rows {
		var nf NewFolder.NewFolder
		thumb := ""
		if err := db.First(&nf, r.ExistingNewFolderID).Error; err == nil {
			thumb = nf.Thumbnail
		}
		items = append(items, dto.DuplicateCandidateItem{
			ID:                  r.ID,
			Name:                r.Name,
			ExistingNewFolderID: r.ExistingNewFolderID,
			ExistingThumbnail:   thumb,
			ExistingPageCount:   r.ExistingPageCount,
			IncomingPageCount:   r.IncomingPageCount,
			CreatedAt:           r.CreatedAt.Format(time.RFC3339),
		})
	}
	return items, total, nil
}

func buildPageURL(staticSegment, folderName, fileName string) string {
	u, err := url.Parse(os.Getenv("API_BASEURL") + "/" + staticSegment + "/")
	if err != nil {
		return ""
	}
	u.Path = path.Join(u.Path, folderName, fileName)
	return u.String()
}

// GetCandidateForCompare backs the /duplicates/:id/compare page — unlike
// ListPending, this does read the filesystem live (both sides), since this
// is the one place that actually needs the current, exact page list.
func GetCandidateForCompare(idStr string) (*dto.DuplicateCompareResponse, error) {
	db := connection.DB
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, errors.New("invalid id")
	}

	var candidate DuplicateCandidate.DuplicateCandidate
	if err := db.First(&candidate, id).Error; err != nil {
		return nil, err
	}

	existingFolder, err := FolderRepositorys.GetNewfolderRowFromId(strconv.Itoa(int(candidate.ExistingNewFolderID)))
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data folder lama: %w", err)
	}

	existingFiles, err := FolderRepositorys.ScanFiles(filepath.Join(os.Getenv("DST_DIR"), existingFolder.Name))
	if err != nil {
		return nil, fmt.Errorf("gagal membaca folder lama: %w", err)
	}
	incomingFiles, err := FolderRepositorys.ScanFiles(candidate.IncomingPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca folder baru: %w", err)
	}
	sort.Strings(existingFiles)
	sort.Strings(incomingFiles)

	resp := &dto.DuplicateCompareResponse{ID: candidate.ID, Name: candidate.Name}
	for _, f := range existingFiles {
		resp.ExistingPages = append(resp.ExistingPages, dto.DuplicateComparePage{
			Name: f,
			URL:  buildPageURL("new", existingFolder.Name, f),
		})
	}
	for _, f := range incomingFiles {
		resp.IncomingPages = append(resp.IncomingPages, dto.DuplicateComparePage{
			Name: f,
			URL:  buildPageURL("sementara", candidate.Name, f),
		})
	}
	return resp, nil
}

// nameExistsAnywhere is the cross-table check that was missing entirely
// before this feature (see the "Celah" callout on Alur 1 in
// patch_handle_duplication.md) — used here so picking "buat judul baru"
// can't itself create a fresh collision.
func nameExistsAnywhere(db *gorm.DB, name string) (bool, error) {
	var count int64
	if err := db.Model(&tbFolder.Folder{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := db.Model(&NewFolder.NewFolder{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := db.Model(&DuplicateCandidate.DuplicateCandidate{}).Where("name = ? AND status = ?", name, "pending").Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ResolveNewTitle implements the "buat judul baru" action: rename the
// incoming SRC_DIR folder to a non-colliding title and insert it as a
// normal pending `folders` row, so both the old and new content survive.
func ResolveNewTitle(idStr, newTitle string) error {
	db := connection.DB
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.New("invalid id")
	}

	var candidate DuplicateCandidate.DuplicateCandidate
	if err := db.First(&candidate, id).Error; err != nil {
		return err
	}
	if candidate.Status != "pending" {
		return fmt.Errorf("kandidat ini sudah diresolve sebelumnya (status: %s)", candidate.Status)
	}

	safeTitle, err := UploadRepositorys.SanitizeName(newTitle)
	if err != nil {
		return err
	}
	if exists, err := nameExistsAnywhere(db, safeTitle); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("judul %q sudah dipakai, pilih judul lain", safeTitle)
	}

	newPath := filepath.Join(filepath.Dir(candidate.IncomingPath), safeTitle)
	if err := os.Rename(candidate.IncomingPath, newPath); err != nil {
		return fmt.Errorf("gagal rename folder di SRC_DIR: %w", err)
	}

	thumbnail, thumbErr := FolderRepositorys.BuildThumbnailURL(safeTitle, newPath)
	if thumbErr != nil {
		thumbnail = ""
	}

	now := time.Now()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tbFolder.Folder{Name: safeTitle, Thumbnail: thumbnail}).Error; err != nil {
			return err
		}
		return tx.Model(&candidate).Updates(map[string]any{
			"status":          "resolved_new_title",
			"resolution_note": safeTitle,
			"resolved_at":     now,
		}).Error
	})
}

// ResolveMerge implements the two "merge" actions:
//   - replace: the incoming version wins outright, overwriting the
//     existing DST_DIR folder (deliberately reuses CopyPaste's overwrite
//     behaviour — the user has explicitly opted into it here).
//   - append: incoming pages are copied in as new, renumbered pages after
//     the existing ones, so filenames never collide.
func ResolveMerge(idStr, mode string) error {
	if mode != "replace" && mode != "append" {
		return fmt.Errorf("merge_mode tidak dikenal: %q (harus 'replace' atau 'append')", mode)
	}

	db := connection.DB
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return errors.New("invalid id")
	}

	var candidate DuplicateCandidate.DuplicateCandidate
	if err := db.First(&candidate, id).Error; err != nil {
		return err
	}
	if candidate.Status != "pending" {
		return fmt.Errorf("kandidat ini sudah diresolve sebelumnya (status: %s)", candidate.Status)
	}

	existingFolder, err := FolderRepositorys.GetNewfolderRowFromId(strconv.Itoa(int(candidate.ExistingNewFolderID)))
	if err != nil {
		return fmt.Errorf("gagal mengambil data folder lama: %w", err)
	}
	destPath := filepath.Join(os.Getenv("DST_DIR"), existingFolder.Name)

	var newThumbnail string
	if mode == "replace" {
		if err := FolderRepositorys.CopyPaste(candidate.IncomingPath, destPath, nil); err != nil {
			return fmt.Errorf("gagal menyalin file (replace): %w", err)
		}
		if thumb, thumbErr := FolderRepositorys.BuildNewFolderThumbnailURL(existingFolder.Name, destPath); thumbErr == nil {
			newThumbnail = thumb
		}
	} else {
		if err := appendPages(candidate.IncomingPath, destPath); err != nil {
			return fmt.Errorf("gagal menyalin file (append): %w", err)
		}
	}

	if err := os.RemoveAll(candidate.IncomingPath); err != nil {
		return fmt.Errorf("gagal menghapus folder incoming setelah merge: %w", err)
	}

	now := time.Now()
	return db.Transaction(func(tx *gorm.DB) error {
		if newThumbnail != "" {
			if err := tx.Model(&NewFolder.NewFolder{}).Where("id = ?", existingFolder.ID).Update("thumbnail", newThumbnail).Error; err != nil {
				return err
			}
		}
		return tx.Model(&candidate).Updates(map[string]any{
			"status":          "resolved_merged",
			"resolution_note": mode,
			"resolved_at":     now,
		}).Error
	})
}

// appendPages copies every file from incomingPath into destPath, renumbered
// to continue after destPath's existing pages — renumbering (instead of
// reusing original filenames) is what avoids collisions like both sides
// having a "01.jpg" with different content.
func appendPages(incomingPath, destPath string) error {
	existing, err := FolderRepositorys.ScanFiles(destPath)
	if err != nil {
		return err
	}
	incoming, err := FolderRepositorys.ScanFiles(incomingPath)
	if err != nil {
		return err
	}
	sort.Strings(incoming)

	offset := len(existing)
	width := max(len(strconv.Itoa(offset+len(incoming))), 2) // e.g. "05.jpg" not "5.jpg", matching typical page-name padding

	for i, name := range incoming {
		ext := filepath.Ext(name)
		newName := fmt.Sprintf("%0*d%s", width, offset+i+1, ext)
		if err := copySingleFile(filepath.Join(incomingPath, name), filepath.Join(destPath, newName)); err != nil {
			return err
		}
	}
	return nil
}

func copySingleFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// BackfillExistingCollisions sweeps `folders` for rows that already collide
// by name with an approved `new_folders` row — data that predates this
// feature and so was never routed through QueueCandidate at scan time (see
// "Duplikasi yang SUDAH ada sekarang" in
// zunks/feat/patch_handle_duplication-2.md). Safe to re-run any time
// (idempotent: already-queued names are skipped, matches with no folder
// left on disk are reported as errors rather than failing the whole run).
func BackfillExistingCollisions() (*dto.BackfillResult, error) {
	db := connection.DB
	srcDir := os.Getenv("SRC_DIR")

	var matches []struct {
		FolderID    uint
		NewFolderID uint
		Name        string
	}
	query := "SELECT f.id AS folder_id, nf.id AS new_folder_id, f.name AS name " +
		"FROM folders f JOIN new_folders nf ON nf.name = f.name " +
		"ORDER BY nf.create_at DESC"
	if err := db.Raw(query).Scan(&matches).Error; err != nil {
		return nil, err
	}

	result := &dto.BackfillResult{Errors: []string{}}
	seen := make(map[string]bool, len(matches))

	for _, m := range matches {
		if seen[m.Name] {
			// Nama ini sudah dikonversi lewat pasangan lain di batch yang
			// sama (mis. beberapa new_folders row nama sama, self-duplikat
			// lama) — baris pertama (nf paling baru, berkat ORDER BY di
			// atas) yang dipakai, sisanya dilewati di sini.
			continue
		}
		seen[m.Name] = true

		incomingPath := filepath.Join(srcDir, m.Name)
		if _, statErr := os.Stat(incomingPath); statErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: folder tidak ditemukan di SRC_DIR (%v)", m.Name, statErr))
			continue
		}

		txErr := db.Transaction(func(tx *gorm.DB) error {
			if err := QueueCandidateTx(tx, m.Name, incomingPath, m.NewFolderID); err != nil {
				return err
			}
			return tx.Where("id = ?", m.FolderID).Delete(&tbFolder.Folder{}).Error
		})
		if txErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", m.Name, txErr))
			continue
		}
		result.Converted++
	}

	return result, nil
}
