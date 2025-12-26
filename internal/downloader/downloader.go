package downloader

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aras/presto/internal/resolver"
	"github.com/schollz/progressbar/v3"
)

// Downloader handles parallel package downloads
type Downloader struct {
	workers    int
	httpClient *http.Client
	vendorDir  string
}

// NewDownloader creates a new downloader with specified number of workers
func NewDownloader(workers int) *Downloader {
	return &Downloader{
		workers: workers,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		vendorDir: "vendor",
	}
}

// DownloadAll downloads all packages in parallel
func (d *Downloader) DownloadAll(packages []*resolver.Package) error {
	// Create vendor directory
	if err := os.MkdirAll(d.vendorDir, 0755); err != nil {
		return fmt.Errorf("failed to create vendor directory: %w", err)
	}

	// Create progress bar
	bar := progressbar.NewOptions(len(packages),
		progressbar.OptionSetDescription("⬇️  Downloading"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	// Create worker pool
	jobs := make(chan *resolver.Package, len(packages))
	errors := make(chan error, len(packages))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pkg := range jobs {
				if err := d.downloadPackage(pkg); err != nil {
					errors <- fmt.Errorf("failed to download %s: %w", pkg.Name, err)
				} else {
					_ = bar.Add(1)
				}
			}
		}()
	}

	// Send jobs
	for _, pkg := range packages {
		jobs <- pkg
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(errors)

	// Check for errors
	var downloadErrors []error
	for err := range errors {
		downloadErrors = append(downloadErrors, err)
	}

	if len(downloadErrors) > 0 {
		return fmt.Errorf("download errors: %v", downloadErrors)
	}

	_ = bar.Finish()
	fmt.Println()

	return nil
}

// downloadPackage downloads a single package
func (d *Downloader) downloadPackage(pkg *resolver.Package) error {
	// Skip if already downloaded
	packageDir := filepath.Join(d.vendorDir, pkg.Name)
	if _, err := os.Stat(packageDir); err == nil {
		return nil // Already exists
	}

	// Download archive
	resp, err := d.httpClient.Get(pkg.URL)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "presto-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Close file to ensure everything is flushed to disk before extraction
	tmpFile.Close()

	// Extract archive
	if err := d.extractZip(tmpFile.Name(), packageDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	return nil
}

// extractZip extracts a zip archive to the destination directory
func (d *Downloader) extractZip(zipPath, destDir string) error {
	// Open zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Extract files
	for _, file := range reader.File {
		// Get the file path
		path := filepath.Join(destDir, file.Name)

		// Remove the first directory component (package name with version)
		parts := strings.Split(file.Name, string(filepath.Separator))
		if len(parts) > 1 {
			path = filepath.Join(destDir, filepath.Join(parts[1:]...))
		}

		// Check for directory
		if file.FileInfo().IsDir() {
			_ = os.MkdirAll(path, file.Mode())
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Extract file
		if err := d.extractFile(file, path); err != nil {
			return err
		}
	}

	return nil
}

// extractFile extracts a single file from the zip archive
func (d *Downloader) extractFile(file *zip.File, destPath string) error {
	// Open file in archive
	srcFile, err := file.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy contents
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	return nil
}

// DownloadPackage downloads a single package (public method)
func (d *Downloader) DownloadPackage(pkg *resolver.Package) error {
	return d.downloadPackage(pkg)
}

// SetVendorDir sets the vendor directory path
func (d *Downloader) SetVendorDir(dir string) {
	d.vendorDir = dir
}
