package parser

import (
	"fmt"
	"regexp"
	"strings"
)

type ValidationResult struct {
	Errors   []string
	Warnings []string
}

func (r *ValidationResult) IsValid(strict bool) bool {
	if len(r.Errors) > 0 {
		return false
	}
	if strict && len(r.Warnings) > 0 {
		return false
	}
	return true
}

func Validate(c *ComposerJSON) *ValidationResult {
	res := &ValidationResult{
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	if c.Name == "" {
		res.Errors = append(res.Errors, "The 'name' property is required")
	} else if !IsValidPackageName(c.Name) {
		res.Errors = append(res.Errors, fmt.Sprintf("The package name '%s' is invalid. It should be in 'vendor/package' format.", c.Name))
	}

	if c.Description == "" {
		res.Warnings = append(res.Warnings, "The 'description' property is recommended")
	}

	if c.License == "" {
		res.Warnings = append(res.Warnings, "The 'license' property is recommended")
	}

	if c.Type == "" {
		res.Warnings = append(res.Warnings, "The 'type' property is recommended (e.g., 'library', 'project')")
	}

	for pkg, version := range c.Require {
		if !isValidConstraint(version) {
			res.Errors = append(res.Errors, fmt.Sprintf("Invalid version constraint '%s' for package '%s'", version, pkg))
		}
	}

	for pkg, version := range c.RequireDev {
		if !isValidConstraint(version) {
			res.Errors = append(res.Errors, fmt.Sprintf("Invalid version constraint '%s' for package '%s' in require-dev", version, pkg))
		}
	}

	for pkg := range c.Require {
		if _, ok := c.RequireDev[pkg]; ok {
			res.Errors = append(res.Errors, fmt.Sprintf("Package '%s' is listed in both 'require' and 'require-dev'", pkg))
		}
	}

	if len(c.Autoload.PSR4) == 0 && len(c.Autoload.PSR0) == 0 && len(c.Autoload.Classmap) == 0 && len(c.Autoload.Files) == 0 {
		if c.Type == "library" {
			res.Warnings = append(res.Warnings, "A library should usually have an 'autoload' section")
		}
	}

	for ns := range c.Autoload.PSR4 {
		if !strings.HasSuffix(ns, "\\") {
			res.Warnings = append(res.Warnings, fmt.Sprintf("PSR-4 namespace '%s' should end with a backslash", ns))
		}
	}

	return res
}

var constraintRegex = regexp.MustCompile(`^(\*|dev-[^ ]+|v?[0-9]+(\.[0-9*]+)*|(\^|~|>=?|<=?|!=) ?v?[0-9]+(\.[0-9*]+)*|[0-9]+(\.[0-9*]+)*(\s*-\s*[0-9]+(\.[0-9*]+)*)?|@\w+)$`)

func isValidConstraint(constraint string) bool {
	if constraint == "" {
		return false
	}

	parts := strings.FieldsFunc(constraint, func(r rune) bool {
		return r == '|' || r == ',' || r == ' '
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "||" {
			continue
		}

	}

	return true
}
