# 4. Consolidate repository releases while retaining independent artifact versions

Date: 2026-11-04

## Status

Proposed; supersedes the release-specific decisions in
[ADR 0003](./0003-peer-init-packages.md) if accepted.

## Context

The repository publishes a matrix of three packages and three image flavors. Historically, every
package/flavor artifact created its own Git tag and GitHub Release. This produced many GitHub
Releases for one logical repository change and made the supported release line difficult for
consumers to identify.

The OCI artifacts cannot share one version unconditionally. A change may affect one package, one
flavor across multiple packages, or the complete matrix. Unaffected immutable artifacts should
remain at their existing versions rather than being rebuilt solely to produce a coordinated GitHub
Release.

## Decision

- Keep an independent version for every package/flavor entry in `releaser.yaml`. OCI tags remain
  `<zarf-version>-uds.<revision>-<flavor>` within the package's repository.
- Use a repository-wide `-uds.N` sequence for package-only changes. `uds run bump-release` assigns
  one revision greater than the highest revision for the current `ZARF_SOURCE_VERSION` to the
  selected package/flavor entries. Zarf updates reset every entry to `<zarf-version>-uds.0` through
  Renovate.
- Use the highest version declared in `releaser.yaml` as the consolidated repository release
  version. A push to `main` creates that Git tag when it does not already exist.
- Trigger one release workflow from the consolidated tag. Evaluate the complete package/flavor
  matrix, but publish only entries whose version equals the consolidated release version.
- Before publishing, inspect each selected OCI tag. Skip an artifact that already contains both
  supported architectures and publish when the tag or an architecture is missing. Fail on registry
  or authentication errors rather than treating them as an unpublished artifact.
- Create a package/flavor Git tag for every selected artifact for traceability and compatibility
  with existing tooling. Do not create package- or flavor-specific GitHub Releases.
- Starting with Zarf v0.89.0, keep one GitHub Release per Zarf version. After every selected artifact
  succeeds, create the Release for `-uds.0` or advance the same Release record to the newest `-uds.N`
  tag. Keep every earlier Git and OCI tag fixed. Do not migrate the existing 85–88 releases. Route
  any additional pre-v0.89.0 revisions through the legacy package/flavor release workflow.
- Regenerate categorized notes from the previous Zarf version's final release tag and append an
  inventory of every package/flavor OCI reference and its signing tag. Replace only marked generated
  sections so hand-written upgrade guidance remains intact. Accept the final legacy package release
  tag as the first rolling changelog baseline. Wait for a revision's GitHub Release before publishing
  the next revision's artifacts, serialize Release record changes, and never let an older workflow
  move a Release backward or change Latest to an older Zarf version.
- Protect release and artifact Git tags against update and deletion while allowing the release App
  to create tags. Keep GitHub Release immutability disabled so the Release record can be reassociated.
- Map conventional pull request titles to release-note labels. Renovate pull requests use the
  dependency category regardless of their title. A conventional `!` on other pull requests maps
  to the breaking-change category, but migration guidance remains an explicit author responsibility.
- Validate Zarf values before creating the consolidated tag. Check release readiness in pull
  request CI. Shared component and package build task changes require the complete matrix, flavor
  changes require that flavor across all packages, package changes require all flavors of that
  package, and a Zarf source change requires the complete matrix at `-uds.0`.
- Choose and maintain the GitHub Release helper's structured-language toolchain under
  [ADR 0005](./0005-release-automation-language.md); release behavior does not depend on that choice.

## Consequences

- Consumers see one GitHub Release per Zarf version while OCI artifacts retain precise,
  independently advancing versions. The Release record advances to the latest successful revision,
  so its tag-specific URL changes; earlier Git and OCI tags remain stable.
- A consolidated release is an inventory of the latest successful revision; entries can legitimately
  carry older versions and retain the signing identity of the tag that published them.
- Existing immutable artifacts are reused, avoiding unnecessary rebuilds and republishes.
- Published artifacts use the consolidated tag at publication time in their keyless-signing
  certificate identity. Artifacts published before this decision retain their package/flavor tag
  identities; advancing a GitHub Release does not change either identity.
- The highest artifact version and repository release version are coupled. Introducing an explicit
  repository release version would require a later decision.
- The bump task accepts one package selector and one flavor selector. Mixed package-and-flavor
  changes currently require selecting the complete matrix and may bump unaffected entries.
- Re-running the bump task assigns another revision; contributors run it once per release.
- Release readiness depends on a maintained mapping from changed repository paths to affected
  artifacts.
