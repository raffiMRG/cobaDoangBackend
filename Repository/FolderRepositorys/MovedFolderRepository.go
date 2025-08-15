package FolderRepositorys

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("gagal membuka file sumber %s: %v", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("gagal membuat file tujuan %s: %v", dst, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("gagal menyalin data dari %s ke %s: %v", src, dst, err)
	}

	return nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("gagal membaca direktori sumber %s: %v", src, err)
	}

	// Buat direktori tujuan jika belum ada
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("gagal membuat direktori tujuan %s: %v", dst, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dst, entry.Name())
		fmt.Println("srcPath : " + srcPath)
		fmt.Println("destPath : " + destPath)

		if entry.IsDir() {
			// Rekursif untuk direktori
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			// Salin file
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyPaste(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("gagal membaca direktori sumber: %v", err)
	}

	// Buat direktori tujuan jika belum ada
	if err := os.MkdirAll(destination, os.ModePerm); err != nil {
		return fmt.Errorf("gagal membuat direktori tujuan %s: %v", destination, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(source, entry.Name())
		destPath := filepath.Join(destination, entry.Name())

		if entry.IsDir() {
			// Salin direktori
			if err := copyDir(srcPath, destPath); err != nil {
				return fmt.Errorf("gagal menyalin direktori %s ke %s: %v", srcPath, destPath, err)
			}
		} else {
			// Salin file
			if err := copyFile(srcPath, destPath); err != nil {
				return fmt.Errorf("gagal menyalin file %s ke %s: %v", srcPath, destPath, err)
			}
		}

		fmt.Printf("Berhasil menyalin %s ke %s\n", srcPath, destPath)
	}

	return nil
}
