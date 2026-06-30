# Zarf Init packages for Delivery

Customized [Zarf init](https://docs.zarf.dev/ref/init-package/) packages, published as a matrix of
**image flavor** × **component set** to `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init`.

## Flavors

- `upstream` — upstream defaults (ghcr.io / Docker Hub)
- `registry1` — Iron Bank (`registry1.dso.mil`)
- `unicorn` — Chainguard FIPS (`cgr.dev/defenseunicorns.com`)

## Variants (component set)

- `default` — injector, registry, agent
- `agent-only` — agent only
- `gitea` — injector, registry, agent, gitea

## Tags

For Zarf version `x.x.x`, each variant publishes `x.x.x-<flavor>` plus a variant suffix:

| | default | agent-only | gitea |
|--|---------|-----------|-------|
| upstream | `x.x.x-upstream` | `x.x.x-upstream-agent-only` | `x.x.x-upstream-gitea` |
| registry1 | `x.x.x-registry1` | `x.x.x-registry1-agent-only` | `x.x.x-registry1-gitea` |
| unicorn | `x.x.x-unicorn` | `x.x.x-unicorn-agent-only` | `x.x.x-unicorn-gitea` |

## Deploy

```bash
zarf package deploy oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.80.0-unicorn --confirm
```
