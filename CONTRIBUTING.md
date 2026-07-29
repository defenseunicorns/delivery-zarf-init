# Welcome to the Delivery Zarf Init packages

Thank you for your interest in this Defense Unicorns repository!

This repository broadly follows the contributing guidelines of the Unicorn Delivery Service — see the
[uds-common CONTRIBUTING.md](https://github.com/defenseunicorns/uds-common/blob/main/CONTRIBUTING.md)
for the general workflow (conventional commits, PR titles checked by commitlint, etc.).

## Developer workflow

Local development uses the [UDS CLI](https://github.com/defenseunicorns/uds-cli) task runner — see
the [README Development section](./README.md#development) for the task reference. The typical loop:

```bash
uds run                 # vendor, build, deploy, and test on a fresh uds-k3d cluster
uds run dev             # rebuild + redeploy on the existing cluster while iterating
uds run pre-commit-all  # pre-commit hooks + SPDX license header fix before pushing
```

Note this repository is *not* a UDS package: it publishes customized [Zarf init
packages](https://docs.zarf.dev/ref/init-package/) and does not integrate with the uds-operator, so
some uds-common conventions (bundles, the Package CR, uds-pk releases) intentionally do not apply.
Versioning tracks the upstream Zarf release rather than release-please.

## Zarf values schemas

`packages/*/zarf-values.schema.json` are generated (`uds zarf dev generate-schema packages/<name> -u`
with the flavor template sets). The generator types templated fields as strings; re-apply structured
types (extraEnvVars array, persistence.enabled boolean, affinity/nodeSelector objects, tolerations
arrays) after regenerating.

## Documentation

Significant design decisions are recorded as ADRs in [adr/](./adr/) using the
[template](./adr/template.md).
