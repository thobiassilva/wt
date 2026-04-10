#!/usr/bin/env bash
set -euo pipefail

REPO_URL="https://github.com/thobiassilva/wt.git"
INSTALL_DIR="$HOME/.wt"
BIN_DIR="$HOME/.local/bin"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BOLD='\033[1m'
NC='\033[0m'

info() { printf "${GREEN}>>>${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}aviso:${NC} %s\n" "$*"; }
die()  { printf "${RED}erro:${NC} %s\n" "$*" >&2; exit 1; }

# --- Verificar git ---
command -v git >/dev/null 2>&1 || die "git nao encontrado. Instale o git primeiro."

# --- Clonar ou atualizar ---
if [[ -d "$INSTALL_DIR" ]]; then
    info "Atualizando $INSTALL_DIR..."
    git -C "$INSTALL_DIR" pull --ff-only || die "Falha ao atualizar. Resolva manualmente em $INSTALL_DIR"
else
    info "Clonando em $INSTALL_DIR..."
    git clone "$REPO_URL" "$INSTALL_DIR" || die "Falha ao clonar o repositorio"
fi

# --- Garantir permissao de execucao ---
chmod +x "$INSTALL_DIR/wt"

# --- Criar bin dir se necessario ---
mkdir -p "$BIN_DIR"

# --- Criar symlink ---
ln -sf "$INSTALL_DIR/wt" "$BIN_DIR/wt"
info "Symlink criado: $BIN_DIR/wt -> $INSTALL_DIR/wt"

# --- Verificar PATH ---
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$BIN_DIR"; then
    SHELL_NAME=$(basename "$SHELL")
    case "$SHELL_NAME" in
        zsh)  RC_FILE="$HOME/.zshrc" ;;
        bash) RC_FILE="$HOME/.bashrc" ;;
        *)    RC_FILE="$HOME/.profile" ;;
    esac

    warn "$BIN_DIR nao esta no PATH."
    printf "\n  Adicione ao ${BOLD}%s${NC}:\n\n" "$RC_FILE"
    printf "    export PATH=\"\$HOME/.local/bin:\$PATH\"\n\n"
    printf "  Depois execute: ${BOLD}source %s${NC}\n\n" "$RC_FILE"
else
    info "wt instalado com sucesso. Execute ${BOLD}wt --help${NC} para comecar."
fi
