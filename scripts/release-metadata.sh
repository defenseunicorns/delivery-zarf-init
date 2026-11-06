#!/bin/sh
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -eu

case "${1:?expected matrix or inventory}" in
  matrix)
    # shellcheck disable=SC2016
    yq -o=json -I=0 '
      {"include": [.packages[] | . as $package | .flavors[] |
        {"package": $package.name, "flavor": .name}]}
    ' releaser.yaml
    ;;
  inventory)
    for flavor in $(yq -r '.packages[].flavors[].name' releaser.yaml | sort -u); do
      repository=$(FLAVOR_NAME="$flavor" yq -r '.registries[env(FLAVOR_NAME)] // ""' release-registries.yaml)
      if [ -z "$repository" ]; then
        echo "no publish repository configured for $flavor" >&2
        exit 1
      fi
    done
    # shellcheck disable=SC2016
    yq -n -o=json -I=0 '
      load("releaser.yaml") as $releaser |
      load("release-registries.yaml") as $config |
      {"artifacts": [$releaser.packages[] | . as $package |
        .flavors[] | . as $flavor |
        {"package": $package.name, "flavor": $flavor.name,
         "version": $flavor.version,
         "ociRef": ("oci://" + ($config.registries[$flavor.name] // "") + "/" +
           $package.name + ":" + $flavor.version + "-" + $flavor.name),
         "signingTag": $flavor.version}]}
    '
    ;;
  *)
    echo "expected matrix or inventory" >&2
    exit 1
    ;;
esac
