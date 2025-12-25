$url = "https://github.com/paramientos/presto/releases/latest/download/presto-windows-amd64.exe"
$dest = Join-Path $env:SystemRoot "System32\presto.exe"

Write-Host "🎵 Downloading Presto for Windows..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $url -OutFile "presto.exe" -ErrorAction Stop
    
    if (Test-Path "presto.exe") {
        Write-Host "📥 Installing Presto to $dest..." -ForegroundColor Cyan
        Move-Item -Path "presto.exe" -Destination $dest -Force -ErrorAction Stop
        Write-Host "✨ Presto installed successfully! Run 'presto --version' to verify." -ForegroundColor Green
    }
} catch {
    Write-Host "❌ Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "💡 Try running PowerShell as Administrator." -ForegroundColor Yellow
}
