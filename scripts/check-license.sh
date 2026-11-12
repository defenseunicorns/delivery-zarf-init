#!/usr/bin/env bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

export XDG_STATE_HOME="${XDG_STATE_HOME:-${TMPDIR:-/tmp}/uds-repository-lint}"
mkdir -p "$XDG_STATE_HOME"

readonly config_file=.license_config.yaml
license=$(uds zarf tools yq -r '.license' "$config_file")
copyright=$(uds zarf tools yq -r '.copyright' "$config_file")

if [ -z "$license" ] || [ "$license" = null ] || [ -z "$copyright" ] || [ "$copyright" = null ]; then
  echo "$config_file: license and copyright must both be configured" >&2
  exit 1
fi

args=(-check -l "$license" -s=only -c "$copyright")
while IFS= read -r pattern; do
  [ -n "$pattern" ] || continue
  case "$pattern" in
    \#*) continue ;;
  esac
  args+=(-ignore "$pattern")
done <.gitignore

while IFS= read -r pattern; do
  [ -n "$pattern" ] || continue
  args+=(-ignore "$pattern")
done < <(uds zarf tools yq -r '.ignore[]?' "$config_file")

addlicense "${args[@]}" .
