package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aras/presto/internal/autoload"
	"github.com/aras/presto/internal/downloader"
	"github.com/aras/presto/internal/lockfile"
	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
	"github.com/aras/presto/internal/resolver"
	"github.com/aras/presto/internal/security"
	"github.com/spf13/cobra"
)

var version = "0.1.5"
var verbose bool

func main() {
	rootCmd := &cobra.Command{
		Use:   "presto",
		Short: "🎵 A blazing fast package manager for PHP",
		Long:  `Presto is a high-performance, drop-in replacement for Composer with killer features.`,
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("🎵 Presto v{{.Version}}\n")

	// Add global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Install command
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install dependencies from composer.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall()
		},
	}

	// Require command
	requireCmd := &cobra.Command{
		Use:   "require [packages...]",
		Short: "Add new packages to composer.json",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRequire(args)
		},
	}

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update [packages...]",
		Short: "Update dependencies to latest versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(args)
		},
	}

	// Remove command
	removeCmd := &cobra.Command{
		Use:   "remove [packages...]",
		Short: "Remove packages from composer.json",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args)
		},
	}

	// Show command
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show installed packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow()
		},
	}

	// Audit command (killer feature!)
	auditCmd := &cobra.Command{
		Use:   "audit",
		Short: "🔒 Scan for security vulnerabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit()
		},
	}

	// Why command (killer feature!)
	whyCmd := &cobra.Command{
		Use:   "why [package]",
		Short: "🔍 Show why a package is installed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhy(args[0])
		},
	}

	// Why-not command (killer feature!)
	whyNotCmd := &cobra.Command{
		Use:   "why-not [package] [version]",
		Short: "🚫 Show why a package version cannot be installed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhyNot(args[0], args[1])
		},
	}

	// Init command
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new composer.json file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}

	// Tree command
	treeCmd := &cobra.Command{
		Use:     "tree",
		Short:   "🌳 Show dependency tree",
		Aliases: []string{"map"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTree()
		},
	}

	// Cache command
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage package cache",
	}

	cacheClearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear package cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCacheClear()
		},
	}

	cacheCmd.AddCommand(cacheClearCmd)

	rootCmd.AddCommand(
		installCmd,
		requireCmd,
		updateCmd,
		removeCmd,
		showCmd,
		auditCmd,
		whyCmd,
		whyNotCmd,
		initCmd,
		treeCmd,
		cacheCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// logVerbose prints message only if verbose flag is set
func logVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Printf("🔍 [VERBOSE] "+format+"\n", args...)
	}
}

func runInstall() error {

	fmt.Println("🎵 Presto Install")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Parse composer.json
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return fmt.Errorf("failed to parse composer.json: %w", err)
	}

	fmt.Printf("📦 Project: %s\n", composer.Name)
	fmt.Printf("📝 Description: %s\n\n", composer.Description)

	// Create Packagist client
	client := packagist.NewClient()

	// Resolve dependencies
	fmt.Println("🔍 Resolving dependencies...")
	logVerbose("Starting dependency resolution for %d required packages", len(composer.Require))

	res := resolver.NewResolver(client)
	packages, err := res.Resolve(composer)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	fmt.Printf("✅ Resolved %d packages\n\n", len(packages))
	logVerbose("Resolved packages: %d", len(packages))
	for _, pkg := range packages {
		logVerbose("  - %s (%s) -> %s", pkg.Name, pkg.Version, pkg.URL)
	}

	// Download packages
	fmt.Println("⬇️  Downloading packages...")
	logVerbose("Starting download with %d workers", 8)

	dl := downloader.NewDownloader(8) // 8 parallel workers
	if err := dl.DownloadAll(packages); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Update autoload from downloaded packages (source of truth)
	fmt.Println("🔄 Updating package information...")
	for _, pkg := range packages {
		jsonPath := filepath.Join("vendor", pkg.Name, "composer.json")
		content, err := os.ReadFile(jsonPath)
		if err != nil {
			logVerbose("Could not read composer.json for %s: %v", pkg.Name, err)
			continue
		}

		var pkgJson struct {
			Autoload json.RawMessage `json:"autoload"`
		}
		if err := json.Unmarshal(content, &pkgJson); err == nil && len(pkgJson.Autoload) > 0 {
			pkg.Autoload = pkgJson.Autoload
			logVerbose("Updated autoload for %s from local composer.json", pkg.Name)
		}
	}

	// Generate autoload
	fmt.Println("\n📝 Generating autoload files...")
	logVerbose("Generating PSR-4 autoload files")

	gen := autoload.NewGenerator()
	if err := gen.Generate(composer, packages); err != nil {
		return fmt.Errorf("autoload generation failed: %w", err)
	}

	// Generate composer.lock
	fmt.Println("🔒 Generating composer.lock...")
	logVerbose("Generating lock file")

	lockGen := lockfile.NewGenerator()
	if err := lockGen.Generate(composer, packages); err != nil {
		return fmt.Errorf("lock file generation failed: %w", err)
	}

	fmt.Println("\n✨ Installation complete!")
	return nil
}

func runRequire(packages []string) error {
	fmt.Printf("🎵 Adding packages: %v\n", packages)

	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	client := packagist.NewClient()

	for _, pkg := range packages {
		fmt.Printf("🔍 Fetching %s...\n", pkg)
		info, err := client.GetPackage(pkg)
		if err != nil {
			return fmt.Errorf("package %s not found: %w", pkg, err)
		}

		// Add to composer.json
		if composer.Require == nil {
			composer.Require = make(map[string]string)
		}
		composer.Require[pkg] = info.LatestVersion

		fmt.Printf("✅ Added %s: %s\n", pkg, info.LatestVersion)
	}

	// Save composer.json
	if err := parser.WriteComposerJSON("composer.json", composer); err != nil {
		return err
	}

	// Run install
	return runInstall()
}

func runUpdate(packages []string) error {
	fmt.Println("🎵 Updating dependencies...")

	if len(packages) == 0 {
		fmt.Println("📦 Updating all packages")
	} else {
		fmt.Printf("📦 Updating: %v\n", packages)
	}

	// Re-run install to update
	return runInstall()
}

func runRemove(packages []string) error {
	fmt.Printf("🎵 Removing packages: %v\n", packages)

	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		delete(composer.Require, pkg)
		delete(composer.RequireDev, pkg)
		fmt.Printf("✅ Removed %s\n", pkg)
	}

	return parser.WriteComposerJSON("composer.json", composer)
}

func runShow() error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	fmt.Println("🎵 Installed Packages")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📦 Production Dependencies:")
	for pkg, version := range composer.Require {
		fmt.Printf("  • %s: %s\n", pkg, version)
	}

	if len(composer.RequireDev) > 0 {
		fmt.Println("\n🔧 Development Dependencies:")
		for pkg, version := range composer.RequireDev {
			fmt.Printf("  • %s: %s\n", pkg, version)
		}
	}

	return nil
}

func runAudit() error {
	fmt.Println("🎵 Security Audit")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	auditor := security.NewAuditor()
	vulnerabilities, err := auditor.ScanProject(composer)
	if err != nil {
		return err
	}

	if len(vulnerabilities) == 0 {
		fmt.Println("✅ No vulnerabilities found!")
		return nil
	}

	fmt.Printf("⚠️  Found %d vulnerabilities:\n\n", len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		fmt.Printf("[%s] %s@%s\n", vuln.Severity, vuln.Package, vuln.Version)
		fmt.Printf("  CVE: %s\n", vuln.CVE)
		fmt.Printf("  Description: %s\n", vuln.Description)
		fmt.Printf("  Fix: %s\n\n", vuln.Fix)
	}

	return nil
}

func runWhy(packageName string) error {
	fmt.Printf("🎵 Why is %s installed?\n", packageName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	// Check if directly required
	if version, ok := composer.Require[packageName]; ok {
		fmt.Printf("\n✅ Directly required in composer.json\n")
		fmt.Printf("   Version: %s\n", version)
		return nil
	}

	// Check dependencies (simplified)
	client := packagist.NewClient()
	res := resolver.NewResolver(client)
	tree, err := res.BuildDependencyTree(composer, packageName)
	if err != nil {
		return fmt.Errorf("not found in dependency tree: %w", err)
	}

	fmt.Println("\n📊 Dependency chain:")
	fmt.Println(tree)

	return nil
}

func runWhyNot(packageName, version string) error {
	fmt.Printf("🎵 Why can't %s@%s be installed?\n", packageName, version)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	client := packagist.NewClient()
	res := resolver.NewResolver(client)

	conflicts, err := res.CheckConflicts(composer, packageName, version)
	if err != nil {
		return err
	}

	if len(conflicts) == 0 {
		fmt.Println("✅ No conflicts! You can install this version.")
		return nil
	}

	fmt.Println("\n❌ Conflicts found:")

	for _, conflict := range conflicts {
		fmt.Printf("  • %s\n", conflict)
	}

	fmt.Println("\n💡 To install:")
	fmt.Println("  1. Update conflicting packages")
	fmt.Println("  2. Or use a different version")

	return nil
}

func runInit() error {
	fmt.Println("🎵 Initialize new project")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	composer := &parser.ComposerJSON{
		Name:        "vendor/project",
		Description: "A new PHP project",
		Type:        "project",
		License:     "MIT",
		Require: map[string]string{
			"php": "^8.1",
		},
		Autoload: parser.AutoloadConfig{
			PSR4: map[string]string{
				"App\\": "src/",
			},
		},
	}

	if err := parser.WriteComposerJSON("composer.json", composer); err != nil {
		return err
	}

	fmt.Println("✅ Created composer.json")
	return nil
}

func runCacheClear() error {
	fmt.Println("🎵 Clearing cache...")

	cacheDir := ".presto/cache"
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	logVerbose("Removed cache directory: %s", cacheDir)

	fmt.Println("✅ Cache cleared")
	return nil
}

func runTree() error {
	fmt.Println("🌳 Generating dependency map...")

	// 1. Parse composer.json
	pkgJson, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return fmt.Errorf("failed to parse composer.json: %w", err)
	}

	// 2. Resolve packages (to get the graph)
	client := packagist.NewClient()
	res := resolver.NewResolver(client)

	fmt.Println("🔍 Resolving dependencies (this may take a moment)...")
	packages, err := res.Resolve(pkgJson)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Map packages for quick lookup
	pkgMap := make(map[string]*resolver.Package)
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	fmt.Printf("\n📦 %s\n", pkgJson.Name)

	// Traverse
	var printDeps func(deps map[string]string, prefix string, visited map[string]bool)
	printDeps = func(deps map[string]string, prefix string, visited map[string]bool) {
		i := 0
		count := len(deps)

		// Filter out platform packages from count to properly draw connectors
		filteredDeps := make([]string, 0, count)
		for name := range deps {
			if name == "php" || strings.HasPrefix(name, "ext-") || strings.HasSuffix(name, "-implementation") {
				continue
			}
			filteredDeps = append(filteredDeps, name)
		}
		count = len(filteredDeps)

		for _, name := range filteredDeps {
			constraint := deps[name]

			isLast := i == count-1
			connector := "├──"
			if isLast {
				connector = "└──"
			}

			// Get resolved version
			version := constraint
			var subDeps map[string]string
			if pkg, ok := pkgMap[name]; ok {
				version = pkg.Version
				subDeps = pkg.Require
			}

			fmt.Printf("%s%s %s (%s)\n", prefix, connector, name, version)

			if len(subDeps) > 0 {
				// Avoid cycles
				if !visited[name] {
					newVisited := make(map[string]bool)
					for k, v := range visited {
						newVisited[k] = v
					}
					newVisited[name] = true

					newPrefix := prefix + "│   "
					if isLast {
						newPrefix = prefix + "    "
					}
					printDeps(subDeps, newPrefix, newVisited)
				}
			}
			i++
		}
	}

	printDeps(pkgJson.Require, "", make(map[string]bool))

	return nil
}
