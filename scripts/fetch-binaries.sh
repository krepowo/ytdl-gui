#!/usr/bin/env bash
#
# Fetch the bundled binaries into resources/.
#
# These are ~220 MB and are NOT committed to git (see .gitignore). Run this once
# after cloning, and again when you want to refresh yt-dlp (its site extractors
# change often, so a newer build fixes more sites).
#
#   ./scripts/fetch-binaries.sh
#
# Then build:  wails build -nsis -installscope user
#
# IMPORTANT: yt-dlp must be the standalone ONEFILE build (yt-dlp.exe, ~18 MB).
# Do NOT copy the winget/scoop "onedir" install (Program Files\yt-dlp\yt-dlp.exe
# plus an _internal\ folder) — that binary needs its sibling DLLs and fails with
# "Failed to load Python DLL ... _internal\python310.dll" when moved alone.
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEST="$ROOT/resources"
mkdir -p "$DEST"

YTDLP_VERSION="${YTDLP_VERSION:-latest}"
FFMPEG_URL="${FFMPEG_URL:-https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip}"

echo "==> Fetching yt-dlp ($YTDLP_VERSION)"
if [ "$YTDLP_VERSION" = "latest" ]; then
  curl -fL --retry 3 -o "$DEST/yt-dlp.exe" \
    "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
else
  curl -fL --retry 3 -o "$DEST/yt-dlp.exe" \
    "https://github.com/yt-dlp/yt-dlp/releases/download/${YTDLP_VERSION}/yt-dlp.exe"
fi

echo "==> Fetching ffmpeg + ffprobe (essentials build)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
curl -fL --retry 3 -o "$TMP/ffmpeg.zip" "$FFMPEG_URL"
unzip -q "$TMP/ffmpeg.zip" -d "$TMP/x"
# The archive nests binaries under ffmpeg-<version>-essentials_build/bin/.
find "$TMP/x" -name 'ffmpeg.exe' -exec cp {} "$DEST/ffmpeg.exe" \;
find "$TMP/x" -name 'ffprobe.exe' -exec cp {} "$DEST/ffprobe.exe" \;

echo "==> Done. resources/:"
ls -lh "$DEST"
