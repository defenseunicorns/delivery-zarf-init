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
and per-artifact version entries otherwise follow the UDS package template and `uds-pk` conventions.
Document any additional exception in [`docs/justifications.md`](./docs/justifications.md).

Renovate advances all package versions when Zarf updates. For a package-only release, use the bump
task to assign the next repository-wide `-uds.N` revision to every affected package/flavor:

```bash
uds run bump-release --set PACKAGE=init --set FLAVOR=unicorn
```

Use `all` to update a wider set in one release, for example
`--set PACKAGE=all --set FLAVOR=unicorn`. Run the task once per release so every selected entry gets
the same revision; running it again assigns another revision. Unselected entries retain their
existing OCI versions. Pull request CI checks artifact-affecting changes and reports any
package/flavor entries that still need the release bump. Flavor files require that flavor across all
packages, package files require that package across all flavors, and shared component or package
build task changes require all package/flavor entries. If a change spans both package and flavor
scopes, use `--set PACKAGE=all --set FLAVOR=all`; this may bump unaffected entries.

Registry, Socat, and Gitea updates require this explicit bump unless they are part of a Zarf update.
Zarf updates reset every package/flavor entry to `-uds.0` through Renovate.

## Releases

Merging a new highest version in `releaser.yaml` publishes changed artifacts and creates or
advances one GitHub Release for that Zarf version. See the [release process](./README.md#ci-and-release-behavior)
for publication, signing, and release-note details.

Release automation uses Go commands in `cmd/`. After changing them, run
`uds run pre-commit-all` for formatting, linting, vetting, and tests. PR CI runs the same Go hooks.
Renovate groups Go toolchain and hook updates with CI dependencies under the existing automerge
policy. See [ADR 0005](./adr/0005-release-automation-language.md) for the toolchain decision.

## Architecture decisions

Record significant design decisions in [`adr/`](./adr/) using [`adr/template.md`](./adr/template.md).
