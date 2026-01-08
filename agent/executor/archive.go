package executor

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CreateArchive(sourcePath string, destPath string, format string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "tar.gz", "tgz":
		return createTarGzArchive(sourcePath, destPath)
	case "zip":
		return createZipArchive(sourcePath, destPath)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func createTarGzArchive(sourcePath string, destPath string) error {
	sourcePath = filepath.Clean(sourcePath)
	destPath = filepath.Clean(destPath)

	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	gw.Name = filepath.Base(destPath)
	gw.ModTime = time.Now().UTC()
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	baseDir := filepath.Dir(sourcePath)

	return filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if name == "." || name == "" {
			name = filepath.ToSlash(filepath.Base(sourcePath))
		}

		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, target)
			if err != nil {
				return err
			}
			hdr.Name = name
			hdr.ModTime = time.Now().UTC()
			return tw.WriteHeader(hdr)
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = name
		hdr.ModTime = time.Now().UTC()

		if info.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !mode.IsRegular() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

func createZipArchive(sourcePath string, destPath string) error {
	sourcePath = filepath.Clean(sourcePath)
	destPath = filepath.Clean(destPath)

	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}

	outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	baseDir := filepath.Dir(sourcePath)

	return filepath.WalkDir(sourcePath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		// Skip symlinks for zip archives to avoid surprising behavior.
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if name == "." || name == "" {
			name = filepath.ToSlash(filepath.Base(sourcePath))
		}

		if info.IsDir() {
			if !strings.HasSuffix(name, "/") {
				name += "/"
			}
			hdr := &zip.FileHeader{
				Name:     name,
				Method:   zip.Deflate,
				Modified: time.Now().UTC(),
			}
			_, err := zw.CreateHeader(hdr)
			return err
		}

		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = name
		hdr.Method = zip.Deflate
		hdr.Modified = time.Now().UTC()

		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
}
