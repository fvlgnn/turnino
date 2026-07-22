#!/bin/sh
set -eu

release_version=${1:?release version is required}
release_os=${GOOS:?GOOS is required}
release_arch=${GOARCH:?GOARCH is required}
release_commit=$(git rev-parse HEAD)
release_date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
release_name="turnino_${release_version#v}_${release_os}_${release_arch}"
release_dir="dist/${release_name}"
binary_name=turnino

if [ "$release_os" = "windows" ]; then
  binary_name=turnino.exe
fi

mkdir -p "$release_dir"
go build -trimpath \
  -ldflags="-s -w -X main.version=${release_version} -X main.commit=${release_commit} -X main.date=${release_date}" \
  -o "${release_dir}/${binary_name}" ./cmd/turnino

cp README.md "${release_dir}/"

if [ "$release_os" = "windows" ]; then
  (cd dist && zip -r "${release_name}.zip" "${release_name}")
else
  tar -C dist -czf "dist/${release_name}.tar.gz" "${release_name}"
fi

rm -r "$release_dir"
