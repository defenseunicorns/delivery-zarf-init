#!/usr/bin/env bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

base_ref="${1:?base ref is required}"
ready=true
last_changed=0

read_value() {
  local revision="$1"
  local file="$2"
  local query="$3"

  if [[ "$revision" == "worktree" ]]; then
    yq -r "$query" "$file"
  else
    git show "$revision:$file" | yq -r "$query"
  fi
}

normalize_version() {
  sed -E 's/^v?([0-9]+(\.[0-9]+)+).*/\1/'
}

check_lockstep() {
  local component="$1"
  shift
  local changed=0
  local expected=""
  local mismatched=false
  local spec file query old_value new_value normalized

  for spec in "$@"; do
    IFS='|' read -r file query <<< "$spec"
    old_value=$(read_value "$base_ref" "$file" "$query")
    new_value=$(read_value worktree "$file" "$query")

    if [[ "$old_value" != "$new_value" ]]; then
      changed=$((changed + 1))
    fi

    normalized=$(printf '%s\n' "$new_value" | normalize_version)
    if [[ -z "$expected" ]]; then
      expected="$normalized"
    elif [[ "$normalized" != "$expected" ]]; then
      mismatched=true
    fi
  done

  last_changed=$changed
  if [[ "$changed" -eq 0 ]]; then
    return
  fi

  if [[ "$changed" -ne "$#" ]]; then
    echo "$component is waiting for all flavor pins to update ($changed/$# changed)" >&2
    ready=false
  fi
  if [[ "$mismatched" == "true" ]]; then
    echo "$component flavor versions do not match" >&2
    ready=false
  fi
}

zarf=(
  'tasks.yaml|.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default'
  'flavors/upstream.yaml|.package.create.set.agent_image_tag'
  'flavors/registry1.yaml|.package.create.set.agent_image_tag'
  'flavors/registry1-arm64.yaml|.package.create.set.agent_image_tag'
  'flavors/unicorn.yaml|.package.create.set.agent_image_tag'
  'releaser.yaml|.packages[0].flavors[0].version'
  'releaser.yaml|.packages[0].flavors[1].version'
  'releaser.yaml|.packages[0].flavors[2].version'
  'releaser.yaml|.packages[1].flavors[0].version'
  'releaser.yaml|.packages[1].flavors[1].version'
  'releaser.yaml|.packages[1].flavors[2].version'
  'releaser.yaml|.packages[2].flavors[0].version'
  'releaser.yaml|.packages[2].flavors[1].version'
  'releaser.yaml|.packages[2].flavors[2].version'
)
registry=(
  'flavors/upstream.yaml|.package.create.set.registry_image_tag'
  'flavors/registry1.yaml|.package.create.set.registry_image_tag'
  'flavors/registry1-arm64.yaml|.package.create.set.registry_image_tag'
  'flavors/unicorn.yaml|.package.create.set.registry_image_tag'
)
socat=(
  'flavors/upstream.yaml|.package.create.set.proxy_image_tag'
  'flavors/registry1.yaml|.package.create.set.proxy_image_tag'
  'flavors/registry1-arm64.yaml|.package.create.set.proxy_image_tag'
  'flavors/unicorn.yaml|.package.create.set.proxy_image_tag'
)
gitea=(
  'flavors/upstream.yaml|.package.create.set.gitea_image | split(":")[-1]'
  'flavors/registry1.yaml|.package.create.set.gitea_image | split(":")[-1]'
  'flavors/registry1-arm64.yaml|.package.create.set.gitea_image | split(":")[-1]'
  'flavors/unicorn.yaml|.package.create.set.gitea_image | split(":")[-1]'
)

check_lockstep zarf "${zarf[@]}"
check_lockstep registry "${registry[@]}"
check_lockstep socat "${socat[@]}"
check_lockstep gitea "${gitea[@]}"
gitea_images_changed=$last_changed

old_chart=$(read_value "$base_ref" components/common/zarf.yaml '.components[] | select(.name == "git-server") | .charts[0].version')
new_chart=$(read_value worktree components/common/zarf.yaml '.components[] | select(.name == "git-server") | .charts[0].version')
if [[ "$old_chart" != "$new_chart" && "$gitea_images_changed" -eq 0 ]]; then
  echo "gitea chart is waiting for all flavor images" >&2
  ready=false
fi

if [[ "$ready" != "true" ]]; then
  exit 1
fi

echo "Renovate component versions are ready"
