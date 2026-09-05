package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func pinDestPath(root, name string, pin ArtifactPin) (string, error) {
	rel := strings.TrimSpace(pin.Dest)
	if rel == "" {
		return deployBinPath(root, name), nil
	}
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("artifact %s dest %q must be a relative path inside the deploy root", name, rel)
	}
	return filepath.Join(root, filepath.Clean(rel)), nil
}

func pinSidecarPath(root, name, dest string, pin ArtifactPin) string {
	if strings.TrimSpace(pin.Unpack) != "" {
		return filepath.Join(root, "artifacts", name+".sha")
	}
	return shaSidecarPath(dest)
}

func installFetchedPin(root, name string, pin ArtifactPin, tmp string) error {
	dest, err := pinDestPath(root, name, pin)
	if err != nil {
		return err
	}
	unpack := strings.TrimSpace(pin.Unpack)
	switch unpack {
	case "":
		return installAtomic(tmp, dest)
	case "tar.gz", "tgz":
		return extractTarGzAtomic(tmp, dest)
	default:
		return fmt.Errorf("artifact %s: unsupported unpack %q", name, unpack)
	}
}

func extractTarGzAtomic(src, dest string) error {
	staging := dest + ".new"
	_ = os.RemoveAll(staging)
	if err := extractTarGz(src, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	backup := dest + ".old"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, backup); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
	}
	if err := os.Rename(staging, dest); err != nil {
		if _, berr := os.Stat(backup); berr == nil {
			_ = os.Rename(backup, dest)
		}
		_ = os.RemoveAll(staging)
		return err
	}
	_ = os.RemoveAll(backup)
	return nil
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := tarEntryPath(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			link := filepath.Clean(strings.ReplaceAll(hdr.Linkname, "\\", "/"))
			if filepath.IsAbs(link) || !filepath.IsLocal(link) {
				return fmt.Errorf("tar symlink escapes dest: %s -> %s", hdr.Name, hdr.Linkname)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := hdr.FileInfo().Mode().Perm()
			if mode == 0 {
				mode = 0o644
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			log.Printf("[deploy-sync] skip tar entry type=%v name=%s", hdr.Typeflag, hdr.Name)
		}
	}
}

func tarEntryPath(dest, name string) (string, error) {
	clean := filepath.Clean(strings.ReplaceAll(name, "\\", "/"))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "." || clean == "" {
		return dest, nil
	}
	if !filepath.IsLocal(clean) {
		return "", fmt.Errorf("tar entry escapes dest: %s", name)
	}
	return filepath.Join(dest, clean), nil
}
