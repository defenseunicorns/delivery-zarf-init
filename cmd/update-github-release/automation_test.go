// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func automationCommand(t *testing.T, dir string, env []string, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func mustAutomationCommand(t *testing.T, dir string, env []string, name string, args ...string) string {
	t.Helper()
	output, err := automationCommand(t, dir, env, name, args...)
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return output
}

func writeFixtureFile(t *testing.T, dir, path, contents string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func releaseFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, path := range []string{"tasks.yaml", "releaser.yaml", "release-registries.yaml"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", path))
		if err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, dir, path, string(contents))
	}
	mustAutomationCommand(t, dir, nil, "yq", "-i", `(.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default) = "v0.88.0"`, "tasks.yaml")
	setFixtureVersions(t, dir, "all", "all", "v0.88.0-uds.0")
	setFixtureVersions(t, dir, "all", "unicorn", "v0.88.0-uds.1")
	writeFixtureFile(t, dir, "packages/init/config.yaml", "before\n")
	writeFixtureFile(t, dir, "flavors/upstream.yaml", "before\n")
	writeFixtureFile(t, dir, "tasks/package.yaml", "before\n")
	mustAutomationCommand(t, dir, nil, "git", "init", "-q")
	mustAutomationCommand(t, dir, nil, "git", "add", ".")
	mustAutomationCommand(t, dir, nil, "git", "-c", "user.name=Release Test", "-c", "user.email=release-test@example.invalid", "commit", "-qm", "baseline")
	return dir
}

func setFixtureVersions(t *testing.T, dir, packageName, flavorName, version string) {
	t.Helper()
	env := []string{"PACKAGE_NAME=" + packageName, "FLAVOR_NAME=" + flavorName, "RELEASE_VERSION=" + version}
	mustAutomationCommand(t, dir, env, "yq", "-i", `(.packages[] | select(env(PACKAGE_NAME) == "all" or .name == env(PACKAGE_NAME)) | .flavors[] | select(env(FLAVOR_NAME) == "all" or .name == env(FLAVOR_NAME)) | .version) = env(RELEASE_VERSION)`, "releaser.yaml")
}

func checkReadiness(t *testing.T, dir string, wantSuccess bool, wantText string) {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "check-release-readiness.sh"))
	if err != nil {
		t.Fatal(err)
	}
	output, err := automationCommand(t, dir, nil, script, "HEAD")
	if (err == nil) != wantSuccess || !strings.Contains(output, wantText) {
		t.Fatalf("readiness: success=%t, output=%q; want success=%t and %q", err == nil, output, wantSuccess, wantText)
	}
}

func TestReleaseReadinessPackageAndFlavorScope(t *testing.T) {
	t.Run("package", func(t *testing.T) {
		dir := releaseFixture(t)
		writeFixtureFile(t, dir, "packages/init/config.yaml", "after\n")
		checkReadiness(t, dir, false, "init/upstream changed artifact inputs")
		setFixtureVersions(t, dir, "init", "all", "v0.88.0-uds.2")
		checkReadiness(t, dir, true, "3 release entries are ready at v0.88.0-uds.2")
	})
	t.Run("flavor", func(t *testing.T) {
		dir := releaseFixture(t)
		writeFixtureFile(t, dir, "flavors/upstream.yaml", "after\n")
		checkReadiness(t, dir, false, "init-gitea/upstream changed artifact inputs")
		setFixtureVersions(t, dir, "all", "upstream", "v0.88.0-uds.2")
		checkReadiness(t, dir, true, "3 release entries are ready at v0.88.0-uds.2")
	})
}

func TestReleaseReadinessSharedBuildAndRevision(t *testing.T) {
	dir := releaseFixture(t)
	writeFixtureFile(t, dir, "tasks/package.yaml", "after\n")
	checkReadiness(t, dir, false, "init-agent-only/registry1 changed artifact inputs")
	setFixtureVersions(t, dir, "all", "all", "v0.88.0-uds.1")
	checkReadiness(t, dir, false, "expected v0.88.0-uds.2")
	setFixtureVersions(t, dir, "all", "all", "v0.88.0-uds.2")
	checkReadiness(t, dir, true, "9 release entries are ready at v0.88.0-uds.2")
}

func TestReleaseReadinessRegistryRouting(t *testing.T) {
	dir := releaseFixture(t)
	mustAutomationCommand(t, dir, nil, "yq", "-i", `.registries.unicorn = "ghcr.io/example/private"`, "release-registries.yaml")
	checkReadiness(t, dir, false, "init-agent-only/registry1 changed artifact inputs")
	setFixtureVersions(t, dir, "all", "all", "v0.88.0-uds.2")
	checkReadiness(t, dir, true, "9 release entries are ready at v0.88.0-uds.2")
}

func TestReleaseReadinessInitialRegistryMetadata(t *testing.T) {
	dir := releaseFixture(t)
	mustAutomationCommand(t, dir, nil, "git", "rm", "-q", "release-registries.yaml")
	mustAutomationCommand(t, dir, nil, "git", "-c", "user.name=Release Test", "-c", "user.email=release-test@example.invalid", "commit", "-qm", "legacy baseline")
	contents, err := os.ReadFile(filepath.Join("..", "..", "release-registries.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, dir, "release-registries.yaml", string(contents))
	checkReadiness(t, dir, true, "No package release required")
}

func TestReleaseReadinessZarfReset(t *testing.T) {
	dir := releaseFixture(t)
	mustAutomationCommand(t, dir, nil, "yq", "-i", `(.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default) = "v0.89.0"`, "tasks.yaml")
	setFixtureVersions(t, dir, "init", "upstream", "v0.89.0-uds.0")
	checkReadiness(t, dir, false, "A Zarf update must set every releaser entry to v0.89.0-uds.0")
	setFixtureVersions(t, dir, "all", "all", "v0.89.0-uds.0")
	checkReadiness(t, dir, true, "9 release entries are ready at v0.89.0-uds.0")
}

func TestBumpReleaseTask(t *testing.T) {
	dir := releaseFixture(t)
	releaser := filepath.Join(dir, "releaser.yaml")
	project, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	mustAutomationCommand(t, project, nil, "uds", "run", "bump-release", "--with", "file="+releaser, "--set", "PACKAGE=init", "--set", "FLAVOR=unicorn", "--set", "ZARF_SOURCE_VERSION=v0.88.0", "--no-progress")
	selected := mustAutomationCommand(t, dir, nil, "yq", "-r", `.packages[] | select(.name == "init") | .flavors[] | select(.name == "unicorn") | .version`, "releaser.yaml")
	if strings.TrimSpace(selected) != "v0.88.0-uds.2" {
		t.Fatalf("selected version = %q, want v0.88.0-uds.2", selected)
	}
	unchanged := mustAutomationCommand(t, dir, nil, "yq", "-r", `.packages[] | select(.name == "init-gitea") | .flavors[] | select(.name == "upstream") | .version`, "releaser.yaml")
	if strings.TrimSpace(unchanged) != "v0.88.0-uds.0" {
		t.Fatalf("unselected version = %q, want v0.88.0-uds.0", unchanged)
	}
}

func TestReleaseMetadata(t *testing.T) {
	dir := releaseFixture(t)
	setFixtureVersions(t, dir, "all", "all", "v0.89.0-uds.0")
	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "release-metadata.sh"))
	if err != nil {
		t.Fatal(err)
	}
	matrixOutput := mustAutomationCommand(t, dir, nil, "sh", script, "matrix")
	var matrix struct {
		Include []struct {
			Package string `json:"package"`
			Flavor  string `json:"flavor"`
		} `json:"include"`
	}
	if err := json.Unmarshal([]byte(matrixOutput), &matrix); err != nil {
		t.Fatal(err)
	}
	wantMatrix := []string{
		"init/upstream", "init/registry1", "init/unicorn",
		"init-agent-only/upstream", "init-agent-only/registry1", "init-agent-only/unicorn",
		"init-gitea/upstream", "init-gitea/registry1", "init-gitea/unicorn",
	}
	var gotMatrix []string
	for _, entry := range matrix.Include {
		gotMatrix = append(gotMatrix, entry.Package+"/"+entry.Flavor)
	}
	if !reflect.DeepEqual(gotMatrix, wantMatrix) {
		t.Fatalf("publish matrix = %v, want %v", gotMatrix, wantMatrix)
	}

	inventoryOutput := mustAutomationCommand(t, dir, nil, "sh", script, "inventory")
	var inventory releaseInventory
	if err := json.Unmarshal([]byte(inventoryOutput), &inventory); err != nil {
		t.Fatal(err)
	}
	if len(inventory.Artifacts) != 9 {
		t.Fatalf("wrong artifact count: %d", len(inventory.Artifacts))
	}
	if inventory.Artifacts[0].OCIRef != "oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.89.0-uds.0-upstream" ||
		inventory.Artifacts[1].OCIRef != "oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.89.0-uds.0-registry1" ||
		inventory.Artifacts[2].OCIRef != "oci://ghcr.io/defenseunicorns/packages/private/delivery-zarf-init/init:v0.89.0-uds.0-unicorn" {
		t.Fatalf("wrong OCI routing or signing identity: %+v", inventory.Artifacts[:3])
	}
	for _, artifact := range inventory.Artifacts {
		if artifact.SigningTag != "v0.89.0-uds.0" {
			t.Fatalf("%s/%s signing tag = %q", artifact.Package, artifact.Flavor, artifact.SigningTag)
		}
	}
	if _, err := renderInventory(inventory, "v0.89.0-uds.0"); err != nil {
		t.Fatal(err)
	}
	mustAutomationCommand(t, dir, nil, "yq", "-i", `del(.registries.unicorn)`, "release-registries.yaml")
	missingOutput, err := automationCommand(t, dir, nil, "sh", script, "inventory")
	if err == nil || !strings.Contains(missingOutput, "no publish repository configured for unicorn") {
		t.Fatalf("missing registry was accepted: err=%v output=%q", err, missingOutput)
	}
}

func TestReleaseTagValidation(t *testing.T) {
	project, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	script := mustAutomationCommand(t, project, nil, "yq", "-r", `.jobs.validate.steps[] | select(.name == "Validate release tag") | .run`, ".github/workflows/release.yaml")
	for _, tc := range []struct {
		name, tag, want string
		valid           bool
	}{
		{"current", "v0.88.0-uds.1", "", true},
		{"artifact", "init-v0.88.0-uds.1-unicorn", "invalid consolidated release tag", false},
		{"future", "v0.89.0-uds.0", "does not match consolidated release version", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := releaseFixture(t)
			scriptFile := filepath.Join(t.TempDir(), "validate.sh")
			writeFixtureFile(t, filepath.Dir(scriptFile), filepath.Base(scriptFile), script)
			output, err := automationCommand(t, dir, []string{"GITHUB_REF_NAME=" + tc.tag}, "bash", scriptFile)
			if (err == nil) != tc.valid || !strings.Contains(output, tc.want) {
				t.Fatalf("validation: success=%t, output=%q; want success=%t and %q", err == nil, output, tc.valid, tc.want)
			}
		})
	}
	triggers := mustAutomationCommand(t, project, nil, "yq", "-r", `.on.push.tags[]`, ".github/workflows/release.yaml")
	if strings.TrimSpace(triggers) != "v*.*.*-uds.*" {
		t.Fatalf("unexpected release tag triggers: %q", triggers)
	}
}
