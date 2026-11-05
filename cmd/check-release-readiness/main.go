// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

// Package main checks that artifact-affecting changes have matching release versions.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type releaseConfig struct {
	Packages []struct {
		Name    string `json:"name"`
		Flavors []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"flavors"`
	} `json:"packages"`
}

func command(input []byte, name string, args ...string) []byte {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("%s %s: %v", name, strings.Join(args, " "), err)
	}
	return output
}

func yq(input []byte, expression string) string {
	return strings.TrimSpace(string(command(input, "yq", "-r", expression, "-")))
}

func main() {
	log.SetFlags(0)
	if len(os.Args) != 2 {
		log.Fatal("usage: check-release-readiness BASE_REF")
	}
	base := os.Args[1]
	baseTasks := command(nil, "git", "show", base+":tasks.yaml")
	baseReleaser := command(nil, "git", "show", base+":releaser.yaml")
	currentTasks, err := os.ReadFile("tasks.yaml")
	if err != nil {
		log.Fatal(err)
	}
	currentReleaser, err := os.ReadFile("releaser.yaml")
	if err != nil {
		log.Fatal(err)
	}

	buildTasks := `[ (.includes[] | select(has("create")) | .create),
      (.tasks[] | select(.name == "create-package" or .name == "publish-package") |
        {"name": .name, "inputs": (.inputs // {} | with_entries(.value = .value.default)), "actions": .actions}) ]`
	buildJSON := func(input []byte) []byte {
		return command(input, "yq", "-o=json", "-I=0", buildTasks, "-")
	}
	requireAll := !bytes.Equal(buildJSON(baseTasks), buildJSON(currentTasks))
	baseSource := yq(baseTasks, `.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default`)
	currentSource := yq(currentTasks, `.variables[] | select(.name == "ZARF_SOURCE_VERSION") | .default`)
	requireAll = requireAll || baseSource != currentSource

	packages := map[string]bool{}
	flavors := map[string]bool{}
	paths := command(nil, "git", "diff", "--name-only", "-z", base, "--",
		"components", "flavors", "packages", "tasks/package.yaml", "release-registries.yaml")
	registryExisted := exec.Command("git", "cat-file", "-e", base+":release-registries.yaml").Run() == nil
	for _, path := range strings.Split(strings.TrimSuffix(string(paths), "\x00"), "\x00") {
		switch {
		case strings.HasPrefix(path, "components/"), path == "tasks/package.yaml":
			requireAll = true
		case path == "release-registries.yaml" && registryExisted:
			requireAll = true
		case path == "flavors/upstream.yaml", path == "flavors/unicorn.yaml":
			flavors[strings.TrimSuffix(strings.TrimPrefix(path, "flavors/"), ".yaml")] = true
		case strings.HasPrefix(path, "flavors/registry1") && strings.HasSuffix(path, ".yaml"):
			flavors["registry1"] = true
		case strings.HasPrefix(path, "flavors/"):
			requireAll = true
		case strings.HasPrefix(path, "packages/"):
			packages[strings.SplitN(strings.TrimPrefix(path, "packages/"), "/", 2)[0]] = true
		}
	}

	readConfig := func(input []byte) releaseConfig {
		var config releaseConfig
		if err := json.Unmarshal(command(input, "yq", "-o=json", "-I=0", ".", "-"), &config); err != nil {
			log.Fatal(err)
		}
		return config
	}
	previous := map[string]string{}
	maxRevision := -1
	for _, pkg := range readConfig(baseReleaser).Packages {
		for _, flavor := range pkg.Flavors {
			previous[pkg.Name+"/"+flavor.Name] = flavor.Version
			if revision, found := strings.CutPrefix(flavor.Version, currentSource+"-uds."); found {
				n, err := strconv.Atoi(revision)
				if err != nil || n < 0 {
					log.Fatalf("invalid base UDS revision: %s", flavor.Version)
				}
				maxRevision = max(maxRevision, n)
			}
		}
	}
	expected := currentSource + "-uds.0"
	if baseSource == currentSource {
		if maxRevision < 0 {
			log.Fatalf("cannot resolve the base UDS revision for %s", currentSource)
		}
		expected = fmt.Sprintf("%s-uds.%d", currentSource, maxRevision+1)
	}

	changed, failed := 0, false
	for _, pkg := range readConfig(currentReleaser).Packages {
		for _, flavor := range pkg.Flavors {
			key := pkg.Name + "/" + flavor.Name
			bumped := previous[key] != flavor.Version
			if bumped {
				changed++
				if flavor.Version != expected {
					fmt.Fprintf(os.Stderr, "%s uses %s; expected %s\n", key, flavor.Version, expected)
					failed = true
				}
			}
			if (requireAll || packages[pkg.Name] || flavors[flavor.Name]) && !bumped {
				fmt.Fprintf(os.Stderr, "%s changed artifact inputs but remains at %s\n", key, flavor.Version)
				failed = true
			}
		}
	}
	if changed == 0 && (requireAll || len(packages)+len(flavors) > 0) {
		fmt.Fprintln(os.Stderr, "artifact inputs changed without a releaser.yaml version bump")
		failed = true
	}
	if failed {
		fmt.Fprintln(os.Stderr, "Expected release version:", expected)
		if baseSource != currentSource {
			fmt.Fprintf(os.Stderr, "A Zarf update must set every releaser entry to %s.\n", expected)
		}
		os.Exit(1)
	}
	if changed > 0 {
		fmt.Printf("%d release entries are ready at %s\n", changed, expected)
	} else {
		fmt.Println("No package release required")
	}
}
