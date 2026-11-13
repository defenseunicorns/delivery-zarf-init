# Upgrading

## v0.87.0-uds.2

Registry values are configured independently for the bootstrap and final registry phases. Values
under `zarf-registry` no longer apply to the `zarf-seed-registry` component.

Before upgrading from an earlier release, add `zarf-seed-registry` to the values supplied to the
init package. Copy the persistence, storage backend, service account, and replica settings that the
seed registry needs during bootstrap. Keep phase-specific settings separate; the seed registry can
use fewer replicas than the final registry.

For example, an S3-backed registry configuration should explicitly configure both phases:

```yaml
zarf-seed-registry: &registry-storage
  replicaCount: 1
  persistence:
    enabled: false
  serviceAccount:
    name: docker-registry-sa
    create: true
  extraEnvVars:
    - name: REGISTRY_STORAGE
      value: s3
    - name: REGISTRY_STORAGE_S3_REGION
      value: <region>
    - name: REGISTRY_STORAGE_S3_BUCKET
      value: <bucket>

zarf-registry:
  <<: *registry-storage
  replicaCount: 2
```

Deploy variables such as `REGISTRY_PVC_ENABLED` remain upstream Zarf inputs and are not renamed by
this change. The migration applies to Helm values supplied through `--values`, `--set-values`, or a
UDS package `values` map.
