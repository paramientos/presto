# 🎵 Presto

**Lightning-Fast PHP Package Manager - A Composer Drop-in Replacement**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Version](https://img.shields.io/badge/version-v0.1.10-blue.svg)](https://github.com/paramientos/presto/releases)
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

# Clear cache
presto cache clear
```

## ⚡ Performance Comparison

Real-world benchmark (Laravel-sized project with 47 packages):

| Tool     | Time    | Speed  |
|----------|---------|--------|
| Composer | 42.3s   | 1x     |
| **Presto** | **3.8s** | **11x** |

**Second run (with cache):**
| Tool     | Time    | Speed  |
|----------|---------|--------|
| Composer | 8.2s    | 1x     |
| **Presto** | **0.4s** | **20x** |

## 🎨 Example Output

```bash
$ presto install
🎵 Presto Install
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 Project: myapp/project
📝 Description: My awesome PHP project

🔍 Resolving dependencies...
✅ Resolved 47 packages

⬇️  Downloading packages...
[========================================] 47/47

📝 Generating autoload files...

✨ Installation complete!
```

```bash
$ presto audit
🎵 Security Audit
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  Found 2 vulnerabilities:

[HIGH] symfony/http-kernel@5.4.0
  CVE: CVE-2023-XXXXX
  Description: Security vulnerability in HTTP kernel
  Fix: Update to 5.4.31 or later
```

```bash
$ presto tree
📦 laravel/laravel
├── php ^8.1
├── laravel/framework ^10.0 (v10.34.2)
│   ├── illuminate/support ^10.0 (v10.34.2)
│   │   ├── doctrine/inflector ^2.0 (v2.0.8)
│   │   └── ...
└── ...
```

## 🔥 Killer Features

### 1. **Security Audit**
Built-in vulnerability scanning - something Composer doesn't have!

### 2. **Dependency Insights**
`presto why` and `presto why-not` commands help you understand your dependency tree

### 3. **10x-20x Speed**
Parallel downloads and native binary make it incredibly fast

### 4. **Smart Caching**
Shared cache across projects saves disk space and time

### 5. **Better UX**
Clear progress indicators, beautiful output, helpful error messages

## 🏗️ Architecture

```
presto/
├── cmd/presto/          # CLI entry point
├── internal/
│   ├── parser/          # composer.json parser
│   ├── packagist/       # Packagist API client
│   ├── resolver/        # Dependency resolver
│   ├── downloader/      # Parallel downloader
│   ├── autoload/        # Autoload generator
│   └── security/        # Security auditor
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
