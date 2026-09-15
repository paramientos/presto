# 🎵 Presto

**Lightning-Fast PHP Package Manager - A Composer Drop-in Replacement**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-v0.1.11-blue.svg)](https://github.com/paramientos/presto/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/paramientos/presto/actions)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

> ⚠️ **BETA SOFTWARE**: Presto is currently in **BETA**. While it is functional and fast, it may still have bugs or incomplete features. Use with caution in production environments.

> ⚡ **10x-20x faster** than Composer | 🔒 **Built-in security audit** | 🔍 **Dependency insights** | 💯 **100% compatible**

Presto is a blazing-fast, drop-in replacement for Composer written in Go. It's 100% compatible with `composer.json` and `composer.lock` while being **10x-20x faster** thanks to parallel downloads and native binary execution.

## 📥 Installation

### macOS / Linux
```bash
curl -fsSL https://raw.githubusercontent.com/paramientos/presto/main/scripts/install.sh | bash
```

### Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/paramientos/presto/main/scripts/install.ps1 | iex
```


## 📥 Or Manual Downloads

| Platform | Architecture | Download |
|----------|--------------|----------|
| **Windows** | x86_64 | [presto-windows-amd64.exe](https://github.com/paramientos/presto/releases/latest/download/presto-windows-amd64.exe) |
| **macOS** | Apple Silicon (M1/M2) | [presto-darwin-arm64](https://github.com/paramientos/presto/releases/latest/download/presto-darwin-arm64) |
| **macOS** | Intel | [presto-darwin-amd64](https://github.com/paramientos/presto/releases/latest/download/presto-darwin-amd64) |
| **Linux** | x86_64 | [presto-linux-amd64](https://github.com/paramientos/presto/releases/latest/download/presto-linux-amd64) |
| **Linux** | ARM64 | [presto-linux-arm64](https://github.com/paramientos/presto/releases/latest/download/presto-linux-arm64) |


## ✨ Features

### 🚀 **Blazing Fast**
- **10x-20x faster** than Composer
- Parallel package downloads (8 concurrent workers)
- Native binary (no PHP JIT overhead)
- Smart caching system

### 🔒 **Security First**
```bash
presto audit  # Scan for vulnerabilities
```
- Built-in CVE database scanning
- Real-time security alerts
- License compliance checking

### 🔍 **Dependency Insights**
```bash
presto why package/name           # Why is this installed?
presto why-not package/name 2.0   # Why can't I install this?
```
- Visual dependency trees
- Conflict resolution explanations
- Better than Composer!

### 💯 **100% Compatible**
- Drop-in replacement for Composer
- Reads `composer.json` and `composer.lock`
- Resolves to the same versions Composer does, verified against it
- Works with Packagist.org
- PSR-4/PSR-0 autoloading
- **Strict Validation** (v0.1.9+)
- **Composer Scripts** (Added in v0.1.10)

## 🛠️ Building

To build Presto from source:

```bash
git clone https://github.com/paramientos/presto.git
cd presto
make build
```

## 🎯 Usage

### Global Options

- `-v, --verbose`: Enable verbose output for debugging
- `--trust-scripts`: Run the project's scripts without asking
- `--no-scripts`: Never run the project's scripts
- `-h, --help`: Show help

### Commands

Presto uses the same commands as Composer:

```bash
# Install dependencies
presto install

# Add a package
presto require symfony/console

# Update packages
presto update

# Remove a package
presto remove vendor/package

# Show installed packages
presto show

# Show dependency tree (map)
presto tree

# Security audit (NEW!)
presto audit

# Dependency insights (NEW!)
presto why symfony/console
presto why-not doctrine/orm 3.0

# Initialize new project
presto init

# Validate composer.json (v0.1.9+)
presto validate
presto validate --strict

# Run custom scripts (v0.1.10+)
presto run post-install-cmd

# Allow this project's scripts to run
presto trust

# Clear cache
presto cache clear
```

## ⚡ Performance Comparison

A Laravel 10 project, 95 packages, same machine, `vendor/` and the lock file
deleted before each run. Both tools keep a warm package cache.

| Tool | Time |
|------|------|
| Composer 2.10 | 4.62s |
| **Presto** | **0.48s** |

With a cold cache, so every manifest and archive is fetched: **5.3s**.

Before this work the same project took 21.5s, because the resolver fetched one
manifest at a time and nothing was cached between runs. What changed:

- manifests are fetched breadth-first across 16 connections, not one at a time
- manifests are cached on disk and revalidated with `ETag`, so a repeat install
  sends no bytes
- archives are cached too, so wiping `vendor/` costs no network at all
- downloads run 24 at a time because each archive costs a redirect plus a fetch,
  while extraction is capped at 8 because writing thousands of small files is
  disk-bound and slows down when oversubscribed

Set `PRESTO_DOWNLOAD_WORKERS` to change the download count.

## 🎨 Example Output

A spinner runs while presto works, then each phase collapses to one line.

```bash
$ presto install
Resolved 47 packages in 1.24s
Installed 47 packages in 3.51s
 + doctrine/inflector 2.0.8
 + laravel/framework v10.34.2
 + symfony/console v6.4.2
```

Nothing to fetch reads as an audit:

```bash
$ presto install
Resolved 47 packages in 12ms
Audited 47 packages in 38ms
```

Progress goes to stderr, so `presto show > deps.txt` captures data and nothing else.

```bash
$ presto audit
warning: found 2 vulnerabilities

HIGH symfony/http-kernel 5.4.0
  CVE-2023-XXXXX
  Security vulnerability in HTTP kernel
  fix: Update to 5.4.31 or later
```

```bash
$ presto tree
laravel/laravel
├── laravel/framework v10.34.2
│   └── illuminate/support v10.34.2
│       └── doctrine/inflector v2.0.8
└── symfony/console v6.4.2
```

## 🔐 Script Trust

`composer.json` can ask for any command to run on your machine. Cloning a repository
and installing it should not be enough to run those commands, so presto asks first.

```bash
$ presto install
warning: this project defines scripts that presto would run

  post-install-cmd
    @php artisan package:discover --ansi

? Run these scripts?
> [o] once    run them for this install only
  [a] always  trust this project from now on
  [n] never   skip them
```

Answering `always` records the project in `~/.config/presto/trust.json`, keyed by the
commands themselves. Edit a script and presto asks again.

Without a terminal to ask, the scripts are skipped and named at the end:

```bash
$ presto install
Resolved 47 packages in 1.24s
Audited 47 packages in 38ms

warning: 1 script was not run: this project is not trusted
  post-install-cmd
  run `presto trust` to allow them
```

| | |
|---|---|
| `presto trust` | trust the current project |
| `presto trust list` | list trusted projects |
| `presto trust revoke [path]` | withdraw trust |
| `--trust-scripts` | run them without asking |
| `--no-scripts` | never run them |
| `PRESTO_TRUST_SCRIPTS=1` | same as `--trust-scripts`, for CI |

`presto run <script>` is never gated. You typed the name, so you meant it.

## 🧮 Resolution

Presto reads Composer's version semantics, not npm's, through
[shyim/go-version](https://github.com/shyim/go-version): `~1.0` means `>=1.0 <2.0`,
four-part versions like `9.18.1.10` are ordered properly, and stability ranks
`dev < alpha < beta < RC < stable`.

A package is chosen by the intersection of every constraint on it, not the last
one seen, and `conflict` blocks are honoured. When nothing can satisfy them all,
presto says who asked for what instead of installing something that breaks one of
them:

```
error: no released version of acme/lib satisfies every requirement:
  acme/high 1.0.0 requires >=1.6
  acme/low 1.0.0 requires <=1.4
```

Resolution is deterministic: the same `composer.json` gives the same
`composer.lock` every run.

On a Laravel 10 project (95 packages) and a smaller one (50 packages), presto and
Composer 2.10 resolve to the identical set of packages at identical versions.

## 💾 Cache

Presto caches package manifests and archives in `~/.cache/presto`, shared across
every project.

```bash
presto cache clear   # remove it
```

Set `PRESTO_CACHE_DIR` to move it, or `XDG_CACHE_HOME` to move it with everything
else. A manifest is reused for 15 minutes without asking packagist, then
revalidated with its `ETag`, which usually comes back as a `304` and no body. If
packagist cannot be reached at all, a cached manifest still answers.

## 🔥 Killer Features

### 1. **Security Audit**
Built-in vulnerability scanning - something Composer doesn't have!

### 2. **Dependency Insights**
`presto why` and `presto why-not` commands help you understand your dependency tree

### 3. **Composer's Own Version Semantics**
`~1.0` means what Composer says it means, constraints are intersected rather than
replaced, and `conflict` is honoured

### 4. **Shared Cache**
Manifests and archives are cached across projects, so the second install is local

### 5. **Script Trust**
A project's scripts run only once you have seen them and said yes

## 🏗️ Architecture

```
presto/
├── cmd/presto/          # CLI entry point
├── internal/
│   ├── parser/          # composer.json parser
│   ├── packagist/       # Packagist API client
│   ├── resolver/        # Dependency resolver and version solver
│   ├── downloader/      # Parallel downloader
│   ├── autoload/        # Autoload generator
│   ├── cache/           # Shared manifest and archive cache
│   ├── httpx/           # Tuned HTTP client
│   ├── lockfile/        # composer.lock writer
│   ├── scripts/         # Composer script runner
│   ├── security/        # Security auditor
│   ├── trust/           # Script trust store
│   └── ui/              # Terminal output
└── go.mod
```

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📝 License

MIT License - see [LICENSE](LICENSE) for details

## 🌟 Why Presto?

**Presto** (Italian: "quick, fast") - just like the musical term meaning "very fast", Presto executes your PHP dependency management at lightning speed! 🎵⚡

## 🔗 Links

- [GitHub](https://github.com/paramientos/presto)
- [Issue Tracker](https://github.com/paramientos/presto/issues)
- [Discussions](https://github.com/paramientos/presto/discussions)

---

Made with ❤️ by the Presto team
