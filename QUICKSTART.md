# 🚀 Presto Quick Start Guide

Get up and running with Presto in 5 minutes!

## Installation

### Option 1: Download Binary (Fastest)

**macOS/Linux:**
```bash
# Download latest release
curl -L https://github.com/paramientos/presto/releases/latest/download/presto-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o presto

# Make executable
chmod +x presto

# Move to PATH
sudo mv presto /usr/local/bin/

# Verify
presto --version
```

**Windows:**
Download from [Releases](https://github.com/paramientos/presto/releases) and add to PATH.

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/paramientos/presto.git
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
Resolved 47 packages in 1.24s
Installed 47 packages in 3.51s
 + doctrine/inflector 2.0.8
 + laravel/framework v10.34.2
 + symfony/console v6.4.2
```

A spinner runs while each phase works, then the phase collapses to one line. When
there is nothing new to fetch the second line reads `Audited 47 packages in 38ms`.

The first install of a project that defines scripts stops to ask:

```
warning: this project defines scripts that presto would run

  post-install-cmd
    @php artisan package:discover --ansi

? Run these scripts?
> [o] once    run them for this install only
  [a] always  trust this project from now on
  [n] never   skip them
```

Answer `always` and presto remembers the project. Skipped scripts are named at the
end of the install so nothing goes missing in silence. See `presto trust --help`.

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
warning: found 2 vulnerabilities

HIGH symfony/http-kernel 5.4.0
  CVE-2023-XXXXX
  Security vulnerability in HTTP kernel
  fix: Update to 5.4.31 or later

MEDIUM guzzlehttp/guzzle 7.0.1
  CVE-2023-YYYYY
  SSRF vulnerability
  fix: Update to 7.5.0 or later
```

### 🔍 Dependency Insights

**Why is a package installed?**

```bash
presto why psr/log
```

**Output:**
```
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
doctrine/orm 3.0 conflicts with:
  Requires PHP ^8.2 (you have 8.1)
  symfony/http-kernel requires ^6.0
```

### 📊 Show Installed Packages

```bash
presto show
```

**Output:**
```
Production
  guzzlehttp/guzzle  ^7.0
  monolog/monolog    ^3.0
  symfony/console    ^6.0

Development
  phpunit/phpunit  ^10.0
```

## Performance Comparison

**Laravel 10 project (95 packages), same machine, `vendor/` deleted before each run:**

| Run | Composer 2.10 | Presto |
|-----|---------------|--------|
| Warm cache | 4.62s | **0.48s** |
| Cold cache | | 5.3s |

Presto caches manifests and archives in `~/.cache/presto`, shared across projects.
Run `presto cache clear` to drop it.

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
    curl -L https://github.com/paramientos/presto/releases/latest/download/presto-linux-amd64 -o presto
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

- Read the [full documentation](https://github.com/paramientos/presto)
- Check out [examples](https://github.com/paramientos/presto/tree/main/examples)
- Join the [community discussions](https://github.com/paramientos/presto/discussions)
- Report issues on [GitHub](https://github.com/paramientos/presto/issues)

## Getting Help

- 📖 [Documentation](https://github.com/paramientos/presto)
- 💬 [Discussions](https://github.com/paramientos/presto/discussions)
- 🐛 [Issue Tracker](https://github.com/paramientos/presto/issues)
- 📧 Email: presto@example.com

---

**Happy coding with Presto! 🎵⚡**
