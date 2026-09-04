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

Each package has an independent entry in `releaser.yaml`. OCI tags use
`<zarf-version>-uds.<revision>-<flavor>`, and GitHub release tags use
`<package>-<zarf-version>-uds.<revision>-<flavor>`.

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
releaser.yaml              independent uds-pk versions for all three packages
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
uds run publish-package                      # create both arches, test amd64, then publish
uds run test-zarf-values                     # validate schemas and passthrough
uds run pre-commit-all                       # run repository checks
```

Select another package or flavor with task variables:

```bash
uds run test-install --set PACKAGE=init-agent-only --set FLAVOR=registry1
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
- install-test all three packages on amd64 across all three flavors on trusted pull requests;
- create and install-test the public upstream flavor on fork and Dependabot pull requests; and
- validate Zarf values schemas and passthrough.

Renovate groups runtime dependencies by component. Its PRs stop before package tests until every
flavor pin for the changed component is present at the same normalized version. Gitea chart-only
updates also wait for the flavor images.

Zarf updates also advance every package and flavor in `releaser.yaml`. Registry, Socat, and Gitea
updates do not create a package release by themselves; they ship with the next Zarf release unless
an affected package receives an explicit `-uds.N` revision bump.

Pushes to `main` evaluate each package and flavor independently with package-scoped `uds-pk`.
A release job creates both architectures, install-tests amd64, publishes both, and only then creates
the package-specific GitHub release tag. The tag is the completion marker, so a failed partial
publication is retried on the next push.

The registry can be backed by S3-compatible object storage; see
[`docs/s3-backed-registry.md`](./docs/s3-backed-registry.md).
