# 2. Publish init packages as a flavor × variant matrix built from vendored upstream definitions

Date: 2026-07-24

## Status

Accepted

## Context

The repository previously published a single amd64 init package by importing the upstream init
package from OCI and re-tagging, with Iron Bank images swapped in. We need multiple image sources
(upstream, Iron Bank, Chainguard FIPS), multiple component sets (with/without registry, with/without
gitea), and multi-arch support — without forking upstream behavior.

## Decision

- Vendor the upstream `zarf-dev/zarf` `packages/` tree at the pinned `ZARF_VERSION` (sparse git
  clone into gitignored `.zarf-src/`) and compose variants from it via zarf component imports, so
  packages track upstream behavior exactly and only images are swapped.
- Model **flavors** (image source: `upstream`, `registry1`, `unicorn`) as zarf flavors driven by
  per-flavor `zarf-config/` files, and **variants** (component set: `default`, `agent-only`,
  `gitea`) as separate package definitions under `variants/`.
- Publish tags as `<zarf-version>-<flavor>[-<variant>]`, multi-arch (amd64 + arm64). Versioning
  tracks the upstream Zarf release; there is no independent semver or release-please.
- Expose chart configuration through the Zarf values passthrough (`values:` + per-chart
  `sourcePath`/`targetPath` with excludePaths guarding images, secrets, and security contexts), and
  gate deployments with component `healthChecks`.
- Constraint: gitea images must stay on the minor supported by the gitea chart vendored in upstream
  zarf (chart 12.3.0 → gitea 1.24.x); a renovate packageRule enforces this until a ZARF_VERSION bump
  changes the vendored chart.

## Consequences

- Upstream zarf behavior is inherited byte-for-byte at a pinned version; upgrades are a single
  `ZARF_VERSION` bump (renovate-driven) that atomically moves the CLI, the vendored definitions, and
  the agent image.
- The `.zarf-src/` vendor step is required before any build (`uds run vendor`).
- Some uds-common conventions (bundles, Package CR, uds-pk releases, callable-test/publish) do not
  apply; local equivalents mirror their shape where possible.
