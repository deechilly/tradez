#!/usr/bin/env bash
# record.sh — build tradez and record tour.tape → recording.gif
set -euo pipefail

RED='\033[0;31m'
GRN='\033[0;32m'
YLW='\033[0;33m'
NC='\033[0m'

fail() { echo -e "${RED}Error:${NC} $1"; exit 1; }
ok()   { echo -e "${GRN}✓${NC} $1"; }
info() { echo -e "${YLW}→${NC} $1"; }

# ── dependency checks ─────────────────────────────────────────────────────────
check() {
    command -v "$1" >/dev/null 2>&1 && return 0
    echo -e "${RED}missing:${NC} $1 — $2"
    return 1
}

deps_ok=true
check vhs   "go install github.com/charmbracelet/vhs@latest"     || deps_ok=false
check ttyd  "https://github.com/tsl0922/ttyd/releases  (macOS: brew install ttyd)" || deps_ok=false
check ffmpeg "sudo apt install ffmpeg  (macOS: brew install ffmpeg)"                || deps_ok=false
$deps_ok || fail "install missing dependencies and re-run"

# ── build ─────────────────────────────────────────────────────────────────────
info "Building tradez binary..."
go build -o tradez .
ok "Build complete"

# ── record ────────────────────────────────────────────────────────────────────
info "Recording tour (~40 seconds)..."
info "Note: this will overwrite any save in Game 1 slot (~/.tradez/saves/slot1.json)"
vhs tour.tape
ok "Recording complete → recording.gif"

# ── cleanup ───────────────────────────────────────────────────────────────────
rm -f tradez
ok "Cleaned up binary"
