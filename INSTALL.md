# 📥 Installing Presto

Multiple installation methods for different platforms and preferences.

---

## 🚀 Quick Install

| Platform | Architecture | Download |
|----------|--------------|----------|
| **Windows** | x86_64 | [presto-windows-amd64.exe](https://github.com/paramientos/presto/releases/latest/download/presto-windows-amd64.exe) |
| **macOS** | Apple Silicon (M1/M2) | [presto-darwin-arm64](https://github.com/paramientos/presto/releases/latest/download/presto-darwin-arm64) |
| **macOS** | Intel | [presto-darwin-amd64](https://github.com/paramientos/presto/releases/latest/download/presto-darwin-amd64) |
| **Linux** | x86_64 | [presto-linux-amd64](https://github.com/paramientos/presto/releases/latest/download/presto-linux-amd64) |
| **Linux** | ARM64 | [presto-linux-arm64](https://github.com/paramientos/presto/releases/latest/download/presto-linux-arm64) |

### macOS / Linux (One-liner)

```bash
curl -L https://github.com/paramientos/presto/releases/latest/download/presto-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/') -o presto && chmod +x presto && sudo mv presto /usr/local/bin/
```

### Windows (PowerShell)

```powershell
Invoke-WebRequest -Uri "https://github.com/paramientos/presto/releases/latest/download/presto-windows-amd64.exe" -OutFile "presto.exe"
Move-Item presto.exe C:\Windows\System32\
```


---

## 📦 Package Managers

### Homebrew (macOS/Linux)

```bash
# Coming soon!
brew tap paramientos/presto
brew install presto
```

### Snap (Linux)

```bash
# Coming soon!
sudo snap install presto
```

### Chocolatey (Windows)

```powershell
# Coming soon!
choco install presto
```

---

## 🔨 Build from Source

### Prerequisites

- Go 1.21 or higher
- Git
- Make (optional, but recommended)

### Steps

```bash
# Clone repository
git clone https://github.com/paramientos/presto.git
cd presto

# Install dependencies
make deps

# Build
make build

# Install to system
sudo make install

# Verify
presto --version
```

### Manual Build (without Make)

```bash
# Clone repository
git clone https://github.com/paramientos/presto.git
cd presto

# Install dependencies
go mod download

# Build
go build -o presto ./cmd/presto

# Move to PATH
sudo mv presto /usr/local/bin/

# Verify
presto --version
```

---

## 🐳 Docker

```bash
# Coming soon!
docker pull paramientos/presto:latest
docker run --rm -v $(pwd):/app paramientos/presto install
```

---

## ⚙️ Shell Completion

### Bash

```bash
# Generate completion script
presto completion bash > /etc/bash_completion.d/presto

# Or for user-only installation
presto completion bash > ~/.bash_completion.d/presto
source ~/.bash_completion.d/presto
```

### Zsh

```bash
# Generate completion script
presto completion zsh > /usr/local/share/zsh/site-functions/_presto

# Or for user-only installation
presto completion zsh > ~/.zsh/completion/_presto
```

Add to `~/.zshrc`:
```bash
fpath=(~/.zsh/completion $fpath)
autoload -Uz compinit && compinit
```

### Fish

```bash
presto completion fish > ~/.config/fish/completions/presto.fish
```

### PowerShell

```powershell
presto completion powershell | Out-String | Invoke-Expression
```

Add to PowerShell profile for persistence:
```powershell
presto completion powershell >> $PROFILE
```

---

## ✅ Verify Installation

```bash
# Check version
presto --version
# Output: 🎵 Presto v0.1.5

# Check help
presto --help

# Test with a simple command
presto init
```

---

## 🔄 Updating Presto

### Binary Installation

```bash
# Download latest version
curl -L https://github.com/paramientos/presto/releases/latest/download/presto-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o presto
chmod +x presto
sudo mv presto /usr/local/bin/
```

### From Source

```bash
cd presto
git pull origin main
make build
sudo make install
```

---

## 🗑️ Uninstalling Presto

```bash
# Remove binary
sudo rm /usr/local/bin/presto

# Remove completion scripts (if installed)
sudo rm /etc/bash_completion.d/presto
sudo rm /usr/local/share/zsh/site-functions/_presto

# Remove cache (optional)
rm -rf ~/.presto
```

---

## 🆘 Troubleshooting

### Permission Denied

If you get "permission denied" when running presto:

```bash
chmod +x /usr/local/bin/presto
```

### Command Not Found

Make sure `/usr/local/bin` is in your PATH:

```bash
echo $PATH | grep /usr/local/bin
```

If not, add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
export PATH="/usr/local/bin:$PATH"
```

### macOS Gatekeeper Warning

If macOS blocks the binary:

```bash
xattr -d com.apple.quarantine /usr/local/bin/presto
```

Or go to System Preferences → Security & Privacy → Allow

---

## 📚 Next Steps

After installation:

1. Read the [Quick Start Guide](QUICKSTART.md)
2. Check out [Examples](examples/)
3. Join the [Community](https://github.com/paramientos/presto/discussions)

---

**Need help?** [Open an issue](https://github.com/paramientos/presto/issues) or check the [documentation](README.md).
