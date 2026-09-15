package packagist

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aras/presto/internal/cache"
	"github.com/aras/presto/internal/httpx"
	version "github.com/shyim/go-version"
)

const (
	PackagistAPIURL = "https://repo.packagist.org"

	// MaxConnections caps the parallel manifest fetches. Packagist serves the p2
	// endpoints from a CDN and asks for no more.
	MaxConnections = 16

	// manifestTTL mirrors the "cache-control: max-age=900" packagist sends on the
	// p2 endpoints. Inside that window a cached manifest is used without asking.
	manifestTTL = 15 * time.Minute
)

// Client handles communication with Packagist API
type Client struct {
	httpClient *http.Client
	baseURL    string

	mu    sync.RWMutex
	cache map[string]*PackageInfo
}

// PackageInfo represents package metadata from Packagist
type PackageInfo struct {
	Name          string
	Description   string
	LatestVersion string
	Versions      map[string]*VersionInfo
	Downloads     int
	Favers        int
}

// VersionInfo represents a specific package version
type VersionInfo struct {
	Name              string            `json:"name"`
	Version           string            `json:"version"`
	VersionNormalized string            `json:"version_normalized"`
	Description       string            `json:"description"`
	Type              string            `json:"type"`
	Keywords          []string          `json:"keywords"`
	Homepage          string            `json:"homepage"`
	License           []string          `json:"license"`
	Authors           []Author          `json:"authors"`
	Require           map[string]string `json:"require"`
	RequireDev        map[string]string `json:"require-dev"`
	Conflict          map[string]string `json:"conflict"`
	Autoload          json.RawMessage   `json:"autoload"`
	Time              string            `json:"time"`
	Dist              DistInfo          `json:"dist"`
	Source            SourceInfo        `json:"source"`
	NotificationURL   string            `json:"notification-url"`
}

type Author struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Homepage string `json:"homepage,omitempty"`
	Role     string `json:"role,omitempty"`
}

// DistInfo represents distribution information
type DistInfo struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
	Shasum    string `json:"shasum"`
}

func (d *DistInfo) UnmarshalJSON(data []byte) error {
	if string(data) == "\"__unset\"" || string(data) == "null" {
		return nil
	}
	type Alias DistInfo
	return json.Unmarshal(data, (*Alias)(d))
}

type SourceInfo struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
}

// UnmarshalJSON handles "__unset" strings from Packagist API
func (s *SourceInfo) UnmarshalJSON(data []byte) error {
	if string(data) == "\"__unset\"" || string(data) == "null" {
		return nil
	}
	type Alias SourceInfo
	return json.Unmarshal(data, (*Alias)(s))
}

// PackagistResponse represents the API v2 response
type PackagistResponse struct {
	Packages map[string][]PackageVersionData `json:"packages"`
}

// PackageVersionData represents version data in the API response
type PackageVersionData struct {
	Version string `json:"version"`
}

// PackageMetadata represents the full package metadata
type PackageMetadata struct {
	Packages map[string]map[string]*VersionInfo `json:"packages"`
}

// NewClient creates a new Packagist client
func NewClient() *Client {
	return &Client{
		httpClient: httpx.New(30*time.Second, MaxConnections),
		baseURL:    PackagistAPIURL,
		cache:      make(map[string]*PackageInfo),
	}
}

// GetPackage fetches package information from Packagist. It is safe to call
// from several goroutines.
func (c *Client) GetPackage(name string) (*PackageInfo, error) {
	name = strings.ToLower(strings.TrimSpace(name))

	c.mu.RLock()
	cached, ok := c.cache[name]
	c.mu.RUnlock()

	if ok {
		return cached, nil
	}

	body, err := c.manifest(name)
	if err != nil {
		return nil, err
	}

	info, err := parseManifest(name, body)
	if err != nil {
		return nil, err
	}

	info.LatestVersion = findLatestStable(info.Versions)

	c.mu.Lock()
	c.cache[name] = info
	c.mu.Unlock()

	return info, nil
}

// manifest returns the raw p2 document, from disk when it is still fresh, from
// the network otherwise. A cached copy also answers when the network is down.
func (c *Client) manifest(name string) ([]byte, error) {
	path, err := cache.Metadata(name)
	if err != nil {
		path = ""
	}

	var (
		body []byte
		etag string
	)

	if path != "" {
		body, etag = readCachedManifest(path)

		if body != nil && cachedManifestIsFresh(path) {
			return body, nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/p2/%s.json", c.baseURL, name), nil)
	if err != nil {
		return nil, err
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if body != nil {
			return body, nil
		}

		return nil, fmt.Errorf("failed to fetch package: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusNotModified:
		now := time.Now()
		_ = os.Chtimes(path, now, now)

		return body, nil

	case http.StatusOK:
		fetched, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if path != "" {
			writeCachedManifest(path, fetched, resp.Header.Get("ETag"))
		}

		return fetched, nil

	default:
		if body != nil {
			return body, nil
		}

		return nil, fmt.Errorf("package not found: %s (status: %d)", name, resp.StatusCode)
	}
}

func cachedManifestIsFresh(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return time.Since(info.ModTime()) < manifestTTL
}

func readCachedManifest(path string) ([]byte, string) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, ""
	}

	etag, err := os.ReadFile(path + ".etag")
	if err != nil {
		return body, ""
	}

	return body, strings.TrimSpace(string(etag))
}

func writeCachedManifest(path string, body []byte, etag string) {
	if err := writeAtomic(path, body); err != nil {
		return
	}

	if etag == "" {
		_ = os.Remove(path + ".etag")
		return
	}

	_ = writeAtomic(path+".etag", []byte(etag))
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

func parseManifest(name string, body []byte) (*PackageInfo, error) {
	var apiResp struct {
		Packages map[string][]struct {
			Version         string          `json:"version"`
			Description     string          `json:"description"`
			Type            string          `json:"type"`
			Keywords        []string        `json:"keywords"`
			Homepage        string          `json:"homepage"`
			License         []string        `json:"license"`
			Authors         []Author        `json:"authors"`
			Require         json.RawMessage `json:"require"`
			RequireDev      json.RawMessage `json:"require-dev"`
			Conflict        json.RawMessage `json:"conflict"`
			Autoload        json.RawMessage `json:"autoload"`
			Time            string          `json:"time"`
			Dist            DistInfo        `json:"dist"`
			Source          SourceInfo      `json:"source"`
			NotificationURL string          `json:"notification-url"`
		} `json:"packages"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	versions, ok := apiResp.Packages[name]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for package: %s", name)
	}

	versionMap := make(map[string]*VersionInfo, len(versions))

	var description string

	for _, v := range versions {
		var requireDev map[string]string
		if len(v.RequireDev) > 0 && string(v.RequireDev) != "null" {
			_ = json.Unmarshal(v.RequireDev, &requireDev)
		}

		var require map[string]string
		if len(v.Require) > 0 && string(v.Require) != "null" {
			_ = json.Unmarshal(v.Require, &require)
		}

		var conflict map[string]string
		if len(v.Conflict) > 0 && string(v.Conflict) != "null" {
			_ = json.Unmarshal(v.Conflict, &conflict)
		}

		versionMap[v.Version] = &VersionInfo{
			Name:            name,
			Version:         v.Version,
			Description:     v.Description,
			Type:            v.Type,
			Keywords:        v.Keywords,
			Homepage:        v.Homepage,
			License:         v.License,
			Authors:         v.Authors,
			Require:         require,
			RequireDev:      requireDev,
			Conflict:        conflict,
			Autoload:        v.Autoload,
			Time:            v.Time,
			Dist:            v.Dist,
			Source:          v.Source,
			NotificationURL: v.NotificationURL,
		}

		if v.Description != "" && description == "" {
			description = v.Description
		}
	}

	return &PackageInfo{
		Name:        name,
		Description: description,
		Versions:    versionMap,
	}, nil
}

// findLatestStable picks the newest release, falling back to the newest
// prerelease when a package has never had a stable one.
func findLatestStable(versions map[string]*VersionInfo) string {
	names := make([]string, 0, len(versions))
	for name := range versions {
		names = append(names, name)
	}

	sort.Strings(names)

	var (
		stable    *version.Version
		stableRaw string
		any       *version.Version
		anyRaw    string
	)

	for _, candidate := range names {
		parsed, err := version.NewVersion(candidate)
		if err != nil {
			continue
		}

		if any == nil || parsed.GreaterThan(any) {
			any, anyRaw = parsed, candidate
		}

		if version.Stability(candidate) != "stable" {
			continue
		}

		if stable == nil || parsed.GreaterThan(stable) {
			stable, stableRaw = parsed, candidate
		}
	}

	if stableRaw != "" {
		return stableRaw
	}

	return anyRaw
}

// RecommendedConstraint turns a release into the constraint Composer writes into
// composer.json: ^major.minor, or ^0.minor.patch below 1.0 where a minor bump is
// already a breaking change. An exact version belongs in composer.lock, not here,
// or nothing can ever be updated.
func RecommendedConstraint(release string) string {
	if version.Stability(release) != "stable" {
		return release
	}

	parsed, err := version.NewVersion(release)
	if err != nil {
		return release
	}

	if parsed.Major() == 0 {
		return fmt.Sprintf("^0.%d.%d", parsed.Minor(), parsed.Patch())
	}

	return fmt.Sprintf("^%d.%d", parsed.Major(), parsed.Minor())
}

// GetVersion fetches a specific version of a package
func (c *Client) GetVersion(name, version string) (*VersionInfo, error) {
	info, err := c.GetPackage(name)
	if err != nil {
		return nil, err
	}

	versionInfo, ok := info.Versions[version]
	if !ok {
		return nil, fmt.Errorf("version %s not found for package %s", version, name)
	}

	return versionInfo, nil
}

// SearchPackages searches for packages on Packagist
func (c *Client) SearchPackages(query string) ([]*PackageInfo, error) {
	url := fmt.Sprintf("%s/search.json?q=%s", c.baseURL, query)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResp struct {
		Results []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Downloads   int    `json:"downloads"`
			Favers      int    `json:"favers"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, err
	}

	var packages []*PackageInfo
	for _, result := range searchResp.Results {
		packages = append(packages, &PackageInfo{
			Name:        result.Name,
			Description: result.Description,
			Downloads:   result.Downloads,
			Favers:      result.Favers,
		})
	}

	return packages, nil
}

// DownloadPackage returns the download URL for a package version
func (c *Client) DownloadPackage(name, version string) (string, error) {
	versionInfo, err := c.GetVersion(name, version)
	if err != nil {
		return "", err
	}

	if versionInfo.Dist.URL != "" {
		return versionInfo.Dist.URL, nil
	}

	// Fallback to source if dist is missing
	if versionInfo.Source.URL != "" {
		url := versionInfo.Source.URL
		ref := versionInfo.Source.Reference

		// If it's a Git URL, try to convert it to a ZIP download URL
		// as our downloader only supports ZIPs for now.
		if versionInfo.Source.Type == "git" {
			// GitHub: https://github.com/user/repo -> https://github.com/user/repo/archive/{ref}.zip
			if strings.Contains(url, "github.com") {
				repoURL := strings.TrimSuffix(url, ".git")
				return fmt.Sprintf("%s/archive/%s.zip", repoURL, ref), nil
			}
			// Codeberg: https://codeberg.org/user/repo -> https://codeberg.org/user/repo/archive/{ref}.zip
			if strings.Contains(url, "codeberg.org") {
				repoURL := strings.TrimSuffix(url, ".git")
				return fmt.Sprintf("%s/archive/%s.zip", repoURL, ref), nil
			}
			// GitLab: https://gitlab.com/user/repo -> https://gitlab.com/user/repo/-/archive/{ref}/repo-{ref}.zip
			if strings.Contains(url, "gitlab.com") {
				repoURL := strings.TrimSuffix(url, ".git")
				// Simple fallback for GitLab
				return fmt.Sprintf("%s/-/archive/%s/archive.zip", repoURL, ref), nil
			}
		}

		return url, nil
	}

	return "", fmt.Errorf("no download URL found for %s@%s", name, version)
}
