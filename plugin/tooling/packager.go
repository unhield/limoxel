package tooling

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/plugin"
)

// PackageOptions configures plugin distribution packaging.
type PackageOptions struct {
	ProjectDir string `json:"project_dir"`
	BinaryPath string `json:"binary_path,omitempty"`
	OutputFile string `json:"output_file"`
}

// PackageResult captures packaging output metrics and integrity metadata.
type PackageResult struct {
	ArchivePath    string          `json:"archive_path"`
	Manifest       plugin.Manifest `json:"manifest"`
	ChecksumSHA256 string          `json:"checksum_sha256"`
	Size           int64           `json:"size"`
	FilesIncluded  []string        `json:"files_included"`
}

// Packager packages validated plugin assets into a standardized distribution bundle.
type Packager struct{}

// NewPackager constructs a Packager.
func NewPackager() *Packager {
	return &Packager{}
}

// Package archives the plugin manifest, compiled binary, and documentation.
func (p *Packager) Package(opts PackageOptions) (*PackageResult, error) {
	projDir := filepath.Clean(opts.ProjectDir)
	if projDir == "" || projDir == "." {
		projDir = "."
	}

	manifestPath := filepath.Join(projDir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("plugin manifest not found at %s: %w", manifestPath, err)
	}

	var manifest plugin.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest %s: %w", manifestPath, err)
	}

	outArchive := opts.OutputFile
	if strings.TrimSpace(outArchive) == "" {
		archiveName := fmt.Sprintf("%s-%s.tar.gz", manifest.ID, manifest.Version)
		outArchive = filepath.Join(projDir, "dist", archiveName)
	}
	outArchive = filepath.Clean(outArchive)

	if err := os.MkdirAll(filepath.Dir(outArchive), 0755); err != nil {
		return nil, fmt.Errorf("failed to create distribution directory: %w", err)
	}

	// Determine binary path
	binPath := opts.BinaryPath
	if binPath != "" {
		binPath = filepath.Clean(binPath)
		if _, err := os.Stat(binPath); err != nil {
			return nil, fmt.Errorf("specified binary not found at %s: %w", binPath, err)
		}
	}

	archiveFile, err := os.Create(outArchive)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive %s: %w", outArchive, err)
	}

	packageSuccess := false
	defer func() {
		_ = archiveFile.Close()
		if !packageSuccess {
			_ = os.Remove(outArchive)
		}
	}()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(archiveFile, hasher)

	gw := gzip.NewWriter(multiWriter)
	tw := tar.NewWriter(gw)

	var filesIncluded []string

	// 1. Add plugin.json
	if err := addFileToTar(tw, manifestPath, "plugin.json"); err != nil {
		_ = gw.Close()
		return nil, err
	}
	filesIncluded = append(filesIncluded, "plugin.json")

	// 2. Add Binary if present
	if binPath != "" {
		binName := filepath.Base(binPath)
		if err := addFileToTar(tw, binPath, filepath.Join("bin", binName)); err != nil {
			_ = gw.Close()
			return nil, err
		}
		filesIncluded = append(filesIncluded, filepath.Join("bin", binName))
	}

	// 3. Add README.md if present
	readmePath := filepath.Join(projDir, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		if err := addFileToTar(tw, readmePath, "README.md"); err == nil {
			filesIncluded = append(filesIncluded, "README.md")
		}
	}

	if err := tw.Close(); err != nil {
		_ = gw.Close()
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	fi, err := archiveFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat archive: %w", err)
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))
	packageSuccess = true

	return &PackageResult{
		ArchivePath:    outArchive,
		Manifest:       manifest,
		ChecksumSHA256: checksum,
		Size:           fi.Size(),
		FilesIncluded:  filesIncluded,
	}, nil
}

func addFileToTar(tw *tar.Writer, srcPath, tarRelPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	header.Name = filepath.ToSlash(tarRelPath)
	header.ModTime = info.ModTime().UTC()

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(tw, file); err != nil {
		return err
	}

	return nil
}
