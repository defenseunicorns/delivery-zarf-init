// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	for _, path := range []string{"tasks.yaml", "releaser.yaml"} {
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

func TestReleaseTagRouting(t *testing.T) {
	project, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	script := mustAutomationCommand(t, project, nil, "yq", "-r", `.jobs.validate.steps[] | select(.id == "tag") | .run`, ".github/workflows/release.yaml")
	for _, tc := range []struct {
		name, tag, version, want string
	}{
		{"legacy artifact", "init-v0.88.0-uds.1-unicorn", "", "rolling=false\nskip=false\n"},
		{"rolling artifact", "init-v0.89.0-uds.0-unicorn", "v0.89.0-uds.0", "rolling=false\nskip=true\n"},
		{"consolidated", "v0.89.0-uds.0", "v0.89.0-uds.0", "rolling=true\nskip=false\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := releaseFixture(t)
			if tc.version != "" {
				setFixtureVersions(t, dir, "all", "all", tc.version)
			}
			outputFile := filepath.Join(t.TempDir(), "output")
			scriptFile := filepath.Join(t.TempDir(), "validate.sh")
			writeFixtureFile(t, filepath.Dir(scriptFile), filepath.Base(scriptFile), script)
			env := []string{"GITHUB_REF_NAME=" + tc.tag, "GITHUB_OUTPUT=" + outputFile}
			mustAutomationCommand(t, dir, env, "bash", scriptFile)
			output, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(output) != tc.want {
				t.Fatalf("tag outputs = %q, want %q", output, tc.want)
			}
		})
	}
	gates := mustAutomationCommand(t, project, nil, "yq", "-r", `[.jobs.values.needs, .jobs.values.if, .jobs.publish.if] | join("|")`, ".github/workflows/release.yaml")
	if strings.TrimSpace(gates) != "validate|needs.validate.outputs.skip != 'true'|needs.validate.outputs.skip != 'true'" {
		t.Fatalf("artifact-tag job gates = %q", gates)
	}
}
