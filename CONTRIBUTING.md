# Contributing to Delivery Zarf Init

This repository follows the general
[`uds-common` contributing guidance](https://github.com/defenseunicorns/uds-common/blob/main/CONTRIBUTING.md),
including conventional commits and repository linting.

## Development workflow

Use the UDS CLI task runner from the repository root:

```bash
uds run test-install
uds run test-zarf-values
uds run pre-commit-all
```

Select a release with `PACKAGE` and `FLAVOR`. The valid package names are `init`,
`init-agent-only`, and `init-gitea`; the valid flavors are `upstream`, `registry1`, and `unicorn`.

```bash
uds run test-install --set PACKAGE=init-gitea --set FLAVOR=upstream
```

Run `uds run generate-zarf-values-schemas` after changing exposed Zarf values. The generator
restores native chart types that the pinned Zarf CLI cannot infer from templated values.

## Package scope

These artifacts are `ZarfInitConfig` packages deployed before UDS Core. A UDS bundle and UDS
`Package` CR are therefore intentionally absent. Package creation, task entry points, validation,
and package-scoped releases otherwise follow the UDS package template and `uds-pk` conventions.
Document any additional exception in [`docs/justifications.md`](./docs/justifications.md).

Renovate advances package versions with Zarf updates. For a package-only release, increment
`-uds.N` for every flavor of each affected package in `releaser.yaml`.

## Architecture decisions

Record significant design decisions in [`adr/`](./adr/) using [`adr/template.md`](./adr/template.md).
