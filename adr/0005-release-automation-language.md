# 5. Adopt Go for repository-local release automation

Date: 2026-11-05

## Status

Accepted for the repository-local release helper. This record assigns toolchain ownership and
documents the CI controls for normal repository review; it does not introduce a separate approval
process. The branch implements the helper, but no release has been published through it.

## Context

The proposed rolling-release workflow needs a helper to compare versioned GitHub Releases,
identify the final legacy baseline, generate and preserve release-note sections, inventory OCI
artifacts, and create or advance one Release record after publication succeeds. An earlier draft
implemented that helper in JavaScript through `actions/github-script` and added a separately pinned
Node.js runtime for local tests and CI.

This repository has shell and YAML automation, but no established Go or Python toolchain. The
owning team reports a substantial, maintained Go ecosystem with vetted Zarf and UDS libraries,
while its Python ecosystem and vetted-library set are lighter. Sibling infrastructure repositories
owned by the team also use Go, including code that depends on Zarf libraries. That evidence makes
team maintenance and potential library reuse material selection factors; it does not mean this
helper needs a Zarf or UDS dependency today.

Concretely, [Defense Unicorns `pkg`](https://github.com/defenseunicorns/pkg) publishes common Go
modules for command execution, helpers, and OCI artifacts. [UDS CLI](https://github.com/defenseunicorns/uds-cli)
and [Zarf](https://github.com/zarf-dev/zarf) are Go projects whose code and package concepts are
familiar to this ecosystem. They are not blanket-approved dependencies: UDS CLI is primarily an
application, and importability, API stability, license compatibility, and transitive dependencies
must be checked for any specific package before use.

The Development Automation Best Practices guidance supports Go and Python for structured
automation and requires explicit ownership when introducing a toolchain; this record names that
owner. The helper runs in GitHub Actions and locally during tests. It is not shipped to package
users or intended as a shared CLI. Its work is sequential GitHub API and JSON/text processing;
package builds and workflow concurrency remain outside the helper. The workflow already uses `gh`
and `yq` elsewhere.

## Decision Drivers

| Factor | Go | Python |
| --- | --- | --- |
| Team fit and ownership | Existing team ecosystem, review experience, and vetted libraries reduce maintenance risk. This repository still needs its own Go CI profile. | New repository toolchain with less team-wide library vetting and ownership experience. |
| Task fit | Standard library covers JSON, text, subprocess, HTTP, and testing. More code is likely. | Standard library also fits; concise for API and text processing. |
| Execution and portability | CI builds and runs the helper on the same Ubuntu runner with a pinned Go compiler; local contributors need a compatible Go toolchain. A future target-specific build can run without an installed Go toolchain. | The source can run on hosts with a compatible interpreter and reproducibly provisioned environment. A packaged application could change that distribution model. |
| Dependencies and security | Start with standard library and existing `gh`/`yq`. Adopt a shared Zarf/UDS module only if it replaces meaningful custom logic and passes normal dependency review. | Standard-library runtime is also possible, but a Python profile adds uv and its lint/type/test tools to the team's maintenance set. |
| Longevity | Aligns future release or package metadata work with the team's maintained ecosystem. | Lower initial implementation effort, but creates a less-established support path for durable release automation. |
| Concurrency and performance | No decisive advantage: workflow serialization, not language concurrency, preserves release ordering. | Adequate for a few paginated API calls. |

The decisive factor is maintained organizational fit, not a claim that Go is inherently more secure
than Python. Both can meet the same correctness and supply-chain controls. Python's shorter
implementation does not outweigh the team's established Go review, tooling, and dependency
practices for long-lived release automation. Go's potential binary portability is not a deciding
benefit for this repository-local helper: CI builds it where it runs, and local tests still require
Go. A Go binary is specific to a target operating system and architecture, not universal. If the
helper later runs on other hosts or in a minimal image, build and test each supported target;
check whether cgo or system-library dependencies affect execution. Python is also portable at the
source level, but each target needs a compatible interpreter and environment unless the tool is
packaged differently. Review the Go choice against the actual implementation's size and dependencies.

Keeping JavaScript would retain the ready-made Octokit client but also retain a separate Node.js
source/test toolchain outside the repository's Go/Python guidance. Extending Bash would put
pagination, JSON state transitions, and note preservation into shell beyond its intended role.
Neither alternative reduces the long-term ownership burden enough to justify it here.

## Decision

Adopt Go as this repository's first maintained structured-language toolchain, initially for the
repository-local GitHub Release helper. Keep UDS task actions in POSIX `sh`, short CI glue in Bash,
and package-building tools unchanged. This is not a mandate to rewrite existing shell or a ban on
Python when a future task has a stronger Python-specific fit.

Use Go's standard library for release-state logic and authenticated GitHub HTTP calls. This avoids
adding a Go library or depending on the runner's `gh` version for release-record updates. Validate
tag names, API responses, and release-note markers before mutation; propagate API and authentication
errors; never print tokens. Provide the GitHub App token through `GH_TOKEN`. The helper must only
mutate the intended GitHub Release record; Git and OCI tags remain immutable. Preserve behavioral
coverage for legacy baselines, notes, ordering, Latest, and retry/failure cases.

The repository's Go profile for this toolchain is:

- Ownership: `@defenseunicorns/delivery-iac-coe` owns the Go version, direct dependencies,
  lint/security tools, vulnerability response, and replacement. Changes follow normal repository
  review, with no additional toolchain approval gate.
- Runtime: keep the `go` directive as the minimum source-compatibility version and declare an exact
  supported compiler in the `toolchain` directive. Renovate tracks the toolchain with other CI
  dependencies; advance the minimum only when code or dependencies require it. Require the
  toolchain version in the runner's tool cache before invoking SHA-pinned `actions/setup-go`, so
  the action does not fetch a compiler. Use `GOTOOLCHAIN=local` in CI so a version mismatch fails
  instead of downloading a toolchain.
- Dependencies: start with a standard-library-only module and reuse libraries already vetted by
  the team when they fit. For each new library, pin it in `go.mod`/`go.sum` and document due
  diligence on provenance, maintenance, API stability, license compatibility, and transitive
  dependencies in the PR. Do not add dependencies merely to justify the language choice. For
  disconnected CI, pre-provision or vendor modules, or use an internally controlled mirror.
- Quality: expose tests through `uds run test-release-automation`; run `gofmt`, `go test ./...`,
  and the repository-standard `golangci-lint` configuration. Unit tests must use a fake API
  transport or command runner and must not require GitHub credentials. Keep the compiler pin
  current through Renovate and review Go security advisories when it updates. Do not make
  `govulncheck` a required CI gate while this undistributed helper has no third-party Go modules;
  reconsider if dependencies are added or the helper is distributed.
- Distribution: build and execute repository-local source on the same CI runner; do not publish a
  binary solely for this helper. If it becomes a shared tool, follow the guidance's Section 3:
  publish precompiled, versioned binaries for each supported OS/architecture with verification,
  provenance, and appropriate SBOMs, and make disconnected availability explicit. Do not require
  target hosts to install Go or assume that one binary works on every target.

The helper uses only the Go standard library. Official `golangci-lint` pre-commit hooks at the
signed [`v2.15.0` release](https://github.com/golangci/golangci-lint/releases/tag/v2.15.0) provide
formatting, full linting, and configuration validation locally and in PR CI. Renovate groups hook
updates with CI dependencies.
Zarf and UDS Registry use `govulncheck`, so that vetted scanner is available if this helper later
needs dependency scanning. An ad hoc `govulncheck ./...` scan of the current helper found no
vulnerabilities, but the live database result is not a substitute for keeping the compiler current.

## Consequences

- Go becomes a maintained repository toolchain with an explicit owner and recurring update cost.
  Its stronger team fit should make security review and future maintenance easier, but does not
  replace dependency review or secure API handling.
- The CI release job must build the helper with a pinned compiler. After migration it no longer
  needs `actions/github-script` for release updates. The PR-label workflow's separate use of that
  action is outside this decision and needs no repository Node.js test runtime.
- The helper may be longer than a Python version. Keep its API boundary and tests direct; avoid a
  CLI framework or Zarf/UDS imports unless they simplify real requirements.
- Go does not remove the local development toolchain requirement. Its runtime and cross-compilation
  advantages become material only if this helper is distributed beyond the repository.
- Reconsider Python if this work becomes predominantly Python-native data processing, or if the
  owning team's maintained ecosystem changes. Document any new CI download and use an internal
  source when the execution environment requires one.

## References

- Development Automation Best Practices, sections 1, 3, 4, 5, and 6 (workspace guidance).
- [GitHub REST release endpoints](https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28).
- [GitHub REST API version support](https://docs.github.com/en/rest/about-the-rest-api/api-versions).
- [Go toolchain selection and download behavior](https://go.dev/doc/toolchain).
- [Go target platforms and cgo constraints](https://go.dev/doc/install/source#environment).
- [Go security practices and govulncheck](https://go.dev/doc/security/best-practices).
- [uv dependency locking](https://docs.astral.sh/uv/concepts/projects/sync/) for the Python
  alternative.
