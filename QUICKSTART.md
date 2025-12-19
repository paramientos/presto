# 🚀 Presto Quick Start Guide

Get up and running with Presto in 5 minutes!

## Installation

### Option 1: Download Binary (Fastest)

**macOS/Linux:**
```bash
# Download latest release
curl -L https://github.com/aras/presto/releases/latest/download/presto-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o presto

# Make executable
chmod +x presto

# Move to PATH
sudo mv presto /usr/local/bin/

# Verify
presto --version
```

**Windows:**
Download from [Releases](https://github.com/aras/presto/releases) and add to PATH.

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/aras/presto.git
cd presto

# Install dependencies
make deps

# Build
make build

# Install (optional)
sudo make install

# Verify
presto --version
```

## Basic Usage

### 1. Initialize a New Project

```bash
# Create new project
mkdir my-php-project
cd my-php-project

# Initialize composer.json
presto init
```

### 2. Install Dependencies

```bash
# Install from existing composer.json
presto install
```

**Output:**
```
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

### 3. Add Packages

```bash
# Add a package
presto require symfony/console

# Add multiple packages
presto require guzzlehttp/guzzle monolog/monolog

# Add dev dependency
presto require --dev phpunit/phpunit
```

### 4. Update Dependencies

```bash
# Update all packages
presto update

# Update specific package
presto update symfony/console
```

### 5. Remove Packages

```bash
# Remove a package
presto remove vendor/package
```

## Killer Features

### 🔒 Security Audit

Scan your project for vulnerabilities:

```bash
presto audit
```

**Output:**
```
🎵 Security Audit
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  Found 2 vulnerabilities:

[HIGH] symfony/http-kernel@5.4.0
  CVE: CVE-2023-XXXXX
  Description: Security vulnerability in HTTP kernel
  Fix: Update to 5.4.31 or later

[MEDIUM] guzzlehttp/guzzle@7.0.1
  CVE: CVE-2023-YYYYY
  Description: SSRF vulnerability
  Fix: Update to 7.5.0 or later
```

### 🔍 Dependency Insights

**Why is a package installed?**

```bash
presto why psr/log
```

**Output:**
```
🎵 Why is psr/log installed?
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📊 Dependency chain:
Your project
  └─ symfony/console (^6.0)
      └─ psr/log (^3.0)
```

**Why can't I install a version?**

```bash
presto why-not doctrine/orm 3.0
```

**Output:**
```
🎵 Why can't doctrine/orm@3.0 be installed?
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

❌ Conflicts found:

  • Requires PHP ^8.2 (you have 8.1)
  • symfony/http-kernel requires ^6.0

💡 To install:
  1. Update PHP to 8.2
  2. Update conflicting packages
```

### 📊 Show Installed Packages

```bash
presto show
```

**Output:**
```
🎵 Installed Packages
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📦 Production Dependencies:
  • symfony/console: ^6.0
  • guzzlehttp/guzzle: ^7.0
  • monolog/monolog: ^3.0

🔧 Development Dependencies:
  • phpunit/phpunit: ^10.0
```

## Performance Comparison

**Laravel-sized project (47 packages):**

| Command | Composer | Presto | Speedup |
|---------|----------|--------|---------|
| First install | 42.3s | 3.8s | **11x faster** |
| Cached install | 8.2s | 0.4s | **20x faster** |

## Common Workflows

### Starting a New Laravel Project

```bash
# Create project
mkdir my-laravel-app
cd my-laravel-app

# Initialize
presto init

# Add Laravel
presto require laravel/framework

# Install
presto install
```

### Migrating from Composer

Presto is a drop-in replacement - no migration needed!

```bash
# Just use presto instead of composer
presto install  # instead of: composer install
presto require symfony/console  # instead of: composer require
```

Your existing `composer.json` and `composer.lock` work as-is!

## Cache Management

```bash
# Clear cache
presto cache clear

# Cache is automatically managed
# Shared across projects for space efficiency
```

## Tips & Tricks

### 1. **Faster CI/CD**

Replace `composer install` with `presto install` in your CI:

```yaml
# .github/workflows/ci.yml
- name: Install dependencies
  run: |
    curl -L https://github.com/aras/presto/releases/latest/download/presto-linux-amd64 -o presto
    chmod +x presto
    ./presto install
```

### 2. **Alias for Convenience**

```bash
# Add to ~/.bashrc or ~/.zshrc
alias composer='presto'
```

Now `composer install` actually runs Presto!

### 3. **Check Before Update**

```bash
# See what would be updated
presto show

# Check for security issues
presto audit

# Then update
presto update
```

## Troubleshooting

### Package Not Found

```bash
# Make sure package name is correct
presto require vendor/package-name

# Search on packagist.org first
```

### Permission Denied

```bash
# Use sudo for global install
sudo presto global require package/name

# Or install locally (recommended)
presto require package/name
```

### Slow Downloads

```bash
# Clear cache and retry
presto cache clear
presto install
```

## Next Steps

- Read the [full documentation](https://github.com/aras/presto)
- Check out [examples](https://github.com/aras/presto/tree/main/examples)
- Join the [community discussions](https://github.com/aras/presto/discussions)
- Report issues on [GitHub](https://github.com/aras/presto/issues)

## Getting Help

- 📖 [Documentation](https://github.com/aras/presto)
- 💬 [Discussions](https://github.com/aras/presto/discussions)
- 🐛 [Issue Tracker](https://github.com/aras/presto/issues)
- 📧 Email: presto@example.com

---

**Happy coding with Presto! 🎵⚡**
