#!/usr/bin/env bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

export XDG_STATE_HOME="${XDG_STATE_HOME:-${TMPDIR:-/tmp}/uds-repository-lint}"
mkdir -p "$XDG_STATE_HOME"

commands_file=$(mktemp)
trap 'rm -f "$commands_file"' EXIT

lint_commands() {
  local file=$1
  local query=$2
  local expression_pattern=$3
  local expression_replacement=$4
  local command

  uds zarf tools yq -r -0 "$query" "$file" >"$commands_file"
  printf 'Processing %s...\n' "$file"
  while IFS= read -r -d '' command; do
    command=$(printf '%s' "$command" | sed -E "s|$expression_pattern|$expression_replacement|g")
    printf '%s\n' "$command" | shellcheck -x \
      -e SC2016,SC2050,SC2001,SC2002,SC2181 -
  done <"$commands_file"
}

task_query='.tasks[].actions[]? | select(.cmd) | "#!/bin/" + (.shell.linux // "sh") + "\n" + .cmd'
for file in tasks.yaml tasks/*.yaml; do
  [ -f "$file" ] || continue
  lint_commands "$file" "$task_query" '\$\{\{[^}]*\}\}' maru-expression
done

zarf_query='.components[].actions | (.onCreate,.onDeploy,.onRemove) | (.before,.after,.onSuccess,.onFailure)[]? | select(.cmd) | "#!/bin/" + (.shell.linux // "sh") + "\n" + .cmd'
while IFS= read -r -d '' file; do
  lint_commands "$file" "$zarf_query" '\{\{[^}]*\}\}' zarf-expression
done < <(find . \
  \( -type d \( -name .git -o -name .zarf-src -o -name .terraform -o -name build -o -name dist -o -name tmp -o -name vendored \) -prune \) \
  -o -type f -name zarf.yaml -print0)
