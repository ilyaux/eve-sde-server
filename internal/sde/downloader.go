package sde

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultSDEURL = "https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip"
	userAgent     = "EVE-SDE-Server/1.0 (https://github.com/ilyaux/eve-sde-server)"
)

// Downloader downloads and extracts CCP's EVE Online SDE archive.
type Downloader struct {
	sdeURL     string
	dataDir    string
	httpClient *http.Client
}

// NewDownloader creates an SDE downloader. Empty values use production defaults.
func NewDownloader(sdeURL, dataDir string) *Downloader {
	if sdeURL == "" {
		sdeURL = DefaultSDEURL
	}
	if dataDir == "" {
		dataDir = "data/sde"
	}

	return &Downloader{
		sdeURL:  sdeURL,
		dataDir: dataDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

// Download downloads the SDE zip file and returns the local path and SHA-256 checksum.
func (d *Downloader) Download() (string, string, error) {
	if err := os.MkdirAll(d.dataDir, 0755); err != nil {
		return "", "", fmt.Errorf("create data dir: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, d.sdeURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download SDE: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download SDE: unexpected status %s", resp.Status)
	}

	tmp, err := os.CreateTemp(d.dataDir, "sde-*.zip")
	if err != nil {
		return "", "", fmt.Errorf("create temp zip: %w", err)
	}
	tmpPath := tmp.Name()

	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(tmp, hash), resp.Body)
	closeErr := tmp.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("write zip: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("close zip: %w", closeErr)
	}

	zipPath := filepath.Join(d.dataDir, "sde.zip")
	if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("replace existing zip: %w", err)
	}
	if err := os.Rename(tmpPath, zipPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", "", fmt.Errorf("move zip into place: %w", err)
	}

	return zipPath, hex.EncodeToString(hash.Sum(nil)), nil
}

// Extract extracts a downloaded SDE zip into destDir.
func (d *Downloader) Extract(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()

	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("clear extract dir: %w", err)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create extract dir: %w", err)
	}

	for _, file := range reader.File {
		target, err := safeExtractPath(destDir, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return fmt.Errorf("create directory %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", file.Name, err)
		}

		dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = src.Close()
			return fmt.Errorf("create extracted file %s: %w", target, err)
		}

		_, copyErr := io.Copy(dst, src)
		closeSrcErr := src.Close()
		closeDstErr := dst.Close()
		if copyErr != nil {
			return fmt.Errorf("extract %s: %w", file.Name, copyErr)
		}
		if closeSrcErr != nil {
			return fmt.Errorf("close zip entry %s: %w", file.Name, closeSrcErr)
		}
		if closeDstErr != nil {
			return fmt.Errorf("close extracted file %s: %w", target, closeDstErr)
		}
	}

	return nil
}

func safeExtractPath(destDir, zipName string) (string, error) {
	if strings.TrimSpace(zipName) == "" {
		return "", fmt.Errorf("zip entry has empty name")
	}

	target := filepath.Join(destDir, filepath.FromSlash(zipName))
	destAbs, err := filepath.Abs(destDir)
	if err != nil {
		return "", fmt.Errorf("resolve extract dir: %w", err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve zip entry: %w", err)
	}

	rel, err := filepath.Rel(destAbs, targetAbs)
	if err != nil {
		return "", fmt.Errorf("validate zip entry path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("unsafe zip entry path: %s", zipName)
	}

	return targetAbs, nil
}
