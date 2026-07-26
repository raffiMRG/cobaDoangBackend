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

		copiedBytes, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return written, skipped, copyErr
		}

		// Guards against a client that reports a non-empty part but the body
		// read as 0 bytes — io.Copy treats that as a clean, error-free
		// transfer (0 bytes isn't an I/O error), so this is the only place
		// left to catch it. Concretely: the translate worker retrying an
		// upload after a 401 with an already-EOF'd file handle used to sail
		// through exactly this path and leave a "successful" completed
		// translation whose every page was a 0-byte file (see
		// zunks/feat/patch-error-translate-1.md).
		if fh.Size > 0 && copiedBytes == 0 {
			os.Remove(destPath)
			return written, skipped, fmt.Errorf("uploaded file %q is empty (expected %d bytes) — refusing to save a truncated file", fh.Filename, fh.Size)
		}

		written++
	}

	return written, skipped, nil
}
