# 2. Publish init packages as a flavor x package matrix built from vendored upstream definitions

Date: 2026-07-24

## Status

Accepted

## Context

We need multiple image sources (upstream, Iron Bank, Chainguard FIPS), multiple component sets
(with/without registry, with/without gitea), and multi-arch support, without forking upstream Zarf
behavior.

## Decision

- Vendor the upstream `zarf-dev/zarf` `packages/` tree at the pinned `ZARF_VERSION` (sparse git
  clone into gitignored `.zarf-src/`) so packages track upstream behavior; images are swapped per
  flavor and deploy-time configuration is layered on top, nothing else is forked.
- Layer definitions the way UDS Core does:
  - `src/init/common/zarf.yaml` imports the upstream components and holds the shared config
    (values passthrough with excludePaths, healthChecks, gitea chart pin) exactly once.
  - `src/init/zarf.yaml` imports common and defines the flavors (`upstream`, `registry1`,
    `unicorn`), driven by per-flavor `zarf-config/` files.
  - `packages/{default,agent-only,gitea}/zarf.yaml` one-shot import components from `src/init`
    without redefining flavors; each publishes a separate artifact so consumers pull only the
    components they deploy.
- Publish tags as `<zarf-version>-<flavor>[-<package>]`, multi-arch. Versioning is anchored to
  Iron Bank as the lowest common denominator; there is no independent semver or release-please.
- Gitea runs on a newer chart than upstream zarf's pin (overridden in common) because Chainguard
  only ships gitea >=1.26; a renovate packageRule keeps the gitea images on the chart-supported
  minor across all flavors.
- Expose chart configuration through the Zarf values passthrough (per-chart `sourcePath` mappings
  with excludePaths guarding images, secrets, and security contexts; generated schemas) and gate
  deployments with component `healthChecks`. Upstream's init package has not adopted values yet;
  this is a deliberate bet on where zarf is heading, and the local mappings retire if upstream
  adds its own.

## Consequences

- Upstream zarf behavior is inherited at a pinned version; upgrades are a single renovate-driven
  `ZARF_VERSION` bump gated on Iron Bank publishing.
- The `.zarf-src/` vendor step is required before any build (`uds run vendor`).
- Some uds-common conventions (bundles, Package CR, uds-pk releases, callable-test/publish) do not
  apply; local equivalents mirror their shape where possible.
- The values feature is alpha in zarf; the e2e values assertions in `test:all` are the tripwire for
  behavior changes on zarf version bumps.
