# S3-backed registry

The registry deployed by the `default` and `gitea` variants can store its blobs in any S3-compatible
object store instead of a PVC, configured entirely at deploy time through the Zarf values
passthrough. Verified end-to-end against the MinIO instance shipped by
[uds-k3d](https://github.com/defenseunicorns/uds-k3d)'s dev stack.

## Local dev flow (uds-k3d MinIO)

```bash
uds run setup-uds-cluster   # uds-k3d cluster; tests/uds-config.yaml provisions the bucket + user
uds run create
uds run deploy-s3           # deploys with tests/zarf-values-s3.yaml
uds run deploy-workload && uds run test:all
```

Bucket and user are provisioned via deploy-time overrides on the `uds-k3d-dev` package
([tests/uds-config.yaml](../tests/uds-config.yaml)); note those overrides *replace* the default
`uds` bucket/user arrays. The registry is pointed at the bucket via
[tests/zarf-values-s3.yaml](../tests/zarf-values-s3.yaml).

## Required settings

Two settings beyond the usual `REGISTRY_STORAGE_S3_*` env vars are load-bearing:

- **`persistence.enabled: false`** — the registry chart injects the filesystem storage driver when
  persistence is on, and distribution panics if two storage drivers are configured.
- **`REGISTRY_STORAGE_REDIRECT_DISABLE: "true"`** — the S3 driver redirects blob pulls to presigned
  URLs by default; containerd on the node cannot resolve an in-cluster MinIO hostname, so the
  registry must proxy blobs. (Not needed if the S3 endpoint is resolvable from the nodes, e.g. real
  AWS S3.)

## Auth

- **Static keys** (MinIO, or any S3-compatible store): `REGISTRY_STORAGE_S3_ACCESSKEY` /
  `SECRETKEY` env vars, ideally via `valueFrom.secretKeyRef` in `extraEnvVars`.
- **IAM role / service account (IRSA, real AWS S3 only)**: set the upstream deploy variables
  `REGISTRY_CREATE_SERVICE_ACCOUNT=true` and `REGISTRY_SERVICE_ACCOUNT_ANNOTATIONS` (the
  `eks.amazonaws.com/role-arn` annotation) and omit the key env vars — the AWS SDK credential chain
  picks up the web identity token. uds-k3d's MinIO has no OIDC/STS federation, so this path does not
  apply there.
