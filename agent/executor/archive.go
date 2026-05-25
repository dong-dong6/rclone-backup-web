package executor

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ArchiveOptions struct {
	Password string
	Log      func(line string)
}

func CreateArchive(ctx context.Context, sourcePath string, destPath string, format string, opts *ArchiveOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "tar.gz", "tgz":
		return createTarGzArchive(ctx, sourcePath, destPath)
	case "zip":
		return createZipArchive(ctx, sourcePath, destPath)
	case "7z":
		return create7zArchive(ctx, sourcePath, destPath, opts)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func createTarGzArchive(ctx context.Context, sourcePath string, destPath string) error {
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
		if err := ctx.Err(); err != nil {
			return err
		}
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

		return copyWithContext(ctx, tw, f)
	})
}

func createZipArchive(ctx context.Context, sourcePath string, destPath string) error {
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
		if err := ctx.Err(); err != nil {
			return err
		}
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

		return copyWithContext(ctx, w, f)
	})
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}

func create7zArchive(ctx context.Context, sourcePath string, destPath string, opts *ArchiveOptions) error {
	sourcePath = filepath.Clean(sourcePath)
	destPath = filepath.Clean(destPath)

	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}

	exe, err := exec.LookPath("7z")
	if err != nil {
		if exeAlt, errAlt := exec.LookPath("7zz"); errAlt == nil {
			exe = exeAlt
		} else {
			return fmt.Errorf("7z not found: please install 7-Zip/p7zip on the agent host")
		}
	}

	baseDir := filepath.Dir(sourcePath)
	target := filepath.Base(sourcePath)
	if target == "" || target == "." {
		target = "."
	}

	args := []string{"a", "-t7z", "-y", destPath, target}
	if opts != nil {
		password := strings.TrimSpace(opts.Password)
		if password != "" {
			args = append(args, "-p"+password, "-mhe=on")
		}
	}

	if opts != nil && opts.Log != nil {
		opts.Log(formatCommandForLog(exe, args))
		opts.Log("[archive] 7z starting...")
	}

	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = baseDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	logFn := func(string) {}
	if opts != nil && opts.Log != nil {
		logFn = opts.Log
	}

	stream := func(r io.Reader, prefix string) {
		defer func() {
			_, _ = io.Copy(io.Discard, r)
		}()

		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		scanner.Split(scanLinesCRLF)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if logFn != nil {
				logFn(prefix + line)
			}
		}
		if err := scanner.Err(); err != nil && logFn != nil {
			logFn(prefix + "output reader error: " + err.Error())
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		stream(stdout, "[7z] ")
	}()
	go func() {
		defer wg.Done()
		stream(stderr, "[7z][stderr] ")
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	if waitErr != nil {
		return fmt.Errorf("7z archive failed: %w", waitErr)
	}

	if opts != nil && opts.Log != nil {
		opts.Log("[archive] 7z completed")
	}

	return nil
}
