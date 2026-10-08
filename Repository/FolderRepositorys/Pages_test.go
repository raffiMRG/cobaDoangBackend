package FolderRepositorys

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func makeFolder(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		p := filepath.Join(dir, f)
		if f[len(f)-1] == '/' {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestPageFilesNaturalOrderImagesOnly(t *testing.T) {
	dir := makeFolder(t, "10.jpg", "2.jpg", "1.jpg", "Thumbs.db", "notes.txt", "extra/", "COVER.PNG")
	got, err := PageFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	// digits sort before letters, numbers by value, extension case-insensitive
	want := []string{"1.jpg", "2.jpg", "10.jpg", "COVER.PNG"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Folders with two revisions mixed in (zunks/error/E02-duplicates.md, e.g.
// 7675): the order must be total and stable, and the thumbnail's file must
// be exactly the reader's page 1.
func TestMixedRevisionsStableAndThumbnailIsPageOne(t *testing.T) {
	dir := makeFolder(t, "002.webp", "01.png", "001.webp", "02.png", "1.jpg")
	got, _ := PageFiles(dir)
	// same page number -> next differing char decides (".jpg" < ".png" < ".webp")
	want := []string{"1.jpg", "01.png", "001.webp", "02.png", "002.webp"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if first, _ := firstPageFile(dir); first != got[0] {
		t.Fatalf("thumbnail file %q != reader page 1 %q", first, got[0])
	}
}

func TestFirstPageFileErrors(t *testing.T) {
	if _, err := firstPageFile(filepath.Join(t.TempDir(), "gone")); !errors.Is(err, ErrFolderMissing) {
		t.Fatalf("missing folder: %v", err)
	}
	if _, err := firstPageFile(makeFolder(t, "Thumbs.db", "sub/")); !errors.Is(err, ErrNoImage) {
		t.Fatalf("no images: %v", err)
	}
}

// The canonical URL must use the real folder name ("&", "'") correctly
// percent-encoded — never HTML entities — so it matches the file on disk.
func TestBuildNewFolderThumbnailURLEncoding(t *testing.T) {
	t.Setenv("API_BASEURL", "http://100.113.51.65:8080")
	name := "Me & Kasumi San's [English]"
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"02.jpg", "01.jpg"} {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}

	got, err := BuildNewFolderThumbnailURL(name, dir)
	want := "http://100.113.51.65:8080/new/Me%20&%20Kasumi%20San%27s%20%5BEnglish%5D/01.jpg"
	if err != nil || got != want {
		t.Fatalf("got %q (%v), want %q", got, err, want)
	}
}
