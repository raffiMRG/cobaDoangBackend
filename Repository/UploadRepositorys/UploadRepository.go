package UploadRepositorys

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// ErrInvalidName marks a rejected folder/file name as a client input error
// (400), distinct from I/O failures (500) — the Controller checks
// errors.Is(err, ErrInvalidName) to pick the right status code.
var ErrInvalidName = errors.New("invalid name")

// SanitizeName rejects path separators, ".." and empty names — folderName
// and each uploaded filename both come from a remote client over the
// network, so filepath.Join-ing them into SRC_DIR without this check would
// let a crafted name escape SRC_DIR entirely. Exported for reuse anywhere
// else a user-supplied name gets joined into a filesystem path (e.g.
// renaming a new_folders entry's on-disk directory).
func SanitizeName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("%w: name cannot be empty", ErrInvalidName)
	}
	base := filepath.Base(name)
	if base != name || base == "." || base == ".." {
		return "", fmt.Errorf("%w: %s", ErrInvalidName, name)
	}
	return base, nil
}

// SaveFolderFiles writes every uploaded file into srcDir/folderName,
// creating the folder if needed. A file whose name already exists in the
// destination is left untouched and counted as skipped rather than
// overwritten — a re-uploaded/updated zip should only add new pages, not
// clobber existing ones that may have been manually edited/renamed since.
func SaveFolderFiles(srcDir, folderName string, files []*multipart.FileHeader) (written, skipped int, err error) {
	safeFolderName, err := SanitizeName(folderName)
	if err != nil {
		return 0, 0, err
	}

	destDir := filepath.Join(srcDir, safeFolderName)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, 0, err
	}

	for _, fh := range files {
		safeFileName, err := SanitizeName(fh.Filename)
		if err != nil {
			return written, skipped, err
		}

		destPath := filepath.Join(destDir, safeFileName)
		if _, statErr := os.Stat(destPath); statErr == nil {
			skipped++
			continue
		}

		src, err := fh.Open()
		if err != nil {
			return written, skipped, err
		}

		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			return written, skipped, err
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return written, skipped, copyErr
		}

		written++
	}

	return written, skipped, nil
}
