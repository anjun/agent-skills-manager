#!/usr/bin/env bash
# Build multi-platform binaries and create a GitHub Release.
# Usage:
#   ./scripts/release.sh 0.3.0
#   ./scripts/release.sh v0.3.0 --draft
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION_RAW="${1:-}"
if [[ -z "$VERSION_RAW" ]]; then
  echo "usage: $0 <version> [--draft]" >&2
  exit 2
fi
shift || true
DRAFT=0
for a in "$@"; do
  case "$a" in
    --draft) DRAFT=1 ;;
  esac
done

VERSION="${VERSION_RAW#v}"
TAG="v${VERSION}"
DIST="$ROOT/dist"
rm -rf "$DIST"
mkdir -p "$DIST"

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

echo "Building $TAG …"
for t in "${targets[@]}"; do
  set -- $t
  goos=$1
  goarch=$2
  out="agent-skills-manager-${goos}-${goarch}"
  if [[ "$goos" == "windows" ]]; then
    out="${out}.exe"
  fi
  echo "  → $out"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o "$DIST/$out" \
    ./cmd/agent-skills-manager
done

echo "Creating GitHub Release $TAG …"
args=(release create "$TAG" "$DIST"/* --title "$TAG" --generate-notes)
if [[ "$DRAFT" == "1" ]]; then
  args+=(--draft)
fi
gh "${args[@]}"
echo "Done: https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/releases/tag/$TAG"
