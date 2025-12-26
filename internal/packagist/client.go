package packagist

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

const (
	PackagistAPIURL = "https://repo.packagist.org"
	CacheDir        = ".presto/cache"
)

// Client handles communication with Packagist API
type Client struct {
	httpClient *http.Client
	baseURL    string
	cache      map[string]*PackageInfo
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
	Authors           []interface{}     `json:"authors"`
	Require           map[string]string `json:"require"`
	RequireDev        map[string]string `json:"require-dev"`
	Autoload          json.RawMessage   `json:"autoload"`
	Time              string            `json:"time"`
	Dist              DistInfo          `json:"dist"`
	Source            SourceInfo        `json:"source"`
}

// DistInfo represents distribution information
type DistInfo struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
	Shasum    string `json:"shasum"`
}

// UnmarshalJSON handles "__unset" strings from Packagist API
func (d *DistInfo) UnmarshalJSON(data []byte) error {
	if string(data) == "\"__unset\"" || string(data) == "null" {
		return nil
	}
	type Alias DistInfo
	return json.Unmarshal(data, (*Alias)(d))
}

// SourceInfo represents source repository information
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
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: PackagistAPIURL,
		cache:   make(map[string]*PackageInfo),
	}
}

// GetPackage fetches package information from Packagist
func (c *Client) GetPackage(name string) (*PackageInfo, error) {
	// Check cache
	if cached, ok := c.cache[name]; ok {
		return cached, nil
	}

	// Normalize package name
	name = strings.ToLower(strings.TrimSpace(name))

	// Use the p2 API endpoint (metadata v2)
	url := fmt.Sprintf("%s/p2/%s.json", c.baseURL, name)

	// Make request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package not found: %s (status: %d)", name, resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response - Packagist v2 format has "packages" with package name as key
	var apiResp struct {
		Packages map[string][]struct {
			Version     string          `json:"version"`
			Description string          `json:"description"`
			Type        string          `json:"type"`
			Require     json.RawMessage `json:"require"`     // Can be null, [], {}, or map
			RequireDev  json.RawMessage `json:"require-dev"` // Can be null, [], {}, or map
			Autoload    json.RawMessage `json:"autoload"`    // Use RawMessage for debugging
			Dist        DistInfo        `json:"dist"`
			Source      SourceInfo      `json:"source"`
		} `json:"packages"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Get versions for this package
	versions, ok := apiResp.Packages[name]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("no versions found for package: %s", name)
	}

	// Convert to our format
	versionMap := make(map[string]*VersionInfo)
	var description string

	for _, v := range versions {
		// Parse require-dev flexibly
		var requireDev map[string]string
		if len(v.RequireDev) > 0 && string(v.RequireDev) != "null" {
			// Try to unmarshal as map, ignore errors if it's an array (empty requirements)
			_ = json.Unmarshal(v.RequireDev, &requireDev)
		}

		// Parse require flexibly
		var require map[string]string
		if len(v.Require) > 0 && string(v.Require) != "null" {

			// Try to unmarshal as map, ignore errors if it's an array (empty requirements)
			_ = json.Unmarshal(v.Require, &require)
		}

		versionMap[v.Version] = &VersionInfo{
			Name:        name,
			Version:     v.Version,
			Description: v.Description,
			Type:        v.Type,
			Require:     require,
			RequireDev:  requireDev,
			Autoload:    v.Autoload,
			Dist:        v.Dist,
			Source:      v.Source,
		}

		if v.Description != "" && description == "" {
			description = v.Description
		}
	}

	info := &PackageInfo{
		Name:        name,
		Description: description,
		Versions:    versionMap,
	}

	// Find latest stable version
	info.LatestVersion = c.findLatestStable(versionMap)

	// Cache the result
	c.cache[name] = info

	return info, nil
}

// findLatestStable finds the latest stable version
func (c *Client) findLatestStable(versions map[string]*VersionInfo) string {
	var latest string
	var latestVer *semver.Version

	for vStr, _ := range versions {
		// Skip dev versions
		if strings.Contains(vStr, "dev") {
			continue
		}

		// Parse version
		v, err := semver.NewVersion(vStr)
		if err != nil {
			// If not a valid semver, try a simple comparison as fallback
			if latest == "" || vStr > latest {
				// Only if it doesn't look like a pre-release
				if !strings.Contains(vStr, "alpha") &&
					!strings.Contains(vStr, "beta") &&
					!strings.Contains(vStr, "RC") {
					latest = vStr
				}
			}
			continue
		}

		// Skip pre-releases for "latest stable"
		if v.Prerelease() != "" {
			continue
		}

		if latestVer == nil || v.GreaterThan(latestVer) {
			latestVer = v
			latest = vStr
		}
	}

	// If no stable found, return any version (prefer non-dev)
	if latest == "" {
		for vStr := range versions {
			if !strings.Contains(vStr, "dev") {
				return vStr
			}
		}
		// Final fallback: just return any
		for vStr := range versions {
			return vStr
		}
	}

	return latest
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
