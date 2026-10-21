#!/bin/sh

# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -eu
# shellcheck disable=SC3040
(set -o pipefail 2>/dev/null) && set -o pipefail

check=${1:-false}
if [ "$#" -gt 1 ]; then
  echo "usage: $0 [true|false]" >&2
  exit 2
fi
case "$check" in
  true | false) ;;
  *)
    printf 'unsupported check mode: %s\n' "$check" >&2
    exit 2
    ;;
esac

settings=$(mktemp)
generated=""

cleanup() {
  rm -f "$settings"
  if [ -n "$generated" ]; then
    rm -f "$generated"
  fi
}
trap cleanup 0
trap 'exit 1' 1 2 3 15

uds zarf tools yq -r \
  '.package.create.set | to_entries[] | select(.key != "pkg_version") | [.key, (.value | tostring)] | @tsv' \
  flavors/upstream.yaml > "$settings"

set --
tab=$(printf '\t')
while IFS="$tab" read -r key value; do
  uppercase_key=$(printf '%s' "$key" | tr '[:lower:]' '[:upper:]')
  set -- "$@" --set "${uppercase_key}=${value}"
done < "$settings"

found=false
stale=false

for values_file in packages/*/zarf-values.yaml; do
  [ -e "$values_file" ] || continue
  found=true
  package_dir=${values_file%/*}
  schema_file="${package_dir}/zarf-values.schema.json"
  generated=$(mktemp "${schema_file}.tmp.XXXXXX")

  if [ "$check" = "true" ]; then
    action=Checking
  else
    action=Generating
  fi
  printf '%s %s\n' "$action" "$schema_file"

  ZARF_CONFIG=flavors/upstream.yaml uds zarf dev generate-schema "$package_dir" \
    --flavor upstream \
    "$@" \
    --delete-not-found \
    --no-color \
    --log-level error > "$generated"

  if [ ! -s "$generated" ]; then
    printf 'ERROR: generated schema is empty: %s\n' "$schema_file" >&2
    exit 1
  fi
  if ! uds zarf tools yq -p=json -oy -e \
    'type == "!!map" and (.properties | type == "!!map")' \
    "$generated" > /dev/null 2>&1; then
    printf 'ERROR: generated schema is invalid: %s\n' "$schema_file" >&2
    exit 1
  fi

  if [ "$check" = "true" ]; then
    if ! cmp -s "$schema_file" "$generated"; then
      printf 'ERROR: %s is stale\n' "$schema_file" >&2
      stale=true
    fi
    rm -f "$generated"
  else
    chmod 0644 "$generated"
    mv "$generated" "$schema_file"
  fi
  generated=""
done

if [ "$found" = "false" ]; then
  echo "no zarf-values.yaml files found" >&2
  exit 1
fi

if [ "$stale" = "true" ]; then
  echo "Run: uds run generate-zarf-values-schemas" >&2
  exit 1
fi
