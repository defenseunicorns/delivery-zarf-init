# Zarf Init packages for Delivery

This repository releases three customized [Zarf init](https://docs.zarf.dev/ref/init-package/)
packages. Each package is created for the `upstream`, `registry1`, and `unicorn` image flavors and
for amd64 and arm64.

The package definitions import the upstream `zarf-dev/zarf` package sources at the Zarf version
pinned by `ZARF_SOURCE_VERSION`. Shared values passthrough, health checks, and chart overrides live
under `components/`.

## Packages

| Release | Package contents | Public OCI repository |
|---|---|---|
| `init` | injector, registry, agent | `ghcr.io/defenseunicorns/delivery-zarf-init/init` |
| `init-agent-only` | agent | `ghcr.io/defenseunicorns/delivery-zarf-init/init-agent-only` |
| `init-gitea` | injector, registry, agent, Gitea | `ghcr.io/defenseunicorns/delivery-zarf-init/init-gitea` |

The `unicorn` packages publish beneath
`ghcr.io/defenseunicorns/packages/private/delivery-zarf-init/` instead.

## Flavors

- `upstream` uses the upstream Zarf, Docker Registry, Socat, and Gitea images.
- `registry1` uses Iron Bank images. Its arm64 images use explicit `-arm64` tags.
- `unicorn` uses Defense Unicorns Chainguard FIPS images.

Each package/flavor build has an entry in `releaser.yaml`. OCI tags use
`<zarf-version>-uds.<revision>-<flavor>`. One GitHub Release per Zarf version records the complete
package/flavor inventory. It points to the highest successfully
published `-uds.N` revision; earlier Git and OCI tags remain fixed.

Verify each artifact against the signing tag in the GitHub Release inventory, not necessarily the
Release's current tag. For example, an artifact published at `v0.89.0-uds.0` retains this identity
when the Release advances to `v0.89.0-uds.1`:

```yaml
keylessVerification:
  certificateIdentity: https://github.com/defenseunicorns/delivery-zarf-init/.github/workflows/release.yaml@refs/tags/v0.89.0-uds.0
  certificateOIDCIssuer: https://token.actions.githubusercontent.com
```

Packages published before the consolidated workflow retain their package- and flavor-specific
certificate identities. Their Git tags remain available for verification even where the old
package/flavor GitHub Release records were removed.

## Deploy

```bash
zarf package deploy \
  oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.84.0-uds.1-upstream \
  --confirm
```

## Repository layout

```text
packages/init/             default init package and values
packages/init-agent-only/  agent-only package and values
packages/init-gitea/       Gitea package and values
components/zarf.yaml       flavor-specific component composition
components/common/         shared component definitions and chart values
flavors/                   flavor and architecture-specific image sources
releaser.yaml              uds-pk versions for all package/flavor builds
release-registries.yaml    publish repository for each flavor
scripts/                   release readiness and metadata generation
tasks/                     init-specific create, deploy, test, and publish helpers
tests/                     package-level fixtures
.zarf-src/                 generated upstream Zarf package sources (gitignored)
```

This is a pre-UDS bootstrap repository. UDS bundles and the UDS `Package` CR are intentionally
inapplicable because the packages run before UDS Core exists. See
[`docs/justifications.md`](./docs/justifications.md).

## Development

The public task names follow the UDS package template:

```bash
uds run                                      # create and install-test init/upstream
uds run create-dev-package                   # create without an SBOM
uds run test-install                         # create, deploy, and verify on a fresh cluster
uds run publish-package                      # create both arches, test when deployable, then publish
uds run bump-release                         # assign the next UDS revision to selected artifacts
uds run test-zarf-values                     # check schemas, manifests, and generated Helm values
uds run test-release-automation              # test release automation
uds run pre-commit-all                       # run repository checks
```

`test-release-automation` runs `gofmt`, `go vet`, and `go test`, including fixture tests for release
readiness, version bumps, and release metadata. It uses a compatible local Go toolchain.
`pre-commit-all` adds `golangci-lint` formatting and linting. PR CI runs the hooks across all files
with the toolchain pinned in `go.mod`. Renovate tracks Go and lint-tool updates with CI dependencies.
See [ADR 0005](./adr/0005-release-automation-language.md) for the toolchain decision.

Select another package or flavor with task variables:

```bash
uds run create-dev-package --set PACKAGE=init-agent-only --set FLAVOR=registry1
uds run create-dev-package --set PACKAGE=init-gitea --set FLAVOR=unicorn --set ARCH=arm64
```

The package names accepted by `PACKAGE` are `init`, `init-agent-only`, and `init-gitea`.

Building `registry1` requires Iron Bank registry credentials. Building `unicorn` requires access to
`cgr.dev/defenseunicorns.com`.

The vendoring task checks out only the upstream Zarf package definitions needed by these packages.
All flavors use the shared `ZARF_SOURCE_VERSION` pin and validation requires their agent and release
versions to match it. Move a stale `.zarf-src` checkout aside after changing the shared pin; the task
does not switch source versions in place.

Test clusters use host ports `8080`, `8443`, and `6551` so they can coexist with a cluster using
the conventional `80`, `443`, and `6550` ports. Override the `K3D_*_PORT` task variables as needed.

## CI and release behavior

Pull requests:

- create all three packages for arm64 across all three flavors on trusted pull requests;
- install-test the registry-bearing packages across all three flavors on trusted pull requests;
- create every public upstream package and install-test its registry-bearing packages on forks and Dependabot;
- validate Zarf values schemas and passthrough; and
- require release bumps for artifact-affecting changes, including package build tasks.

Install tests run only when a pull request changes package definitions, components, flavors, tests,
tasks, or release metadata. Repository automation and documentation-only changes still create the
packages and validate Zarf values, but skip the cluster end-to-end matrix.

Renovate creates two normal dependency groups: package dependencies and CI dependencies. Package
dependencies include the charts, images, and release versions shipped in the Zarf packages; the CI
group contains test fixtures, workflow actions, development tools, and repository automation.
Renovate PRs stop before package tests until every flavor pin for a changed component is present at
the same normalized version. Gitea chart-only updates also wait for the flavor images.

Zarf updates also advance every package and flavor in `releaser.yaml`. Registry, Socat, and Gitea
updates require an explicit `-uds.N` revision bump for the affected package/flavor entries unless
they are included in a Zarf update that resets every entry to `-uds.0`.

Run `uds run bump-release --set PACKAGE=<package> --set FLAVOR=<flavor>` to select artifacts for a
package-only release. The task assigns one revision greater than the highest `-uds.N` for the current
Zarf version. `all` may be used for either selector. Renovate resets all entries to `-uds.0` when it
updates Zarf. Run the task only once for a release because running it again assigns another revision.
If one change affects a package across all flavors and another affects a flavor across all packages,
use `PACKAGE=all` and `FLAVOR=all`; the task may bump unaffected entries in this mixed-scope case.

When that change reaches `main`, automation validates Zarf values before creating one consolidated
tag from the highest version in `releaser.yaml`. The tag evaluates the package/flavor matrix from
that file and publishes entries at the new release version; entries still on earlier revisions remain
available under their existing OCI tags. Before building, ORAS checks
the selected OCI tag for both Zarf platforms (`multi/amd64` and `multi/arm64`); a complete artifact
is skipped, while a missing tag or architecture is published. Registry or authentication errors
fail the check instead of causing an existing artifact to be overwritten. An incomplete tag can be
repaired on retry, but does not enter a GitHub Release until both architectures pass verification.
New artifacts are tested, keyless-signed, and verified against the tag that published them.
Successful artifacts receive package/flavor Git tags, but not separate GitHub Releases.
`release-registries.yaml` supplies the publish repository for each flavor and the release inventory's
OCI references. Changing that routing requires a release bump for every package/flavor entry.

After every selected artifact succeeds, the workflow creates or advances the GitHub Release for
that Zarf version. It regenerates categorized notes from the previous Zarf version's final release
tag and records every package/flavor OCI reference and signing tag. Add hand-written upgrade guidance
outside the `generated-changelog` and `package-inventory` comment blocks; automation preserves text
outside those blocks. A rolling revision waits for its predecessor's GitHub Release before starting
artifact publication; if that predecessor fails, retry it before rerunning the waiting revision. An
older retry cannot move the Release backward. When Zarf advances, the prior Release stays at its
final revision and a new one begins. The Release URL changes with its associated tag, but earlier
Git and OCI tags remain available. Existing releases through v0.88.0 are not migrated. The first
new changelog starts from the preceding Zarf line's final published release tag. A failed release
can be retried from its tagged workflow.

Tag and release creation uses the `uds-release` GitHub Environment with the `APP_CLIENT_ID` variable
and `APP_PRIVATE_KEY` secret for the protected release GitHub App. Repository rulesets must allow
that App to create release tags while blocking tag updates and deletions. GitHub Release immutability
is confirmed disabled for this rollout and must stay disabled so the Release record can advance to
an existing tag; this setting is separate from Git tag protection. An App token is required because
tags created with `GITHUB_TOKEN` do not trigger the tag-based release workflow.

The registry can be backed by S3-compatible object storage; see
[`docs/s3-backed-registry.md`](./docs/s3-backed-registry.md).
