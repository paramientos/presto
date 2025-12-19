package packagist

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
	Autoload          interface{}       `json:"autoload"`
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

// SourceInfo represents source repository information
type SourceInfo struct {
	Type      string `json:"type"`
	URL       string `json:"url"`
	Reference string `json:"reference"`
}

// PackagistResponse represents the API response
type PackagistResponse struct {
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

	// Build API URL
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

	// Parse response
	var apiResp PackagistResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract package info
	packageVersions, ok := apiResp.Packages[name]
	if !ok || len(packageVersions) == 0 {
		return nil, fmt.Errorf("no versions found for package: %s", name)
	}

	info := &PackageInfo{
		Name:     name,
		Versions: packageVersions,
	}

	// Find latest version
	for version, versionInfo := range packageVersions {
		if versionInfo.Description != "" && info.Description == "" {
			info.Description = versionInfo.Description
		}

		// Simple latest version detection (should use semver)
		if info.LatestVersion == "" || !strings.Contains(version, "dev") {
			info.LatestVersion = version
		}
	}

	// Cache the result
	c.cache[name] = info

	return info, nil
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

	if versionInfo.Source.URL != "" {
		return versionInfo.Source.URL, nil
	}

	return "", fmt.Errorf("no download URL found for %s@%s", name, version)
}
