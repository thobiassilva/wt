#!/usr/bin/env bash
set -euo pipefail

REPO="thobiassilva/wt"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

info() { printf "${GREEN}>>>${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}aviso:${NC} %s\n" "$*" >&2; }
die()  { printf "${RED}erro:${NC} %s\n" "$*" >&2; exit 1; }

# --- Detect OS ---
# Supports: Linux (native + WSL), macOS, Git Bash (MINGW/MSYS on Windows)
OS="$(uname -s)"
case "$OS" in
    Linux*)         GOOS="linux" ;;
    Darwin*)        GOOS="darwin" ;;
    MINGW* | MSYS*) GOOS="windows" ;;
    *)              die "Sistema operacional nao suportado: $OS" ;;
esac

# --- Detect arch ---
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)          GOARCH="amd64" ;;
    amd64)           GOARCH="amd64" ;;
    arm64 | aarch64) GOARCH="arm64" ;;
    *)               die "Arquitetura nao suportada: $ARCH" ;;
esac

# --- Detect Rosetta (Darwin arm64 reporting as x86_64) ---
if [[ "$GOOS" == "darwin" && "$GOARCH" == "amd64" ]]; then
    if sysctl -n sysctl.proc_translated 2>/dev/null | grep -q "^1$"; then
        warn "Detectado Rosetta 2. Usando binario arm64 nativo."
        GOARCH="arm64"
    fi
fi

# --- Choose archive format ---
if [[ "$GOOS" == "windows" ]]; then
    ARCHIVE_EXT="zip"
    BINARY_NAME="wt.exe"
else
    ARCHIVE_EXT="tar.gz"
    BINARY_NAME="wt"
fi

# --- Fetch latest version ---
info "Buscando ultima versao..."
if command -v curl >/dev/null 2>&1; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
elif command -v wget >/dev/null 2>&1; then
    VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
else
    die "curl ou wget e necessario para instalar wt"
fi

[[ -n "$VERSION" ]] || die "Nao foi possivel determinar a versao mais recente"
info "Versao: $VERSION"

# --- Build download URLs ---
ARCHIVE="wt_${VERSION#v}_${GOOS}_${GOARCH}.${ARCHIVE_EXT}"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
CHECKSUM_URL="${BASE_URL}/checksums.txt"

# --- Download ---
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

info "Baixando $ARCHIVE..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$TMP_DIR/$ARCHIVE" "$ARCHIVE_URL"
    curl -fsSL -o "$TMP_DIR/checksums.txt" "$CHECKSUM_URL"
else
    wget -qO "$TMP_DIR/$ARCHIVE" "$ARCHIVE_URL"
    wget -qO "$TMP_DIR/checksums.txt" "$CHECKSUM_URL"
fi

# --- Verify checksum ---
info "Verificando checksum..."
if command -v sha256sum >/dev/null 2>&1; then
    SHA_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
    SHA_CMD="shasum -a 256"
else
    warn "sha256sum/shasum nao encontrado — pulando verificacao de checksum"
    SHA_CMD=""
fi

if [[ -n "$SHA_CMD" ]]; then
    EXPECTED=$(grep "$ARCHIVE" "$TMP_DIR/checksums.txt" | awk '{print $1}')
    ACTUAL=$(cd "$TMP_DIR" && $SHA_CMD "$ARCHIVE" | awk '{print $1}')
    [[ "$EXPECTED" == "$ACTUAL" ]] || die "Checksum invalido — download corrompido"
    info "Checksum OK"
fi

# --- Extract ---
if [[ "$ARCHIVE_EXT" == "zip" ]]; then
    if command -v unzip >/dev/null 2>&1; then
        unzip -q "$TMP_DIR/$ARCHIVE" -d "$TMP_DIR"
    else
        die "unzip nao encontrado. Instale via: pacman -S unzip (Git Bash) ou apt install unzip (WSL)"
    fi
else
    tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"
fi

# --- Install ---
mkdir -p "$BIN_DIR"
install -m 0755 "$TMP_DIR/$BINARY_NAME" "$BIN_DIR/$BINARY_NAME"
info "Instalado em: $BIN_DIR/$BINARY_NAME"

# --- Verify PATH ---
if ! printf '%s' "$PATH" | tr ':' '\n' | grep -qx "$BIN_DIR"; then
    SHELL_NAME="$(basename "${SHELL:-sh}")"
    case "$SHELL_NAME" in
        zsh)  RC_FILE="$HOME/.zshrc" ;;
        bash) RC_FILE="$HOME/.bashrc" ;;
        *)    RC_FILE="$HOME/.profile" ;;
    esac

    warn "$BIN_DIR nao esta no PATH."
    printf "\n  Adicione ao ${BOLD}%s${NC}:\n\n" "$RC_FILE"
    printf '    export PATH="$HOME/.local/bin:$PATH"\n\n'
    printf "  Depois execute: ${BOLD}source %s${NC}\n\n" "$RC_FILE"
else
    info "wt ${VERSION} instalado. Execute ${BOLD}wt --help${NC} para comecar."
fi
