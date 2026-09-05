# S3-backed registry

**Prefer S3 backing for the registry whenever the environment provides object storage.** Object
storage removes the PVC as a single point of failure, lets the registry scale horizontally without
RWX volumes, and survives node loss with no data migration. The PVC default exists for environments
with no object store, not as the recommended posture.

The registry deployed by the `init` and `init-gitea` packages is configured for S3 at deploy time
through the Zarf values passthrough. Use AWS S3 with IRSA, or an S3-compatible store backed by
stable out-of-cluster storage.

Create the selected package with the standard task, then supply an environment-specific values
file when deploying the resulting artifact:

```bash
uds run create-dev-package --set PACKAGE=init --set FLAVOR=upstream
uds zarf package deploy <zarf-init-artifact> --values <s3-values-file> --confirm
```

Do not commit static access keys. Reference a pre-existing Kubernetes Secret from
`zarf-registry.extraEnvVars` or use workload identity where the object store supports it.

## Required settings

Two settings beyond the usual `REGISTRY_STORAGE_S3_*` environment variables are load-bearing:

- **`persistence.enabled: false`** — the registry chart injects the filesystem storage driver when
  persistence is on, and distribution fails if two storage drivers are configured.
- **`REGISTRY_STORAGE_REDIRECT_DISABLE: "true"`** — use this when cluster nodes cannot resolve the
  S3 endpoint advertised by presigned URLs, so the registry proxies blob transfers instead.

## Authentication

- **Static keys:** provide `REGISTRY_STORAGE_S3_ACCESSKEY` and
  `REGISTRY_STORAGE_S3_SECRETKEY` through `valueFrom.secretKeyRef` entries in `extraEnvVars`.
- **IAM role/service account:** set the upstream service-account creation and annotation values,
  then omit static key variables so the AWS SDK credential chain uses workload identity.
