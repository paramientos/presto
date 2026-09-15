package downloader

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aras/presto/internal/cache"
	"github.com/aras/presto/internal/httpx"
	"github.com/aras/presto/internal/resolver"
)

// maxExtractors caps concurrent extraction. Downloading is latency-bound and
// wants many workers; writing thousands of small files is disk-bound and slows
// down when oversubscribed, so the two are limited separately.
const maxExtractors = 8

// Downloader handles parallel package downloads
type Downloader struct {
	workers    int
	httpClient *http.Client
	vendorDir  string
	extracting chan struct{}
}

// NewDownloader creates a new downloader with specified number of workers
func NewDownloader(workers int) *Downloader {
	extractors := runtime.NumCPU()
	if extractors > maxExtractors {
		extractors = maxExtractors
	}

	if extractors < 1 {
		extractors = 1
	}

	return &Downloader{
		workers:    workers,
		httpClient: httpx.New(5*time.Minute, workers),
		vendorDir:  "vendor",
		extracting: make(chan struct{}, extractors),
	}
}

// Progress is called as each package lands, with the number finished so far.
type Progress func(done, total int, name string)

// DownloadAll fetches every package in parallel and returns the ones that were
// not already in the vendor directory, sorted by name.
func (d *Downloader) DownloadAll(packages []*resolver.Package, progress Progress) ([]*resolver.Package, error) {
	if err := os.MkdirAll(d.vendorDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create vendor directory: %w", err)
	}

	jobs := make(chan *resolver.Package, len(packages))
	errs := make(chan error, len(packages))

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		done      int
		installed []*resolver.Package
	)

	for i := 0; i < d.workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for pkg := range jobs {
				fetched, err := d.downloadPackage(pkg)
				if err != nil {
					errs <- fmt.Errorf("failed to download %s: %w", pkg.Name, err)
					continue
				}

				mu.Lock()
				done++
				if fetched {
					installed = append(installed, pkg)
				}
				if progress != nil {
					progress(done, len(packages), pkg.Name)
				}
				mu.Unlock()
			}
		}()
	}

	for _, pkg := range packages {
		jobs <- pkg
	}
	close(jobs)

	wg.Wait()
	close(errs)

	var downloadErrors []error
	for err := range errs {
		downloadErrors = append(downloadErrors, err)
	}

	if len(downloadErrors) > 0 {
		return nil, fmt.Errorf("download errors: %v", downloadErrors)
	}

	sort.Slice(installed, func(i, j int) bool {
		return installed[i].Name < installed[j].Name
	})

	return installed, nil
}

// downloadPackage reports whether it had to install the package. The archive is
// kept in the shared cache, so wiping vendor/ costs no network the second time.
func (d *Downloader) downloadPackage(pkg *resolver.Package) (bool, error) {
	packageDir := filepath.Join(d.vendorDir, pkg.Name)
	if _, err := os.Stat(packageDir); err == nil {
		return false, nil
	}

	archive, err := cache.Archive(pkg.Name, pkg.Version, pkg.URL)
	if err != nil {
		return false, fmt.Errorf("failed to open the package cache: %w", err)
	}

	reused := true

	if _, err := os.Stat(archive); err != nil {
		reused = false

		if err := d.fetch(pkg.URL, archive); err != nil {
			return false, err
		}
	}

	if err := d.extract(archive, packageDir); err != nil {
		_ = os.Remove(archive)
		_ = os.RemoveAll(packageDir)

		if !reused {
			return false, fmt.Errorf("extraction failed: %w", err)
		}

		return d.downloadPackage(pkg)
	}

	return true, nil
}

// fetch downloads into a sibling temp file and renames, so a killed download
// never leaves a half-written archive in the cache.
func (d *Downloader) fetch(url, dest string) error {
	resp, err := d.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("download failed: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), dest)
}

func (d *Downloader) extract(archive, dest string) error {
	d.extracting <- struct{}{}
	defer func() { <-d.extracting }()

	return d.extractZip(archive, dest)
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

// DownloadPackage reports whether it had to fetch the package.
func (d *Downloader) DownloadPackage(pkg *resolver.Package) (bool, error) {
	return d.downloadPackage(pkg)
}

// SetVendorDir sets the vendor directory path
func (d *Downloader) SetVendorDir(dir string) {
	d.vendorDir = dir
}
