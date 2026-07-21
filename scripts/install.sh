#!/usr/bin/env bash
# Install agent-skills-manager (asm) from GitHub Releases into PATH.
#
# Usage:
#   curl -fsSL … | bash                          # public repo raw URL
#   bash scripts/install.sh                      # from a clone
#   gh api -H 'Accept: application/vnd.github.raw' \
#     repos/anjun/agent-skills-manager/contents/scripts/install.sh | bash
#
# Env:
#   ASM_GITHUB_REPO   owner/repo   (default: anjun/agent-skills-manager)
#   ASM_VERSION       tag or version, e.g. v0.3.0 / 0.3.0  (default: latest stable)
#   ASM_INSTALL_DIR   install dir  (default: ~/.local/bin, fallback ~/bin)
#   GITHUB_TOKEN / GH_TOKEN   for private repos (optional if `gh auth` works)
set -euo pipefail

REPO="${ASM_GITHUB_REPO:-anjun/agent-skills-manager}"
VERSION_RAW="${ASM_VERSION:-}"
PREFIX_BIN="agent-skills-manager"
CMD_NAME="asm"

info()  { printf '  → %s\n' "$*"; }
warn()  { printf '  ⚠ %s\n' "$*" >&2; }
die()   { printf '  ✗ %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

# ── platform ───────────────────────────────────────────────────────────────

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    linux*)  os=linux ;;
    darwin*) os=darwin ;;
    msys*|mingw*|cygwin*) die "请在 Windows 上下载 .exe 或使用 WSL 运行本脚本" ;;
    *) die "不支持的操作系统: $os" ;;
  esac
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) die "不支持的 CPU 架构: $arch" ;;
  esac
  OS="$os"
  ARCH="$arch"
  ASSET_BASE="${PREFIX_BIN}-${os}-${arch}"
}

# ── install dir ────────────────────────────────────────────────────────────

resolve_install_dir() {
  if [ -n "${ASM_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$ASM_INSTALL_DIR"
  elif [ -d "${HOME}/.local/bin" ] || mkdir -p "${HOME}/.local/bin" 2>/dev/null; then
    INSTALL_DIR="${HOME}/.local/bin"
  else
    INSTALL_DIR="${HOME}/bin"
    mkdir -p "$INSTALL_DIR"
  fi
}

# ── auth / download helpers ────────────────────────────────────────────────

github_token() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    printf '%s' "$GITHUB_TOKEN"
    return
  fi
  if [ -n "${GH_TOKEN:-}" ]; then
    printf '%s' "$GH_TOKEN"
    return
  fi
  if command -v gh >/dev/null 2>&1; then
    gh auth token 2>/dev/null || true
  fi
}

# Resolve tag: empty → latest stable (not prerelease)
resolve_tag() {
  local raw token api tag
  raw="${VERSION_RAW#v}"
  if [ -n "$raw" ]; then
    TAG="v${raw}"
    return
  fi

  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    tag="$(gh release view -R "$REPO" --json tagName -q .tagName 2>/dev/null || true)"
    if [ -n "$tag" ]; then
      TAG="$tag"
      return
    fi
  fi

  token="$(github_token)"
  api="https://api.github.com/repos/${REPO}/releases/latest"
  if [ -n "$token" ]; then
    tag="$(curl -fsSL -H "Authorization: Bearer ${token}" -H "Accept: application/vnd.github+json" \
      -H "User-Agent: asm-install" "$api" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
  else
    tag="$(curl -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: asm-install" \
      "$api" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)"
  fi
  [ -n "$tag" ] || die "无法解析最新 Release。私有仓请: gh auth login 或设置 GITHUB_TOKEN。仓库: ${REPO}"
  TAG="$tag"
}

# Download asset name into $1 (file path). Prefer .tar.gz then bare binary.
download_asset() {
  local dest="$1"
  local names=("${ASSET_BASE}.tar.gz" "${ASSET_BASE}")
  local name url token tmp api_asset

  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    for name in "${names[@]}"; do
      info "尝试 gh 下载: ${name} (${TAG})"
      rm -f "$dest"
      if gh release download "$TAG" -R "$REPO" -p "$name" -O "$dest" --clobber 2>/dev/null \
        && [ -s "$dest" ]; then
        DOWNLOADED_NAME="$name"
        return 0
      fi
      rm -f "$dest"
    done
  fi

  token="$(github_token)"
  for name in "${names[@]}"; do
    info "尝试 API 下载: ${name} (${TAG})"
    # Resolve asset API URL (works for private repos with token)
    api_asset="$(
      if [ -n "$token" ]; then
        curl -fsSL -H "Authorization: Bearer ${token}" -H "Accept: application/vnd.github+json" \
          -H "User-Agent: asm-install" \
          "https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
      else
        curl -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: asm-install" \
          "https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
      fi 2>/dev/null | python3 -c "
import json,sys
name=sys.argv[1]
try:
    d=json.load(sys.stdin)
except Exception:
    sys.exit(1)
for a in d.get('assets') or []:
    if a.get('name')==name:
        print(a.get('url') or '')
        sys.exit(0)
sys.exit(1)
" "$name" 2>/dev/null || true
    )"
    if [ -z "$api_asset" ]; then
      continue
    fi
    tmp="${dest}.part"
    if [ -n "$token" ]; then
      curl -fsSL -H "Authorization: Bearer ${token}" -H "Accept: application/octet-stream" \
        -H "User-Agent: asm-install" -o "$tmp" "$api_asset"
    else
      curl -fsSL -H "Accept: application/octet-stream" -H "User-Agent: asm-install" \
        -o "$tmp" "$api_asset" || {
        # public fallback: browser URL
        curl -fsSL -L -o "$tmp" \
          "https://github.com/${REPO}/releases/download/${TAG}/${name}" || { rm -f "$tmp"; continue; }
      }
    fi
    mv "$tmp" "$dest"
    DOWNLOADED_NAME="$name"
    return 0
  done
  die "找不到适合本机的资源 (${ASSET_BASE}[.tar.gz])。tag=${TAG} repo=${REPO}"
}

install_binary() {
  local archive="$1"
  local target="${INSTALL_DIR}/${CMD_NAME}"
  local full="${INSTALL_DIR}/${PREFIX_BIN}"
  local stage binpath

  stage="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap 'rm -rf "'"$stage"'"' RETURN

  case "$DOWNLOADED_NAME" in
    *.tar.gz)
      tar -xzf "$archive" -C "$stage"
      if [ -f "${stage}/asm" ]; then
        binpath="${stage}/asm"
      elif [ -f "${stage}/${PREFIX_BIN}" ]; then
        binpath="${stage}/${PREFIX_BIN}"
      else
        binpath="$(find "$stage" -type f | head -1)"
      fi
      ;;
    *)
      binpath="$archive"
      ;;
  esac
  [ -n "$binpath" ] && [ -f "$binpath" ] || die "归档中未找到二进制"
  chmod 755 "$binpath"
  if [ "$(uname -s)" = "Darwin" ]; then
    xattr -d com.apple.quarantine "$binpath" 2>/dev/null || true
  fi

  mkdir -p "$INSTALL_DIR"
  # install both asm and long name
  cp "$binpath" "$target"
  chmod 755 "$target"
  cp "$binpath" "$full"
  chmod 755 "$full"
  if [ "$(uname -s)" = "Darwin" ]; then
    xattr -d com.apple.quarantine "$target" 2>/dev/null || true
    xattr -d com.apple.quarantine "$full" 2>/dev/null || true
  fi
}

path_hint() {
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) return 0 ;;
  esac
  printf '\n'
  printf '  ⚠ %s 不在 PATH 中。把下面一行加进 ~/.zshrc 或 ~/.bashrc：\n' "$INSTALL_DIR"
  printf '      export PATH="%s:$PATH"\n' "$INSTALL_DIR"
  printf '    然后执行: source ~/.zshrc   # 或重新打开终端\n'
}

# ── main ───────────────────────────────────────────────────────────────────

main() {
  printf '\n'
  printf '  ┌──────────────────────────────────────────┐\n'
  printf '  │  agent-skills-manager (asm) installer    │\n'
  printf '  └──────────────────────────────────────────┘\n'
  printf '\n'

  need_cmd uname
  need_cmd curl
  need_cmd tar
  need_cmd mktemp
  # python3 optional but preferred for JSON; fallback paths use gh/sed

  detect_platform
  resolve_install_dir
  info "平台: ${OS}/${ARCH}"
  info "仓库: ${REPO}"
  resolve_tag
  info "版本: ${TAG}"
  info "安装到: ${INSTALL_DIR}"

  work="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap 'rm -rf "'"$work"'"' EXIT

  dl="${work}/asset"
  download_asset "$dl"
  info "已下载: ${DOWNLOADED_NAME}"
  install_binary "$dl"

  printf '\n'
  if command -v "$CMD_NAME" >/dev/null 2>&1 || [ -x "${INSTALL_DIR}/${CMD_NAME}" ]; then
    ver="$("${INSTALL_DIR}/${CMD_NAME}" version 2>/dev/null || true)"
    info "安装成功: ${INSTALL_DIR}/${CMD_NAME}  (${ver:-ok})"
    info "同目录也有: ${INSTALL_DIR}/${PREFIX_BIN}"
  fi
  path_hint
  printf '\n'
  printf '  快速开始:\n'
  printf '    asm scan\n'
  printf '    asm plan\n'
  printf '    asm apply\n'
  printf '    asm upgrade --check\n'
  printf '\n'
}

main "$@"
