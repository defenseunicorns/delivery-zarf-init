# Zarf Init for Delivery

This is a custom init package used by delivery with the following features:

- Registry1 agent image
  - Registry1 repo:
    - <https://repo1.dso.mil/dsop/opensource/defenseunicorns/zarf/zarf-agent>
- distribution/distribution image registry v3
  - Still in alpha
    - <https://github.com/distribution/distribution/releases/tag/v3.0.0-alpha.1>
  - Waiting on a registry1 image to be available
    - <https://repo1.dso.mil/dsop/opensource/docker/registry-v2/-/issues/112>
  - Supports s3 IRSA
- No Gitea
- No Logging Stack