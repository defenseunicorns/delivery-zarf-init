# 2. Publish init packages as a flavor x package matrix built from vendored upstream definitions

Date: 2026-07-24

## Status

Superseded by [ADR 0003](./0003-peer-init-packages.md)

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
- Publish tags as `<zarf-version>-uds.N[-<package>]-<flavor>`, multi-arch. Public flavors publish under
  `ghcr.io/defenseunicorns/delivery-zarf-init`; `unicorn` publishes under
  `ghcr.io/defenseunicorns/packages/private/delivery-zarf-init`.
- Each flavor resolves its own Zarf source version: upstream tracks the upstream Zarf agent image,
  registry1 tracks the Iron Bank Zarf agent image, and unicorn tracks the Chainguard FIPS Zarf agent
  image. Renovate updates `zarf_source_version` and the agent image tag together in each flavor
  config. Package revisions are tracked in `releaser.yaml`; local overrides require matching
  `ZARF_VERSION` and `PACKAGE_VERSION` values.
- Gitea runs on a newer chart than upstream zarf's pin (overridden in common) because Chainguard
  only ships gitea >=1.26; a renovate packageRule keeps the gitea images on the chart-supported
  minor across all flavors.
- Expose chart configuration through the Zarf values passthrough (per-chart `sourcePath` mappings
  with excludePaths guarding images, secrets, and security contexts; generated schemas) and gate
  deployments with component `healthChecks`. Upstream's init package has not adopted values yet;
  this is a deliberate bet on where zarf is heading, and the local mappings retire if upstream
  adds its own.

## Consequences

- Upstream zarf behavior is inherited from the version resolved for each flavor; registry1 can lag
  without blocking upstream or unicorn publishes.
- Tests follow affected source paths, while publishing follows only flavor versions changed in
  `releaser.yaml`; CI-only changes do not create package releases.
- The `.zarf-src/` vendor step is required before any build (`uds run package:vendor`) and re-vendors
  automatically when the cached source version does not match the resolved flavor version.
- Some uds-common conventions (bundles, Package CR, uds-pk releases, callable-test/publish) do not
  apply; local equivalents mirror their shape where possible.
- The values feature is alpha in zarf; the e2e values assertions in `test:all` are the tripwire for
  behavior changes on zarf version bumps.
