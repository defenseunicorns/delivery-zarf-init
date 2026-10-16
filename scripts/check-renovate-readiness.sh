#!/bin/sh
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -eu
# shellcheck disable=SC3040 # pipefail is standardized by POSIX.1-2024.
set -o pipefail

base_ref="${1:?base ref is required}"
ready=true
last_changed=0

read_value() {
  if [ "$1" = "worktree" ]; then
    yq -r "$3" "$2"
  else
    document=$(git show "$1:$2")
    printf '%s\n' "$document" | yq -r "$3"
  fi
}

normalize_version() {
  sed 's/^v\{0,1\}\([0-9][0-9]*\(\.[0-9][0-9]*\)\{1,\}\).*/\1/'
}

check_lockstep() {
  lockstep_component="$1"
  shift
  lockstep_total=$#
  lockstep_changed=0
  lockstep_expected=""
  lockstep_mismatched=false

  for lockstep_spec in "$@"; do
    lockstep_file=${lockstep_spec%%|*}
    lockstep_query=${lockstep_spec#*|}
    lockstep_old_value=$(read_value "$base_ref" "$lockstep_file" "$lockstep_query")
    lockstep_new_value=$(read_value worktree "$lockstep_file" "$lockstep_query")

    if [ "$lockstep_old_value" != "$lockstep_new_value" ]; then
      lockstep_changed=$((lockstep_changed + 1))
    fi

    lockstep_normalized=$(printf '%s\n' "$lockstep_new_value" | normalize_version)
    if [ -z "$lockstep_expected" ]; then
      lockstep_expected="$lockstep_normalized"
    elif [ "$lockstep_normalized" != "$lockstep_expected" ]; then
      lockstep_mismatched=true
    fi
  done

  last_changed=$lockstep_changed
  if [ "$lockstep_changed" -eq 0 ]; then
    return
  fi

  if [ "$lockstep_changed" -ne "$lockstep_total" ]; then
    echo "$lockstep_component is waiting for all flavor pins to update ($lockstep_changed/$lockstep_total changed)" >&2
    ready=false
  fi
  if [ "$lockstep_mismatched" = "true" ]; then
    echo "$lockstep_component flavor versions do not match" >&2
    ready=false
  fi
}

check_lockstep zarf \
  'tasks.yaml|.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default' \
  'flavors/upstream.yaml|.package.create.set.agent_image_tag' \
  'flavors/registry1.yaml|.package.create.set.agent_image_tag' \
  'flavors/registry1-arm64.yaml|.package.create.set.agent_image_tag' \
  'flavors/unicorn.yaml|.package.create.set.agent_image_tag' \
  'releaser.yaml|.packages[0].flavors[0].version' \
  'releaser.yaml|.packages[0].flavors[1].version' \
  'releaser.yaml|.packages[0].flavors[2].version' \
  'releaser.yaml|.packages[1].flavors[0].version' \
  'releaser.yaml|.packages[1].flavors[1].version' \
  'releaser.yaml|.packages[1].flavors[2].version' \
  'releaser.yaml|.packages[2].flavors[0].version' \
  'releaser.yaml|.packages[2].flavors[1].version' \
  'releaser.yaml|.packages[2].flavors[2].version'

check_lockstep registry \
  'flavors/upstream.yaml|.package.create.set.registry_image_tag' \
  'flavors/registry1.yaml|.package.create.set.registry_image_tag' \
  'flavors/registry1-arm64.yaml|.package.create.set.registry_image_tag' \
  'flavors/unicorn.yaml|.package.create.set.registry_image_tag'

check_lockstep socat \
  'flavors/upstream.yaml|.package.create.set.proxy_image_tag' \
  'flavors/registry1.yaml|.package.create.set.proxy_image_tag' \
  'flavors/registry1-arm64.yaml|.package.create.set.proxy_image_tag' \
  'flavors/unicorn.yaml|.package.create.set.proxy_image_tag'

check_lockstep gitea \
  'flavors/upstream.yaml|.package.create.set.gitea_image | split(":")[-1]' \
  'flavors/registry1.yaml|.package.create.set.gitea_image | split(":")[-1]' \
  'flavors/registry1-arm64.yaml|.package.create.set.gitea_image | split(":")[-1]' \
  'flavors/unicorn.yaml|.package.create.set.gitea_image | split(":")[-1]'
gitea_images_changed=$last_changed

old_chart=$(read_value "$base_ref" components/common/zarf.yaml '.components[] | select(.name == "git-server") | .charts[0].version')
new_chart=$(read_value worktree components/common/zarf.yaml '.components[] | select(.name == "git-server") | .charts[0].version')
if [ "$old_chart" != "$new_chart" ] && [ "$gitea_images_changed" -eq 0 ]; then
  echo "gitea chart is waiting for all flavor images" >&2
  ready=false
fi

if [ "$ready" != "true" ]; then
  exit 1
fi

echo "Renovate component versions are ready"
