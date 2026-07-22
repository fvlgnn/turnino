#!/bin/sh
set -eu

release_tag=${1:?release tag is required}
changelog_file=${2:-CHANGELOG.md}
release_version=${release_tag#v}

if [ ! -f "$changelog_file" ]; then
  echo "Changelog not found: $changelog_file" >&2
  exit 1
fi

awk -v version="$release_version" '
  $0 ~ "^## \\[" version "\\]([[:space:]]|$)" {
    found = 1
    print
    next
  }
  found && /^## \[/ {
    exit
  }
  found {
    print
  }
  END {
    if (!found) {
      exit 2
    }
  }
' "$changelog_file" || {
  status=$?
  if [ "$status" -eq 2 ]; then
    echo "No changelog section found for $release_tag (expected: ## [$release_version])" >&2
  fi
  exit "$status"
}
