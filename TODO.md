# ✅ Presto TODO List

## 🔥 Immediate (This Week)

### Must Have
- [ ] **composer.lock Integration**
  - [ ] Generate lock file after install
  - [ ] Read existing lock file
  - [ ] Update lock file on require/remove
  - [ ] Validate content hash

- [ ] **GitHub Repository Setup**
  - [ ] Push to GitHub
  - [ ] Add repository description
  - [ ] Add topics/tags
  - [ ] Enable GitHub Pages for docs

- [ ] **First Release**
  - [ ] Create v0.1.0 tag
  - [ ] Build binaries for all platforms
  - [ ] Create GitHub release
  - [ ] Write release notes

### Should Have
- [ ] **Documentation**
  - [ ] Add installation instructions
  - [ ] Add usage examples with GIFs
  - [ ] Create comparison table with Composer
  - [ ] Add troubleshooting section

- [ ] **Testing**
  - [ ] Add parser tests
  - [ ] Add packagist client tests
  - [ ] Add downloader tests
  - [ ] Run tests in CI

---

## 📅 Next Week

### Features
- [ ] **Performance Benchmarks**
  - [ ] Run benchmark.sh script
  - [ ] Document results in README
  - [ ] Create performance comparison graphs
  - [ ] Add to marketing materials

- [ ] **Error Handling**
  - [ ] Better network error messages
  - [ ] Retry logic for failed downloads
  - [ ] Timeout configuration
  - [ ] Offline mode detection

- [ ] **Logging**
  - [ ] Add verbose mode (-v, -vv, -vvv)
  - [ ] Log file support
  - [ ] Debug mode
  - [ ] Quiet mode (-q)

### Quality
- [ ] **Code Quality**
  - [ ] Add golangci-lint config
  - [ ] Fix all linter warnings
  - [ ] Add code comments
  - [ ] Refactor long functions

- [ ] **CI/CD**
  - [ ] Add test coverage reporting
  - [ ] Add code quality badges
  - [ ] Add automated releases
  - [ ] Add dependency updates (Dependabot)

---

## 🎯 This Month

### Core Features
- [ ] **Composer Scripts**
  - [ ] Parse scripts from composer.json
  - [ ] Execute pre/post install hooks
  - [ ] Support custom scripts
  - [ ] Handle script failures

- [ ] **Update Command**
  - [ ] Update all packages
  - [ ] Update specific packages
  - [ ] Update with constraints
  - [ ] Dry-run mode

- [ ] **Remove Command**
  - [ ] Remove package from composer.json
  - [ ] Update lock file
  - [ ] Remove from vendor
  - [ ] Remove unused dependencies

### Developer Experience
- [ ] **Better Output**
  - [ ] Colored output
  - [ ] Emoji support toggle
  - [ ] JSON output mode
  - [ ] Machine-readable output

- [ ] **Configuration**
  - [ ] presto.json config file
  - [ ] Global config
  - [ ] Project config
  - [ ] Environment variables

---

## 🚀 Next Month

### Advanced Features
- [ ] **Custom Repositories**
  - [ ] Git repository support
  - [ ] Path repository support
  - [ ] VCS repository support
  - [ ] Private Packagist support

- [ ] **Global Packages**
  - [ ] `presto global require`
  - [ ] `presto global remove`
  - [ ] `presto global update`
  - [ ] Global bin directory

- [ ] **Workspace Support**
  - [ ] Monorepo detection
  - [ ] Workspace configuration
  - [ ] Symlink local packages
  - [ ] Workspace commands

### Security
- [ ] **Enhanced Audit**
  - [ ] Multiple CVE sources
  - [ ] Severity filtering
  - [ ] Ignore list
  - [ ] Auto-fix suggestions

- [ ] **Package Verification**
  - [ ] Checksum verification
  - [ ] Signature verification
  - [ ] Trust system
  - [ ] Security policies

---

## 💡 Ideas / Backlog

### Nice to Have
- [ ] Shell completion (bash, zsh, fish)
- [ ] Man pages
- [ ] Homebrew formula
- [ ] Docker image
- [ ] Snap package
- [ ] Chocolatey package (Windows)
- [ ] AUR package (Arch Linux)

### Integrations
- [ ] VS Code extension
- [ ] PHPStorm plugin
- [ ] GitHub Actions
- [ ] GitLab CI
- [ ] CircleCI
- [ ] Travis CI

### Community
- [ ] Contributing guide improvements
- [ ] Code of conduct
- [ ] Issue templates
- [ ] PR templates
- [ ] Discussion forum
- [ ] Discord server

---

## 🐛 Known Issues

### Bugs
- [ ] Some packages with empty URLs fail to download
  - **Status:** Partially fixed (meta-packages skip)
  - **TODO:** Better detection and error messages

- [ ] Version constraint edge cases
  - **Status:** Most cases work
  - **TODO:** Test more complex constraints

### Limitations
- [ ] No plugin support yet
- [ ] No script execution
- [ ] No custom repositories
- [ ] No global installation

---

## 📝 Notes

- Items marked with 🔥 are high priority
- Items marked with ⚡ are quick wins
- Items marked with 🎯 are goals for milestones

**Last Updated:** 2025-12-19
