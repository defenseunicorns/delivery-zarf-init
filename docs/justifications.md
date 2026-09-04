# Zarf Init package exceptions

These `ZarfInitConfig` packages establish Zarf before UDS Core exists.

## Peer package roots

The validator requires one root `zarf.yaml`; this repository has three roots under `packages/`.
Running it from a package directory therefore misreports shared root files as missing. The
`license-docs`, `ci-files`, `tool-configs`, `uds-tasks`, and `tests` findings are inapplicable:
governance, workflows, `renovate.json`, `releaser.yaml`, tasks, and tests exist at the repository
root. CI validates every package, flavor, and architecture through repository tasks.

## UDS integration

A UDS bundle and `Package` CR are inapplicable because the UDS Operator and supporting services do
not exist when an init package runs.

## Imported flavors

Package manifests import `components/zarf.yaml`, which defines every `only.flavor` selector. The
validator's `flavors` finding does not resolve that import; CI creates and tests all three flavors.

## Shared workflows

The standard auto-update and scan workflows assume one root package and cannot run source vendoring
for this matrix. Renovate coordinates component updates, the release workflow uses package-scoped
`uds-pk`, and release builds produce SBOMs.

## Tests

The `tests` finding expects package-local browser tests. Init tests are cluster-level tasks in
`tasks/test.yaml`, with fixtures under `tests/`, and run through the standard `test-install` entry
point.

## Vendored source

`.zarf-src/` is an ignored checkout of the pinned upstream Zarf version. Findings inside it are
upstream-owned.
