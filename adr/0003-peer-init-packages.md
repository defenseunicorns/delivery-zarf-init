# 3. Use peer init packages with coordinated releases

Date: 2026-09-03

## Status

Accepted; supersedes [ADR 0002](./0002-flavors-packages-vendored-upstream.md)

## Context

This repository releases three Zarf init packages across upstream, Iron Bank, and Chainguard FIPS
flavors. They bootstrap Zarf before UDS Core exists and must remain aligned with upstream Zarf.

## Decision

- Store the packages under `packages/init/`, `packages/init-agent-only/`, and
  `packages/init-gitea/`; keep shared definitions under `components/` and image inputs under
  `flavors/`.
- Vendor upstream package definitions at one `ZARF_SOURCE_VERSION`. Require every agent flavor and
  package release version to match it.
- Keep values passthrough, health checks, and chart overrides in shared imports without forking
  upstream components.
- Release each package and flavor through its own `releaser.yaml` entry and package-scoped `uds-pk`
  check. Build both architectures and install-test amd64 before publishing.
- Group Renovate updates by component. Zarf updates advance every release entry; Registry, Socat,
  and Gitea updates ship with the next Zarf release unless an affected package receives a
  package-only `-uds.N` revision.
- Omit the UDS bundle and `Package` CR because these packages run before UDS Core.

## Consequences

- Package names, directories, OCI repositories, and release entries match.
- A delayed hardened image blocks only its component update.
- Failed publication remains retryable until its package-specific GitHub release tag exists.
- Repository-level validation and release wrappers must support three peer package roots.
