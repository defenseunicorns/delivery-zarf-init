#!/bin/sh
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -eu
# shellcheck disable=SC3040
(set -o pipefail 2>/dev/null) && set -o pipefail

base_ref="${1:?base ref is required}"
base_releaser=$(mktemp)
entries=$(mktemp)
trap 'rm -f "$base_releaser" "$entries"' EXIT
git show "$base_ref:releaser.yaml" > "$base_releaser"

artifact_changed=false
require_all=false
required_packages=""
required_flavors=""

append_unique() {
  case " $1 " in
    *" $2 "*) printf '%s\n' "$1" ;;
    *) printf '%s %s\n' "$1" "$2" ;;
  esac
}

for path in $(git diff --name-only "$base_ref" -- components flavors packages); do
  artifact_changed=true
  case "$path" in
    components/*) require_all=true ;;
    flavors/upstream.yaml) required_flavors=$(append_unique "$required_flavors" upstream) ;;
    flavors/registry1*.yaml) required_flavors=$(append_unique "$required_flavors" registry1) ;;
    flavors/unicorn.yaml) required_flavors=$(append_unique "$required_flavors" unicorn) ;;
    packages/*)
      package=${path#packages/}
      package=${package%%/*}
      required_packages=$(append_unique "$required_packages" "$package")
      ;;
  esac
done

source_query='.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default'
base_source=$(git show "$base_ref:tasks.yaml" | yq -r "$source_query")
current_source=$(yq -r "$source_query" tasks.yaml)
if [ "$base_source" != "$current_source" ]; then
  artifact_changed=true
  require_all=true
  expected_version="${current_source}-uds.0"
else
  # shellcheck disable=SC2016
  max_revision=$(SOURCE_VERSION="$current_source" yq -r '
    .packages[].flavors[].version |
    split("-uds.") as $parts |
    select(($parts | length) == 2 and $parts[0] == env(SOURCE_VERSION)) |
    $parts[1]
  ' "$base_releaser" | sort -n | tail -n 1)
  case "$max_revision" in
    ''|*[!0-9]*) echo "cannot resolve the base UDS revision for $current_source" >&2; exit 1 ;;
  esac
  expected_version="${current_source}-uds.$((max_revision + 1))"
fi

# shellcheck disable=SC2016
yq -r '.packages[] | .name as $package | .flavors[] | "\($package)|\(.name)|\(.version)"' \
  releaser.yaml > "$entries"

changed_count=0
missing=false
invalid=false
while IFS='|' read -r package flavor version; do
  base_version=$(PACKAGE_NAME="$package" FLAVOR_NAME="$flavor" yq -r '
    .packages[] | select(.name == env(PACKAGE_NAME)) |
    .flavors[] | select(.name == env(FLAVOR_NAME)) | .version // ""
  ' "$base_releaser")

  changed=false
  if [ "$version" != "$base_version" ]; then
    changed=true
    changed_count=$((changed_count + 1))
    if [ "$version" != "$expected_version" ]; then
      echo "$package/$flavor uses $version; expected $expected_version" >&2
      invalid=true
    fi
  fi

  required=false
  if [ "$require_all" = "true" ]; then
    required=true
  else
    case " $required_packages " in *" $package "*) required=true ;; esac
    case " $required_flavors " in *" $flavor "*) required=true ;; esac
  fi
  if [ "$required" = "true" ] && [ "$changed" != "true" ]; then
    echo "$package/$flavor changed artifact inputs but remains at $version" >&2
    missing=true
  fi
done < "$entries"

if [ "$artifact_changed" = "true" ] && [ "$changed_count" -eq 0 ]; then
  echo "artifact inputs changed without a releaser.yaml version bump" >&2
  missing=true
fi

if [ "$missing" = "true" ] || [ "$invalid" = "true" ]; then
  echo "Expected release version: $expected_version" >&2
  if [ "$base_source" != "$current_source" ]; then
    echo "A Zarf update must set every releaser entry to $expected_version." >&2
  else
    suggested_package=all
    suggested_flavor=all
    if [ "$require_all" != "true" ] && [ -z "$required_packages" ]; then
      # shellcheck disable=SC2086
      set -- $required_flavors
      [ "$#" -eq 1 ] && suggested_flavor="$1"
    elif [ "$require_all" != "true" ] && [ -z "$required_flavors" ]; then
      # shellcheck disable=SC2086
      set -- $required_packages
      [ "$#" -eq 1 ] && suggested_package="$1"
    fi
    echo "Run: uds run bump-release --set PACKAGE=$suggested_package --set FLAVOR=$suggested_flavor" >&2
  fi
  exit 1
fi

if [ "$changed_count" -gt 0 ]; then
  echo "$changed_count release entries are ready at $expected_version"
else
  echo "No package release required"
fi
