#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
DIST_DIR="${DIST_DIR:-dist}"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
DIST_DIR="$(cd "$DIST_DIR" && pwd -P)"

ldflags="-s -w -X main.buildVersion=${VERSION} -X main.buildCommit=${COMMIT} -X main.buildDate=${BUILD_DATE}"
default_targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"
read -r -a targets <<< "${TARGETS:-$default_targets}"
commands=(
  "server:eve-sde-server"
  "migrate:eve-sde-migrate"
  "import-sde:eve-sde-import-sde"
)

for target in "${targets[@]}"; do
  IFS="/" read -r goos goarch <<< "$target"
  bundle="eve-sde-server_${VERSION}_${goos}_${goarch}"
  package_root="$(mktemp -d)"
  package_dir="${package_root}/${bundle}"
  mkdir -p "$package_dir"

  ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi

  for command in "${commands[@]}"; do
    IFS=":" read -r cmd output_name <<< "$command"
    echo "building ${output_name}${ext} for ${goos}/${goarch}"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build \
      -trimpath \
      -ldflags "$ldflags" \
      -o "${package_dir}/${output_name}${ext}" \
      "./cmd/${cmd}"
  done

  cp README.md LICENSE "$package_dir/"

  if [[ "$goos" == "windows" ]]; then
    (cd "$package_root" && zip -qr "${DIST_DIR}/${bundle}.zip" "$bundle")
  else
    tar -C "$package_root" -czf "${DIST_DIR}/${bundle}.tar.gz" "$bundle"
  fi

  rm -rf "$package_root"
done

(cd "$DIST_DIR" && sha256sum * > checksums.txt)
