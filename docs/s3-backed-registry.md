# S3-backed registry

**Prefer S3 backing for the registry whenever the environment provides object storage.** Object
storage removes the PVC as a single point of failure, lets the registry scale horizontally without
RWX volumes, and survives node loss with no data migration. The PVC default exists for environments
with no object store, not as the recommended posture.

The registry deployed by the `default` and `gitea` packages is configured for S3 entirely at deploy
time through the Zarf values passthrough. Verified end-to-end against the MinIO instance shipped by
[uds-k3d](https://github.com/defenseunicorns/uds-k3d)'s dev stack; for production use AWS S3 with
IRSA, or any S3-compatible store backed by stable out-of-cluster storage.

## Local dev flow (uds-k3d MinIO)

```bash
uds run test:uds-cluster         # uds-k3d cluster; tests/uds-config.yaml provisions the bucket + user
uds run package:create
uds run test:s3                  # deploys with tests/zarf-values-s3.yaml
uds run test:workload && uds run test:all
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
