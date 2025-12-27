package security

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aras/presto/internal/parser"
)

type Auditor struct {
	httpClient *http.Client
	osvURL     string
	githubURL  string
	packagist  string
}

type Vulnerability struct {
	Package     string
	Version     string
	CVE         string
	Severity    string
	Description string
	Fix         string
	Source      string
}

func NewAuditor() *Auditor {
	return &Auditor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		osvURL:    "https://api.osv.dev/v1/query",
		githubURL: "https://api.github.com/advisories",
		packagist: "https://packagist.org/api/security-advisories",
	}
}

func (a *Auditor) ScanProject(composer *parser.ComposerJSON) ([]*Vulnerability, error) {
	var vulnerabilities []*Vulnerability

	for pkg, version := range composer.Require {
		vulns, err := a.checkPackage(pkg, version)
		if err != nil {
			continue // Skip errors, continue scanning
		}
		vulnerabilities = append(vulnerabilities, vulns...)
	}

	return vulnerabilities, nil
}

func (a *Auditor) checkPackage(name, version string) ([]*Vulnerability, error) {
	if isPlatformPackage(name) {
		return nil, nil
	}

	var allVulns []*Vulnerability

	osvVulns, err := a.checkOSV(name, version)
	if err == nil && len(osvVulns) > 0 {
		allVulns = append(allVulns, osvVulns...)
	}
	packagistVulns, err := a.checkPackagist(name, version)
	if err == nil && len(packagistVulns) > 0 {
		allVulns = append(allVulns, packagistVulns...)
	}

	return deduplicateVulnerabilities(allVulns), nil
}

func (a *Auditor) checkOSV(name, version string) ([]*Vulnerability, error) {
	payload := map[string]interface{}{
		"package": map[string]string{
			"name":      name,
			"ecosystem": "Packagist",
		},
		"version": version,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", a.osvURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Presto/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var osvResp OSVResponse
	if err := json.NewDecoder(resp.Body).Decode(&osvResp); err != nil {
		return nil, err
	}

	var vulnerabilities []*Vulnerability
	for _, vuln := range osvResp.Vulns {
		severity := "MEDIUM"
		if len(vuln.Severity) > 0 {
			severity = vuln.Severity[0].Type
		}

		cveID := vuln.ID

		for _, alias := range vuln.Aliases {
			if strings.HasPrefix(alias, "CVE-") {
				cveID = alias
				break
			}
		}

		v := &Vulnerability{
			Package:     name,
			Version:     version,
			CVE:         cveID,
			Severity:    normalizeSeverity(severity),
			Description: vuln.Summary,
			Fix:         formatOSVFix(vuln.DatabaseSpecific),
			Source:      "OSV",
		}
		vulnerabilities = append(vulnerabilities, v)
	}

	return vulnerabilities, nil
}

func (a *Auditor) checkPackagist(name, version string) ([]*Vulnerability, error) {
	url := a.packagist + "?packages[]=" + name

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Presto/1.0") // version doesnt matter
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var advisories map[string][]Advisory
	if err := json.NewDecoder(resp.Body).Decode(&advisories); err != nil {
		return nil, err
	}

	var vulnerabilities []*Vulnerability

	for _, pkgAdvisories := range advisories {
		for _, advisory := range pkgAdvisories {
			if isVersionAffected(version, advisory.AffectedVersions) {
				vuln := &Vulnerability{
					Package:     name,
					Version:     version,
					CVE:         advisory.CVE,
					Severity:    determineSeverity(advisory.Title),
					Description: advisory.Title,
					Fix:         formatFix(advisory.Link),
					Source:      "Packagist",
				}
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	return vulnerabilities, nil
}

type OSVResponse struct {
	Vulns []OSVVulnerability `json:"vulns"`
}

type OSVVulnerability struct {
	ID               string                 `json:"id"`
	Summary          string                 `json:"summary"`
	Details          string                 `json:"details"`
	Aliases          []string               `json:"aliases"`
	Modified         string                 `json:"modified"`
	Published        string                 `json:"published"`
	Severity         []OSVSeverity          `json:"severity"`
	DatabaseSpecific map[string]interface{} `json:"database_specific"`
	References       []OSVReference         `json:"references"`
}

type OSVSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}
type OSVReference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Advisory struct {
	Title            string   `json:"title"`
	Link             string   `json:"link"`
	CVE              string   `json:"cve"`
	AffectedVersions string   `json:"affectedVersions"`
	Sources          []Source `json:"sources"`
}

type Source struct {
	Name string `json:"name"`
	URL  string `json:"remoteId"`
}

func isPlatformPackage(name string) bool {
	return name == "php" ||
		strings.HasPrefix(name, "ext-") ||
		strings.HasPrefix(name, "lib-")
}

func isVersionAffected(version, affectedVersions string) bool {
	version = strings.TrimPrefix(version, "v")

	if affectedVersions == "" {
		return true
	}

	ranges := strings.Split(affectedVersions, "|")
	for _, rangeStr := range ranges {
		if matchesRange(version, strings.TrimSpace(rangeStr)) {
			return true
		}
	}

	return false
}

func matchesRange(version, constraint string) bool {
	constraints := strings.Split(constraint, ",")

	for _, c := range constraints {
		c = strings.TrimSpace(c)
		if !matchesSingleConstraint(version, c) {
			return false
		}
	}

	return true
}

func matchesSingleConstraint(version, constraint string) bool {
	var operator, targetVersion string

	if strings.HasPrefix(constraint, ">=") {
		operator = ">="
		targetVersion = strings.TrimPrefix(constraint, ">=")
	} else if strings.HasPrefix(constraint, "<=") {
		operator = "<="
		targetVersion = strings.TrimPrefix(constraint, "<=")
	} else if strings.HasPrefix(constraint, ">") {
		operator = ">"
		targetVersion = strings.TrimPrefix(constraint, ">")
	} else if strings.HasPrefix(constraint, "<") {
		operator = "<"
		targetVersion = strings.TrimPrefix(constraint, "<")
	} else if strings.HasPrefix(constraint, "==") || strings.HasPrefix(constraint, "=") {
		operator = "=="
		targetVersion = strings.TrimPrefix(strings.TrimPrefix(constraint, "=="), "=")
	} else {
		// No operator, assume exact match
		operator = "=="
		targetVersion = constraint
	}

	targetVersion = strings.TrimSpace(targetVersion)
	cmp := compareVersions(version, targetVersion)

	switch operator {
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case "==":
		return cmp == 0
	default:
		return false
	}
}

func compareVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int

		if i < len(parts1) {
			numStr := strings.Split(parts1[i], "-")[0]
			p1, _ = strconv.Atoi(numStr)
		}

		if i < len(parts2) {
			numStr := strings.Split(parts2[i], "-")[0]
			p2, _ = strconv.Atoi(numStr)
		}

		if p1 < p2 {
			return -1
		} else if p1 > p2 {
			return 1
		}
	}

	return 0
}

func determineSeverity(title string) string {
	titleLower := strings.ToLower(title)

	if strings.Contains(titleLower, "critical") {
		return "CRITICAL"
	} else if strings.Contains(titleLower, "high") {
		return "HIGH"
	} else if strings.Contains(titleLower, "medium") {
		return "MEDIUM"
	} else if strings.Contains(titleLower, "low") {
		return "LOW"
	}

	if strings.Contains(titleLower, "remote code execution") ||
		strings.Contains(titleLower, "rce") ||
		strings.Contains(titleLower, "sql injection") {
		return "CRITICAL"
	} else if strings.Contains(titleLower, "xss") ||
		strings.Contains(titleLower, "csrf") ||
		strings.Contains(titleLower, "authentication") {
		return "HIGH"
	}

	return "MEDIUM"
}

func formatFix(link string) string {
	if link == "" {
		return "Update to the latest version"
	}
	return "See: " + link
}

func formatOSVFix(dbSpecific map[string]interface{}) string {
	if dbSpecific == nil {
		return "Update to the latest version"
	}

	if fixVersion, ok := dbSpecific["fixed_version"].(string); ok && fixVersion != "" {
		return "Update to version " + fixVersion + " or later"
	}

	if recommendation, ok := dbSpecific["recommendation"].(string); ok && recommendation != "" {
		return recommendation
	}

	return "Update to the latest version"
}

func normalizeSeverity(severity string) string {
	upper := strings.ToUpper(severity)

	if strings.Contains(upper, "CRITICAL") || upper == "CRITICAL" {
		return "CRITICAL"
	}
	if strings.Contains(upper, "HIGH") || upper == "HIGH" {
		return "HIGH"
	}
	if strings.Contains(upper, "MEDIUM") || upper == "MODERATE" || upper == "MEDIUM" {
		return "MEDIUM"
	}
	if strings.Contains(upper, "LOW") || upper == "LOW" {
		return "LOW"
	}

	return "MEDIUM"
}

func deduplicateVulnerabilities(vulns []*Vulnerability) []*Vulnerability {
	seen := make(map[string]bool)
	var result []*Vulnerability

	for _, v := range vulns {
		key := v.CVE
		if key == "" {
			key = v.Package + ":" + v.Description
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, v)
		}
	}

	return result
}
