package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aras/presto/internal/autoload"
	"github.com/aras/presto/internal/cache"
	"github.com/aras/presto/internal/downloader"
	"github.com/aras/presto/internal/lockfile"
	"github.com/aras/presto/internal/packagist"
	"github.com/aras/presto/internal/parser"
	"github.com/aras/presto/internal/resolver"
	"github.com/aras/presto/internal/scripts"
	"github.com/aras/presto/internal/security"
	"github.com/aras/presto/internal/trust"
	"github.com/aras/presto/internal/ui"
	"github.com/spf13/cobra"
)

var version = "0.1.12"

var (
	verbose     bool
	noScripts   bool
	trustScript bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "presto",
		Short: "A blazing fast package manager for PHP",
		Long:  `Presto is a high-performance, drop-in replacement for Composer with killer features.`,
	}

	rootCmd.Version = version
	rootCmd.SetVersionTemplate("presto {{.Version}}\n")

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noScripts, "no-scripts", false, "never run scripts defined by the project")
	rootCmd.PersistentFlags().BoolVar(&trustScript, "trust-scripts", false, "run the project's scripts without asking")

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install dependencies from composer.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(false)
		},
	}

	requireCmd := &cobra.Command{
		Use:   "require [packages...]",
		Short: "Add new packages to composer.json",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRequire(args)
		},
	}

	updateCmd := &cobra.Command{
		Use:   "update [packages...]",
		Short: "Update dependencies to latest versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(args)
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove [packages...]",
		Short: "Remove packages from composer.json",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args)
		},
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show installed packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow()
		},
	}

	auditCmd := &cobra.Command{
		Use:   "audit",
		Short: "Scan for security vulnerabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit()
		},
	}

	whyCmd := &cobra.Command{
		Use:   "why [package]",
		Short: "Show why a package is installed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhy(args[0])
		},
	}

	whyNotCmd := &cobra.Command{
		Use:   "why-not [package] [version]",
		Short: "Show why a package version cannot be installed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhyNot(args[0], args[1])
		},
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new composer.json file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}

	var strictValidate bool
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Checks if composer.json is valid",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(strictValidate)
		},
	}
	validateCmd.Flags().BoolVar(&strictValidate, "strict", false, "Failure on warnings")

	treeCmd := &cobra.Command{
		Use:     "tree",
		Short:   "Show dependency tree",
		Aliases: []string{"map"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTree()
		},
	}

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

	trustCmd := &cobra.Command{
		Use:   "trust",
		Short: "Allow this project's scripts to run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTrust()
		},
	}

	trustListCmd := &cobra.Command{
		Use:   "list",
		Short: "List trusted projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTrustList()
		},
	}

	trustRevokeCmd := &cobra.Command{
		Use:   "revoke [path]",
		Short: "Withdraw trust from a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runTrustRevoke(path)
		},
	}

	trustCmd.AddCommand(trustListCmd, trustRevokeCmd)

	runScriptCmd := &cobra.Command{
		Use:     "run-script [script] [-- args...]",
		Short:   "Run scripts defined in composer.json",
		Aliases: []string{"run"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptName := args[0]
			// Strip a leading "--" separator and pass the rest as script arguments.
			// Supports both: presto run script arg1  and  presto run script -- arg1
			scriptArgs := args[1:]
			if len(scriptArgs) > 0 && scriptArgs[0] == "--" {
				scriptArgs = scriptArgs[1:]
			}
			return runScript(scriptName, scriptArgs...)
		},
	}

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
		validateCmd,
		cacheCmd,
		trustCmd,
		runScriptCmd,
	)

	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		// If cobra doesn't recognise the command, try running it as a composer script.
		if strings.HasPrefix(err.Error(), "unknown command") && len(os.Args) > 1 {
			scriptName := os.Args[1]
			scriptArgs := os.Args[2:]
			if len(scriptArgs) > 0 && scriptArgs[0] == "--" {
				scriptArgs = scriptArgs[1:]
			}
			if scriptErr := runScript(scriptName, scriptArgs...); scriptErr != nil {
				// Script not found — surface the original unknown-command error.
				if strings.Contains(scriptErr.Error(), "script not found") {
					ui.Fail("%v", err)
				} else {
					ui.Fail("%v", scriptErr)
				}
				os.Exit(1)
			}
			return
		}

		ui.Fail("%v", err)
		os.Exit(1)
	}
}

func logVerbose(format string, args ...interface{}) {
	if verbose {
		ui.Note(format, args...)
	}
}

func trustMode() trust.Mode {
	if noScripts {
		return trust.ModeNever
	}

	if trustScript || truthy(os.Getenv("PRESTO_TRUST_SCRIPTS")) {
		return trust.ModeAlways
	}

	return trust.ModeAsk
}

// downloadWorkers is tuned for latency, not bandwidth: every archive costs a
// redirect plus a fetch, so the wall clock is the number of waves, not the bytes.
func downloadWorkers() int {
	const defaultWorkers = 24

	n, err := strconv.Atoi(os.Getenv("PRESTO_DOWNLOAD_WORKERS"))
	if err != nil || n < 1 {
		return defaultWorkers
	}

	return n
}

func truthy(value string) bool {
	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true
	}

	return false
}

func runInstall(forceResolve bool) error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return fmt.Errorf("failed to parse composer.json: %w", err)
	}

	lifecycle := scripts.NewLifecycle(scripts.NewRunner(verbose), composer, trustMode())

	if forceResolve {
		lifecycle.Run("pre-update-cmd")
	} else {
		lifecycle.Run("pre-install-cmd")
	}

	client := packagist.NewClient()
	res := resolver.NewResolver(client)
	res.Log(logVerbose)

	var packages []*resolver.Package

	resolveStart := time.Now()

	if !forceResolve {
		packages, err = packagesFromLock(client, composer, res)
		if err != nil {
			return err
		}
	}

	if len(packages) == 0 {
		spin := ui.StartSpinner("Resolving dependencies")
		res.OnPackage(spin.Detail)

		packages, err = res.Resolve(composer)
		spin.Stop()

		if err != nil {
			return fmt.Errorf("dependency resolution failed: %w", err)
		}
	}

	ui.Result("Resolved", len(packages), time.Since(resolveStart))

	for _, pkg := range packages {
		logVerbose("  %s (%s) -> %s", pkg.Name, pkg.Version, pkg.URL)
	}

	installStart := time.Now()

	spin := ui.StartSpinner("Downloading packages")

	dl := downloader.NewDownloader(downloadWorkers())
	installed, err := dl.DownloadAll(packages, func(done, total int, name string) {
		spin.Detail(fmt.Sprintf("%d/%d %s", done, total, name))
	})
	spin.Stop()

	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	readLocalAutoload(packages)

	if len(installed) == 0 {
		ui.Result("Audited", len(packages), time.Since(installStart))
	} else {
		ui.Result("Installed", len(installed), time.Since(installStart))

		for _, pkg := range installed {
			ui.Added(pkg.Name, pkg.Version)
		}
	}

	lifecycle.Run("pre-autoload-dump")

	spin = ui.StartSpinner("Generating autoload files")
	err = autoload.NewGenerator().Generate(composer, packages)
	spin.Stop()

	if err != nil {
		return fmt.Errorf("autoload generation failed: %w", err)
	}

	lifecycle.Run("post-autoload-dump")

	spin = ui.StartSpinner("Writing composer.lock")
	err = lockfile.NewGeneratorWithClient(client).Generate(composer, packages)
	spin.Stop()

	if err != nil {
		return fmt.Errorf("lock file generation failed: %w", err)
	}

	lifecycle.Run("post-root-package-install")

	if forceResolve {
		lifecycle.Run("post-update-cmd")
	} else {
		lifecycle.Run("post-install-cmd")
	}

	lifecycle.Report()

	return nil
}

func packagesFromLock(client *packagist.Client, composer *parser.ComposerJSON, res *resolver.Resolver) ([]*resolver.Package, error) {
	if _, err := os.Stat("composer.lock"); err != nil {
		return nil, nil
	}

	lock, err := parser.ParseComposerLock("composer.lock")
	if err != nil {
		ui.Warn("composer.lock is unreadable (%v), resolving from composer.json", err)
		return nil, nil
	}

	if lock.ContentHash != lockfile.NewGeneratorWithClient(client).GenerateContentHash(composer) {
		ui.Warn("composer.lock is out of date with composer.json, resolving again")
		return nil, nil
	}

	packages, err := res.ResolveFromLock(lock)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve from lock file: %w", err)
	}

	return packages, nil
}

func readLocalAutoload(packages []*resolver.Package) {
	for _, pkg := range packages {
		content, err := os.ReadFile(filepath.Join("vendor", pkg.Name, "composer.json"))
		if err != nil {
			logVerbose("no composer.json for %s: %v", pkg.Name, err)
			continue
		}

		var manifest struct {
			Autoload json.RawMessage `json:"autoload"`
		}

		if err := json.Unmarshal(content, &manifest); err == nil && len(manifest.Autoload) > 0 {
			pkg.Autoload = manifest.Autoload
		}
	}
}

func runRequire(packages []string) error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	client := packagist.NewClient()

	for _, pkg := range packages {
		spin := ui.StartSpinner("Looking up " + pkg)
		info, err := client.GetPackage(pkg)
		spin.Stop()

		if err != nil {
			return fmt.Errorf("package %s not found: %w", pkg, err)
		}

		if composer.Require == nil {
			composer.Require = make(map[string]string)
		}

		constraint := packagist.RecommendedConstraint(info.LatestVersion)
		composer.Require[pkg] = constraint

		ui.Added(pkg, constraint)
	}

	if err := parser.WriteComposerJSON("composer.json", composer); err != nil {
		return err
	}

	return runInstall(true)
}

func runUpdate(packages []string) error {
	if len(packages) > 0 {
		ui.Note("updating %s", strings.Join(packages, ", "))
	}

	return runInstall(true)
}

func runRemove(packages []string) error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		version, ok := composer.Require[pkg]
		if !ok {
			version = composer.RequireDev[pkg]
		}

		delete(composer.Require, pkg)
		delete(composer.RequireDev, pkg)

		ui.Removed(pkg, version)
	}

	return parser.WriteComposerJSON("composer.json", composer)
}

func runShow() error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	printDependencies("Production", composer.Require)

	if len(composer.RequireDev) > 0 {
		ui.Print("")
		printDependencies("Development", composer.RequireDev)
	}

	return nil
}

func printDependencies(heading string, dependencies map[string]string) {
	ui.Heading(heading)

	names := make([]string, 0, len(dependencies))
	width := 0

	for name := range dependencies {
		names = append(names, name)
		if len(name) > width {
			width = len(name)
		}
	}

	sort.Strings(names)

	for _, name := range names {
		ui.Print("  %-*s  %s", width, name, ui.Dim(dependencies[name]))
	}
}

func runAudit() error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	spin := ui.StartSpinner("Auditing dependencies")
	vulnerabilities, err := security.NewAuditor().ScanProject(composer)
	spin.Stop()

	if err != nil {
		return err
	}

	if len(vulnerabilities) == 0 {
		ui.Status("No known vulnerabilities")
		return nil
	}

	ui.Warn("found %d vulnerabilities", len(vulnerabilities))

	for _, vuln := range vulnerabilities {
		ui.Print("")
		ui.Print("%s %s %s", ui.Bold(vuln.Severity), vuln.Package, ui.Dim(vuln.Version))
		ui.Print("  %s", vuln.CVE)
		ui.Print("  %s", vuln.Description)
		ui.Print("  fix: %s", vuln.Fix)
	}

	return nil
}

func runWhy(packageName string) error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	if version, ok := composer.Require[packageName]; ok {
		ui.Print("%s %s is required by composer.json", packageName, ui.Dim(version))
		return nil
	}

	client := packagist.NewClient()
	res := resolver.NewResolver(client)
	res.Log(logVerbose)

	spin := ui.StartSpinner("Building dependency tree")
	res.OnPackage(spin.Detail)

	tree, err := res.BuildDependencyTree(composer, packageName)
	spin.Stop()

	if err != nil {
		return fmt.Errorf("not found in dependency tree: %w", err)
	}

	ui.Print("%s", tree)

	return nil
}

func runWhyNot(packageName, version string) error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	client := packagist.NewClient()
	res := resolver.NewResolver(client)
	res.Log(logVerbose)

	spin := ui.StartSpinner("Checking conflicts")
	conflicts, err := res.CheckConflicts(composer, packageName, version)
	spin.Stop()

	if err != nil {
		return err
	}

	if len(conflicts) == 0 {
		ui.Status("%s %s can be installed", packageName, ui.Dim(version))
		return nil
	}

	ui.Print("%s %s conflicts with:", packageName, ui.Dim(version))

	for _, conflict := range conflicts {
		ui.Print("  %s", conflict)
	}

	return nil
}

func runInit() error {
	composer := &parser.ComposerJSON{
		Name:        "vendor/project",
		Description: "A new PHP project",
		Type:        "project",
		License:     "MIT",
		Require: map[string]string{
			"php": "^8.1",
		},
		Autoload: parser.AutoloadConfig{
			PSR4: map[string]interface{}{
				"App\\": "src/",
			},
		},
	}

	if err := parser.WriteComposerJSON("composer.json", composer); err != nil {
		return err
	}

	ui.Status("Created composer.json")

	return nil
}

func runCacheClear() error {
	dir, err := cache.Clear()
	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	ui.Status("Cleared %s", dir)

	return nil
}

func runTrust() error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return err
	}

	pending := scripts.Pending(composer)
	if len(pending) == 0 {
		ui.Status("This project defines no scripts")
		return nil
	}

	store, err := trust.Load()
	if err != nil {
		return err
	}

	trust.Describe(pending)
	ui.Blank()

	if err := store.Allow(".", trust.Hash(pending)); err != nil {
		return err
	}

	ui.Status("Trusted %s", trust.Resolve("."))
	ui.Note("editing these scripts asks again")

	return nil
}

func runTrustList() error {
	store, err := trust.Load()
	if err != nil {
		return err
	}

	entries := store.Entries()
	if len(entries) == 0 {
		ui.Status("No trusted projects")
		return nil
	}

	for _, entry := range entries {
		ui.Print("%s  %s", entry.Path, ui.Dim(entry.TrustedAt.Format(time.DateOnly)))
	}

	return nil
}

func runTrustRevoke(path string) error {
	store, err := trust.Load()
	if err != nil {
		return err
	}

	revoked, err := store.Revoke(path)
	if err != nil {
		return err
	}

	if !revoked {
		ui.Status("%s was not trusted", trust.Resolve(path))
		return nil
	}

	ui.Status("Revoked %s", trust.Resolve(path))

	return nil
}

func runTree() error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return fmt.Errorf("failed to parse composer.json: %w", err)
	}

	client := packagist.NewClient()
	res := resolver.NewResolver(client)
	res.Log(logVerbose)

	spin := ui.StartSpinner("Resolving dependencies")
	res.OnPackage(spin.Detail)

	packages, err := res.Resolve(composer)
	spin.Stop()

	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	pkgMap := make(map[string]*resolver.Package, len(packages))
	for _, pkg := range packages {
		pkgMap[pkg.Name] = pkg
	}

	ui.Print("%s", ui.Bold(composer.Name))

	printTree(composer.Require, pkgMap, "", make(map[string]bool))

	return nil
}

func printTree(deps map[string]string, pkgMap map[string]*resolver.Package, prefix string, visited map[string]bool) {
	names := make([]string, 0, len(deps))

	for name := range deps {
		if name == "php" || strings.HasPrefix(name, "ext-") || strings.HasSuffix(name, "-implementation") {
			continue
		}

		names = append(names, name)
	}

	sort.Strings(names)

	for i, name := range names {
		isLast := i == len(names)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		version := deps[name]

		var subDeps map[string]string
		if pkg, ok := pkgMap[name]; ok {
			version = pkg.Version
			subDeps = pkg.Require
		}

		ui.Print("%s%s%s %s", prefix, connector, name, ui.Dim(version))

		if len(subDeps) == 0 || visited[name] {
			continue
		}

		seen := make(map[string]bool, len(visited)+1)
		for k, v := range visited {
			seen[k] = v
		}
		seen[name] = true

		childPrefix := prefix + "│   "
		if isLast {
			childPrefix = prefix + "    "
		}

		printTree(subDeps, pkgMap, childPrefix, seen)
	}
}

func runValidate(strict bool) error {
	path := "composer.json"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("composer.json not found in current directory")
	}

	composer, err := parser.ParseComposerJSON(path)
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	result := parser.Validate(composer)

	for _, warning := range result.Warnings {
		ui.Warn("%s", warning)
	}

	for _, failure := range result.Errors {
		ui.Fail("%s", failure)
	}

	if !result.IsValid(strict) {
		if len(result.Errors) > 0 {
			return fmt.Errorf("composer.json has %d errors", len(result.Errors))
		}

		return fmt.Errorf("composer.json has %d warnings", len(result.Warnings))
	}

	ui.Status("composer.json is valid")

	return nil
}

func runScript(scriptName string, scriptArgs ...string) error {
	composer, err := parser.ParseComposerJSON("composer.json")
	if err != nil {
		return fmt.Errorf("failed to parse composer.json: %w", err)
	}

	if composer.Scripts == nil {
		return fmt.Errorf("script not found: %q (no scripts defined in composer.json)", scriptName)
	}

	if _, ok := composer.Scripts[scriptName]; !ok {
		return fmt.Errorf("script not found: %q", scriptName)
	}

	return scripts.NewRunner(verbose).Run(scriptName, composer, scriptArgs...)
}
