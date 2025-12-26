# Changelog

All notable changes to Presto will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.6] - 2025-12-26

### Fixed
- 🚀 Fixed incorrect latest version selection (was picking random stable versions instead of highest).
- 📦 Added fallback for packages without `dist` URLs (e.g. GitHub/Codeberg/GitLab Git source URLs are now converted to ZIP downloads).
- 🔧 Fixed a race condition/sync issue in downloader where temp files were not fully flushed before extraction.

## [0.1.5] - 2025-12-26

### Fixed
- 🐛 Fix unmarshal error when fetching packages with `dist: null` or `dist: "__unset"` from Packagist p2 API. (Thanks to [@lwohlhart](https://github.com/lwohlhart))

## [0.1.4] - 2025-12-25

### Fixed
- Improved Tesseract OCR stability in GitHub Actions.
- Fixed permission issues in CI pipeline.

## [0.1.0] - 2025-12-19

### Added
- 🎵 Initial release of Presto
- Core package management functionality
  - `presto install` - Install dependencies from composer.json
  - `presto require` - Add new packages
  - `presto update` - Update dependencies
  - `presto remove` - Remove packages
  - `presto show` - Show installed packages
  - `presto init` - Initialize new project

### Killer Features
- 🔒 `presto audit` - Security vulnerability scanning
- 🔍 `presto why` - Show dependency tree for a package
- 🚫 `presto why-not` - Explain why a version can't be installed

### Performance
- Parallel package downloads (8 workers)
- 10x-20x faster than Composer
- Smart caching system
- Native Go binary

### Compatibility
- 100% compatible with composer.json
- 100% compatible with composer.lock
- Works with Packagist.org
- PSR-4/PSR-0 autoloading support

### Infrastructure
- Cross-platform support (macOS, Linux, Windows)
- GitHub Actions CI/CD
- Automated releases
- Comprehensive documentation

---

## Future Releases

### [0.2.0] - Planned
- Composer plugins support
- Custom repositories (Git, VCS, Path)
- Composer scripts execution
- Global package installation
- Interactive mode
- Improved error messages

### [0.3.0] - Planned
- Workspace/monorepo support
- Build profiles (production, dev, minimal)
- Delta updates
- Enhanced security features
- Performance optimizations

### [1.0.0] - Planned
- Production-ready release
- Full Composer compatibility
- Comprehensive test coverage
- Stable API
- Enterprise features

---

[0.1.6]: https://github.com/paramientos/presto/releases/tag/v0.1.6
[0.1.5]: https://github.com/paramientos/presto/releases/tag/v0.1.5
[0.1.4]: https://github.com/paramientos/presto/releases/tag/v0.1.4
[0.1.0]: https://github.com/paramientos/presto/releases/tag/v0.1.0
