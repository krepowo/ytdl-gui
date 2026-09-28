# Runs the app for manual testing.
#
# The app resolves yt-dlp/ffmpeg/ffprobe from the folder NEXT TO the exe (see
# internal/ytdlp/binary.go). The Wails build only emits VideoDownloader.exe, so
# this script stages the three binaries from resources/ into build/bin/ first,
# then launches the app. Without them the UI still runs but every probe/download
# fails with a clear "binary not found" error.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File scripts/run.ps1            # build if needed, stage, run
#   powershell -ExecutionPolicy Bypass -File scripts/run.ps1 -NoBuild   # just stage + run
#   powershell -ExecutionPolicy Bypass -File scripts/run.ps1 -Dev       # wails dev (hot reload)

param(
    [switch]$NoBuild,
    [switch]$Dev
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$binDir = Join-Path $root "build\bin"
$resDir = Join-Path $root "resources"
$exe    = Join-Path $binDir "VideoDownloader.exe"

# --- Dev mode: wails dev (needs the binaries staged the same way) -------------
if ($Dev) {
    foreach ($t in @("yt-dlp.exe", "ffmpeg.exe", "ffprobe.exe")) {
        $src = Join-Path $resDir $t
        if (-not (Test-Path $src)) {
            Write-Host "ERROR: missing $src" -ForegroundColor Red
            Write-Host "Run scripts/fetch-binaries.sh first." -ForegroundColor Yellow
            exit 1
        }
        Copy-Item $src $binDir -Force
    }
    # `unset tmp`: a leaked lowercase $env:tmp shadows Windows TMP and mangles
    # the build path. Remove it for the child process.
    if (Test-Path Env:\tmp) { Remove-Item Env:\tmp }
    & wails dev
    exit $LASTEXITCODE
}

# --- Build if asked (default) ------------------------------------------------
if (-not $NoBuild) {
    Write-Host "Building..." -ForegroundColor Cyan
    if (Test-Path Env:\tmp) { Remove-Item Env:\tmp }
    & wails build -clean
    if ($LASTEXITCODE -ne 0) { Write-Host "Build failed." -ForegroundColor Red; exit 1 }
}

if (-not (Test-Path $exe)) {
    Write-Host "ERROR: $exe not found. Run without -NoBuild." -ForegroundColor Red
    exit 1
}

# --- Stage the bundled binaries next to the exe -------------------------------
foreach ($t in @("yt-dlp.exe", "ffmpeg.exe", "ffprobe.exe")) {
    $src = Join-Path $resDir $t
    if (-not (Test-Path $src)) {
        Write-Host "WARNING: missing $src - downloads will fail." -ForegroundColor Yellow
        continue
    }
    Copy-Item $src $binDir -Force
}
Write-Host "Binaries staged in build\bin" -ForegroundColor Green

Write-Host "Launching VideoDownloader..." -ForegroundColor Cyan
& $exe
