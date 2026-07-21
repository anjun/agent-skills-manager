#!/usr/bin/env bash
# 从 GitHub Releases 安装 asm（当前仅支持 macOS 通用包 asm-macos.zip）
#
#   gh api -H 'Accept: application/vnd.github.raw' \
#     repos/anjun/agent-skills-manager/contents/scripts/install.sh | bash
#   bash scripts/install.sh
#
# Env: ASM_GITHUB_REPO  ASM_VERSION  ASM_INSTALL_DIR  GITHUB_TOKEN/GH_TOKEN
set -euo pipefail

REPO="${ASM_GITHUB_REPO:-anjun/agent-skills-manager}"
VERSION_RAW="${ASM_VERSION:-}"
CMD_NAME="asm"
ASSET_NAME="asm-macos.zip"

info()  { printf '  → %s\n' "$*"; }
die()   { printf '  ✗ %s\n' "$*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

detect_platform() {
  local os arch
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "$os" in
    darwin*) ;;
    *) die "当前安装包仅支持 macOS（Intel / Apple 芯片）。检测到: $os" ;;
  esac
  case "$arch" in
    arm64)  ARCH_BIN="asm-arm64"; CHIP="Apple 芯片 (arm64)" ;;
    x86_64) ARCH_BIN="asm-amd64"; CHIP="Intel 芯片 (amd64)" ;;
    *) die "不支持的 Mac 架构: $arch" ;;
  esac
  ARCH_RAW="$arch"
}

resolve_install_dir() {
  if [ -n "${ASM_INSTALL_DIR:-}" ]; then
    INSTALL_DIR="$ASM_INSTALL_DIR"
  else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
}

github_token() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then printf '%s' "$GITHUB_TOKEN"; return; fi
  if [ -n "${GH_TOKEN:-}" ]; then printf '%s' "$GH_TOKEN"; return; fi
  if command -v gh >/dev/null 2>&1; then gh auth token 2>/dev/null || true; fi
}

resolve_tag() {
  local raw token tag api
  raw="${VERSION_RAW#v}"
  if [ -n "$raw" ]; then TAG="v${raw}"; return; fi

  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    tag="$(gh release view -R "$REPO" --json tagName -q .tagName 2>/dev/null || true)"
    if [ -n "$tag" ]; then TAG="$tag"; return; fi
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
  [ -n "$tag" ] || die "无法解析最新 Release。私有仓请 gh auth login 或设置 GITHUB_TOKEN"
  TAG="$tag"
}

download_zip() {
  local dest="$1"
  if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
    info "gh 下载 ${ASSET_NAME} (${TAG})"
    rm -f "$dest"
    if gh release download "$TAG" -R "$REPO" -p "$ASSET_NAME" -O "$dest" --clobber 2>/dev/null && [ -s "$dest" ]; then
      return 0
    fi
  fi

  local token api_url
  token="$(github_token)"
  info "API 下载 ${ASSET_NAME} (${TAG})"
  api_url="$(
    if [ -n "$token" ]; then
      curl -fsSL -H "Authorization: Bearer ${token}" -H "Accept: application/vnd.github+json" \
        -H "User-Agent: asm-install" "https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
    else
      curl -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: asm-install" \
        "https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
    fi | python3 -c "
import json,sys
want=sys.argv[1]
d=json.load(sys.stdin)
for a in d.get('assets') or []:
    if a.get('name')==want:
        print(a.get('url') or '')
        raise SystemExit(0)
raise SystemExit(1)
" "$ASSET_NAME" 2>/dev/null || true
  )"
  [ -n "$api_url" ] || die "Release ${TAG} 中没有 ${ASSET_NAME}"

  if [ -n "$token" ]; then
    curl -fsSL -H "Authorization: Bearer ${token}" -H "Accept: application/octet-stream" \
      -H "User-Agent: asm-install" -o "$dest" "$api_url"
  else
    curl -fsSL -H "Accept: application/octet-stream" -H "User-Agent: asm-install" \
      -o "$dest" "$api_url" \
      || curl -fsSL -L -o "$dest" "https://github.com/${REPO}/releases/download/${TAG}/${ASSET_NAME}"
  fi
  [ -s "$dest" ] || die "下载失败"
}

install_from_zip() {
  local zipfile="$1"
  local stage target
  stage="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap 'rm -rf "'"$stage"'"' RETURN
  unzip -q -o "$zipfile" -d "$stage"
  [ -f "${stage}/${ARCH_BIN}" ] || die "zip 内缺少 ${ARCH_BIN}"

  target="${INSTALL_DIR}/${CMD_NAME}"
  mkdir -p "$INSTALL_DIR"
  cp -f "${stage}/${ARCH_BIN}" "$target"
  chmod 755 "$target"
  xattr -d com.apple.quarantine "$target" 2>/dev/null || true
}

path_hint() {
  case ":$PATH:" in
    *":${INSTALL_DIR}:"*) return 0 ;;
  esac
  printf '\n'
  printf '  ⚠ %s 不在 PATH。把下面一行加进 ~/.zshrc：\n' "$INSTALL_DIR"
  printf '      export PATH="%s:$PATH"\n' "$INSTALL_DIR"
  printf '    然后: source ~/.zshrc  或新开终端\n'
}

main() {
  printf '\n'
  printf '  ┌──────────────────────────────────────────┐\n'
  printf '  │  asm installer (macOS 通用包)             │\n'
  printf '  └──────────────────────────────────────────┘\n'
  printf '\n'

  need_cmd uname
  need_cmd curl
  need_cmd unzip
  need_cmd mktemp

  detect_platform
  resolve_install_dir
  info "芯片: ${CHIP}"
  info "仓库: ${REPO}"
  resolve_tag
  info "版本: ${TAG}"
  info "安装到: ${INSTALL_DIR}"

  work="$(mktemp -d)"
  # shellcheck disable=SC2064
  trap 'rm -rf "'"$work"'"' EXIT
  dl="${work}/${ASSET_NAME}"
  download_zip "$dl"
  install_from_zip "$dl"

  printf '\n'
  ver="$("${INSTALL_DIR}/${CMD_NAME}" version 2>/dev/null || true)"
  info "安装成功: ${INSTALL_DIR}/${CMD_NAME}  (${ver:-ok})"
  path_hint
  printf '\n'
  printf '  快速开始:  asm scan\n'
  printf '  以后升级:  asm upgrade\n'
  printf '\n'
}

main "$@"
