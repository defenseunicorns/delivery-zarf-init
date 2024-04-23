# Zarf Init for Delivery

This repository contains zarf init packages for delivery.

## zarf-init-registry1

- Registry1 agent image
  - Registry1 repo:
    - <https://repo1.dso.mil/dsop/opensource/defenseunicorns/zarf/zarf-agent>
- distribution/distribution image registry v3
  - Still in alpha
    - <https://github.com/distribution/distribution/releases/tag/v3.0.0-alpha.1>
  - Waiting on a registry1 image to be available
    - <https://repo1.dso.mil/dsop/opensource/docker/registry-v2/-/issues/112>
  - Supports s3 IRSA

## zarf-init-gitea-registry1

- same as zarf-init-registry1, but also includes gitea
