# Zarf Init packages for Delivery

Customized [Zarf init](https://docs.zarf.dev/ref/init-package/) packages, published as a matrix of
**image flavor** × **component set** to `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init`.

The component definitions are imported from the upstream
[zarf-dev/zarf `packages/`](https://github.com/zarf-dev/zarf/tree/main/packages) tree at the pinned
Zarf version, so the packages track upstream behavior; images are swapped per flavor and shared
config is layered on top.

## Flavors

- `upstream` — upstream defaults (ghcr.io / Docker Hub)
- `registry1` — Iron Bank (`registry1.dso.mil`)
- `unicorn` — Chainguard FIPS images from the Defense Unicorns Chainguard org (`cgr.dev/defenseunicorns.com`)

### Unicorn flavor images

| Component | Image |
|-----------|-------|
| agent | `ghcr.io/zarf-dev/zarf/agent` (upstream image, already Chainguard-built) |
| registry | `cgr.dev/defenseunicorns.com/distribution-fips` |
| registry proxy | `cgr.dev/defenseunicorns.com/socat-fips` |
| gitea | `cgr.dev/defenseunicorns.com/gitea-fips` |

Pulling the unicorn flavor images requires Chainguard registry access
(`chainctl auth login` or a pull token for `cgr.dev/defenseunicorns.com`).

Iron Bank ships arm64 images under separate `-arm64` tags rather than multi-arch manifests, which is
why `zarf-config/registry1-arm64.yaml` exists alongside `zarf-config/registry1.yaml`.

## Packages (component set)

- `default` — injector, registry, agent
- `agent-only` — agent only (for clusters using an external registry)
- `gitea` — injector, registry, agent, gitea

## Tags

For Zarf version `x.x.x`, each package publishes `x.x.x-<flavor>` plus a package suffix:

| | default | agent-only | gitea |
|--|---------|-----------|-------|
| upstream | `x.x.x-upstream` | `x.x.x-upstream-agent-only` | `x.x.x-upstream-gitea` |
| registry1 | `x.x.x-registry1` | `x.x.x-registry1-agent-only` | `x.x.x-registry1-gitea` |
| unicorn | `x.x.x-unicorn` | `x.x.x-unicorn-agent-only` | `x.x.x-unicorn-gitea` |

Tags are multi-arch (amd64 + arm64).

## Deploy

```bash
zarf package deploy oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.81.1-unicorn --confirm
```

## Development

Local tasks are run with the [UDS CLI](https://github.com/defenseunicorns/uds-cli) (maru). The
default loop — vendor upstream packages, build the `gitea` (superset) package, then deploy and test
on a fresh uds-k3d cluster:

```bash
uds run
```

Useful tasks (see `uds run --list-all`):

```bash
uds run dev                                      # rebuild + redeploy on the existing cluster
uds run create --set FLAVOR=registry1            # build one package (PACKAGE=gitea by default)
uds run create-all --set FLAVOR=unicorn          # build all three packages
uds run remove                                   # remove the deployed init package
uds run test:all                                 # health checks + agent mutation + values assertions
uds run cleanup                                  # tear down the uds-k3d cluster and artifacts
uds run lint:all                                 # full lint suite (matches CI)
uds run pre-commit-all                           # pre-commit hooks + SPDX header fix
```

Only one init package can exist per cluster: redeploying the same package upgrades in place, but
switching packages needs `uds run remove` first (or a fresh cluster). To compare flavors or packages
side by side, use separate clusters: `uds run --set CLUSTER_NAME=zarf-unicorn --set FLAVOR=unicorn`
(note cluster creation switches the current kubeconfig context, so start clusters one at a time).

Building the `registry1` flavor requires Iron Bank credentials (`uds zarf tools registry login registry1.dso.mil`),
and `unicorn` requires Chainguard access as noted above.

The registry can be backed by S3-compatible object storage; see
[docs/s3-backed-registry.md](./docs/s3-backed-registry.md).
