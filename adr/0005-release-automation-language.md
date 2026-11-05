# 5. Adopt Go for repository-local release automation

Date: 2026-11-05

## Status

Accepted for repository-local release automation.

## Context

The rolling-release workflow checks artifact changes against release versions and updates one
GitHub Release after publication succeeds. The Release command selects the previous consolidated
baseline, preserves generated notes, and inventories OCI artifacts. An earlier draft used
`actions/github-script` and a separately pinned Node.js runtime for local tests and CI.

This repository has shell and YAML automation, but no established Go or Python toolchain. The
owning team reports a substantial, maintained Go ecosystem with vetted Zarf and UDS libraries,
while its Python ecosystem and vetted-library set are lighter. Sibling infrastructure repositories
owned by the team also use Go, including code that depends on Zarf libraries. That evidence makes
team maintenance and potential library reuse material selection factors; it does not mean this
automation needs a Zarf or UDS dependency today.

Concretely, [Defense Unicorns `pkg`](https://github.com/defenseunicorns/pkg) publishes common Go
modules for command execution, helpers, and OCI artifacts. [UDS CLI](https://github.com/defenseunicorns/uds-cli)
and [Zarf](https://github.com/zarf-dev/zarf) are Go projects whose code and package concepts are
familiar to this ecosystem. They are not blanket-approved dependencies: UDS CLI is primarily an
application, and importability, API stability, license compatibility, and transitive dependencies
must be checked for any specific package before use.

The commands run in GitHub Actions and local tests. They use Git, `yq`, and the GitHub API;
package builds and workflow concurrency remain outside them.

## Decision Drivers

| Factor | Go | Python |
| --- | --- | --- |
| Team fit and ownership | Existing team ecosystem, review experience, and vetted libraries reduce maintenance risk. This repository still needs its own Go CI profile. | New repository toolchain with less team-wide library vetting and ownership experience. |
| Task fit | Standard library covers JSON, text, subprocess, HTTP, and testing. More code is likely. | Standard library also fits; concise for API and text processing. |
| Execution and portability | CI builds and runs the commands on Ubuntu with a pinned Go compiler; local contributors need a compatible Go toolchain. A future target-specific build can run without an installed Go toolchain. | The source can run on hosts with a compatible interpreter and reproducibly provisioned environment. A packaged application could change that distribution model. |
| Dependencies and security | Start with standard library and existing `gh`/`yq`. Adopt a shared Zarf/UDS module only if it replaces meaningful custom logic and passes normal dependency review. | Standard-library runtime is also possible, but a Python profile adds uv and its lint/type/test tools to the team's maintenance set. |
| Longevity | Aligns future release or package metadata work with the team's maintained ecosystem. | Lower initial implementation effort, but creates a less-established support path for durable release automation. |
| Concurrency and performance | No decisive advantage: workflow serialization, not language concurrency, preserves release ordering. | Adequate for a few paginated API calls. |

Team ownership, not inherent language security or binary portability, decides this choice. Python
would be shorter, but needs a less-established team toolchain. CI builds Go where it runs; local
tests still require Go. If distributed later, each OS/architecture needs a tested binary, with cgo
and system-library constraints checked. Python source likewise needs a compatible interpreter and
environment on each target. Revisit the choice if implementation size outweighs the team fit.

Keeping JavaScript would retain the ready-made Octokit client but also retain a separate Node.js
source/test toolchain outside the repository's Go/Python guidance. Extending Bash would put
pagination, JSON state transitions, and note preservation into shell beyond its intended role.
Neither alternative reduces the long-term ownership burden enough to justify it here.

## Decision

Adopt Go as this repository's first maintained structured-language toolchain for release readiness
and GitHub Release updates. Keep UDS task actions in POSIX `sh`, short CI glue in Bash,
and package-building tools unchanged. This is not a mandate to rewrite existing shell or a ban on
Python when a future task has a stronger Python-specific fit.

Use Go's standard library for readiness policy, release-state logic, and authenticated GitHub HTTP
calls. This avoids adding a Go library or depending on the runner's `gh` version for release-record
updates. Validate tag names, API responses, and release-note markers before mutation; propagate API
and authentication errors; never print tokens. Provide the GitHub App token through `GH_TOKEN`.
The Release command must only mutate the intended GitHub Release record; Git and OCI tags remain
immutable. Test baselines, notes, ordering, Latest, and retry/failure cases.

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
  and the repository-standard `golangci-lint` configuration. Test GitHub calls with fakes and
  readiness against temporary Git fixtures; tests must not require GitHub credentials. Keep the compiler pin
  current through Renovate and review Go security advisories when it updates. Do not make
  `govulncheck` a required CI gate while these undistributed commands have no third-party Go modules;
  reconsider if dependencies are added or the commands are distributed.
- Distribution: build and execute repository-local source on the same CI runner; do not publish a
  binary solely for these commands. If they become shared tools, follow the guidance's Section 3:
  publish precompiled, versioned binaries for each supported OS/architecture with verification,
  provenance, and appropriate SBOMs, and make disconnected availability explicit. Do not require
  target hosts to install Go or assume that one binary works on every target.

## Consequences

- Go becomes a maintained repository toolchain with an explicit owner and recurring update cost.
  Its stronger team fit should make security review and future maintenance easier, but does not
  replace dependency review or secure API handling.
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
