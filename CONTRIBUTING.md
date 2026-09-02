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

Run `uds run generate-zarf-values-schemas` after changing exposed values. The generator restores
native chart types that the pinned Zarf CLI cannot infer from templated values. `uds run
test-zarf-values` lints the schema, checks drift, and verifies rendered passthrough.

## Documentation

Significant design decisions are recorded as ADRs in [adr/](./adr/) using the
[template](./adr/template.md).
