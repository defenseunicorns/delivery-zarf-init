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

The repository publishes one consolidated GitHub release representing the package/flavor versions
declared in `releaser.yaml`. The highest declared version is the consolidated release tag. Merging a
new highest version validates Zarf values, then creates that tag and starts publication automatically.
Only entries carrying the new version are published; older entries remain available under their
existing OCI tags.

Each published artifact also receives a package/flavor Git tag for traceability and compatibility
with `uds-pk`, but those tags do not create GitHub Release records. The consolidated GitHub Release
is created only after every selected artifact succeeds, and its notes inventory all current
package/flavor OCI versions. Automation maps conventional pull request titles to the release-note
labels `enhancement`, `bug`, `documentation`, and `breaking-change`; Renovate pull requests appear
under Dependencies regardless of title. Unmatched pull requests appear under Other Changes. A `!`
in a non-Renovate conventional title places the pull request under Breaking Changes, but explicit
upgrade guidance is still required because generated notes cannot infer migration details from a
title. GitHub selects the Latest release automatically.

## Architecture decisions

Record significant design decisions in [`adr/`](./adr/) using [`adr/template.md`](./adr/template.md).
