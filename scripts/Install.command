#!/bin/bash
# 解压 asm-macos.zip 后，双击本文件即可安装（自动识别 Apple / Intel 芯片）
set -e
cd "$(dirname "$0")"
clear 2>/dev/null || true

echo ""
echo "  ┌──────────────────────────────────────┐"
echo "  │   asm 一键安装 (macOS Intel/Apple)    │"
echo "  └──────────────────────────────────────┘"
echo ""

arch="$(uname -m)"
case "$arch" in
  arm64)  SRC="./asm-arm64"; CHIP="Apple 芯片 (arm64)" ;;
  x86_64) SRC="./asm-amd64"; CHIP="Intel 芯片 (amd64)" ;;
  *)
    echo "  ✗ 不支持的架构: $arch"
    echo "  按回车关闭…"; read -r _; exit 1
    ;;
esac

if [ ! -f "$SRC" ]; then
  echo "  ✗ 找不到 $SRC"
  echo "    请确认已完整解压 asm-macos.zip 后再双击本文件。"
  echo "  按回车关闭…"; read -r _; exit 1
fi

echo "  检测到: $CHIP"
echo "  使用:   $SRC"
echo ""

DEST_DIR="${HOME}/.local/bin"
mkdir -p "$DEST_DIR"

xattr -dr com.apple.quarantine . 2>/dev/null || true

cp -f "$SRC" "${DEST_DIR}/asm"
chmod 755 "${DEST_DIR}/asm"
xattr -d com.apple.quarantine "${DEST_DIR}/asm" 2>/dev/null || true

# PATH
ensure_path_line='export PATH="$HOME/.local/bin:$PATH"'
wrote_rc=""
for rc in "${HOME}/.zprofile" "${HOME}/.zshrc" "${HOME}/.bash_profile" "${HOME}/.bashrc"; do
  if [ -f "$rc" ] && grep -q '\.local/bin' "$rc" 2>/dev/null; then
    wrote_rc="already:$rc"
    break
  fi
done
if [ -z "$wrote_rc" ]; then
  rc="${HOME}/.zshrc"
  if [ -n "${SHELL:-}" ] && echo "$SHELL" | grep -q bash; then
    rc="${HOME}/.bash_profile"
  fi
  {
    echo ""
    echo "# added by asm installer"
    echo "$ensure_path_line"
  } >> "$rc"
  wrote_rc="wrote:$rc"
fi

echo "  ✓ 已安装到: ${DEST_DIR}/asm"
if ver="$("${DEST_DIR}/asm" version 2>/dev/null)"; then
  echo "  ✓ 版本: ${ver}"
fi
echo ""
case "$wrote_rc" in
  wrote:*)
    echo "  ✓ 已配置 PATH → ${wrote_rc#wrote:}"
    echo "    请【新开一个终端】，然后输入:"
    ;;
  *)
    echo "  请【新开一个终端】（若尚未生效），然后输入:"
    ;;
esac
echo ""
echo "      asm version"
echo "      asm scan"
echo ""
echo "  以后升级:  asm upgrade"
echo ""
echo "  按回车关闭…"
read -r _
