// Package gitref materializes Git revisions without modifying the working tree.
package gitref

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Export writes the files tracked by ref into destination using git archive.
// It does not check out the revision or otherwise modify the repository.
func Export(ref, destination string) error {
	repository, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current repository directory: %w", err)
	}
	return ExportFrom(repository, ref, destination)
}

// ExportFrom writes the files tracked by ref from repository into destination.
// It does not check out the revision or otherwise modify the repository.
func ExportFrom(repository, ref, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create Git export directory: %w", err)
	}

	command := exec.Command("git", "archive", "--format=tar", ref)
	command.Dir = repository
	stderr := &strings.Builder{}
	command.Stderr = stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("prepare Git archive for %q: %w", ref, err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start Git archive for %q: %w", ref, err)
	}

	extractErr := extractTar(stdout, destination)
	if extractErr != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return fmt.Errorf("extract Git reference %q: %w", ref, extractErr)
	}
	waitErr := command.Wait()
	if waitErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return fmt.Errorf("read Git reference %q: %s", ref, message)
		}
		return fmt.Errorf("read Git reference %q: %w", ref, waitErr)
	}
	return nil
}

func extractTar(reader io.Reader, destination string) error {
	archive := tar.NewReader(reader)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target, err := archivePath(destination, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			// Git may include PAX metadata records before regular files.
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&0o777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, archive)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported archive entry %q", header.Name)
		}
	}
}

func archivePath(destination, name string) (string, error) {
	path := filepath.Clean(filepath.FromSlash(name))
	if path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return filepath.Join(destination, path), nil
}
