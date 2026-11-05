// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package main

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	changelogStart = "<!-- generated-changelog:start -->"
	changelogEnd   = "<!-- generated-changelog:end -->"
	inventoryStart = "<!-- package-inventory:start -->"
	inventoryEnd   = "<!-- package-inventory:end -->"
)

var (
	tagPattern  = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)-uds\.([0-9]+)$`)
	namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

type version [4]int

type releaseInventory struct {
	Artifacts []releaseArtifact `json:"artifacts"`
}

type releaseArtifact struct {
	Package    string `json:"package"`
	Flavor     string `json:"flavor"`
	Version    string `json:"version"`
	OCIRef     string `json:"ociRef"`
	SigningTag string `json:"signingTag"`
}

type githubRelease struct {
	ID          int64  `json:"id"`
	TagName     string `json:"tag_name"`
	Draft       bool   `json:"draft"`
	Immutable   bool   `json:"immutable"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	CreatedAt   string `json:"created_at"`
}

type releaseRequest struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	MakeLatest string `json:"make_latest"`
}

type releaseClient interface {
	GetTag(context.Context, string, string) error
	ListReleases(context.Context, string) ([]githubRelease, error)
	GenerateNotes(context.Context, string, string, string) (string, error)
	CreateRelease(context.Context, string, releaseRequest) (int64, error)
	UpdateRelease(context.Context, string, int64, releaseRequest) error
}

func parseTag(tag string) (version, bool) {
	match := tagPattern.FindStringSubmatch(tag)
	if match == nil {
		return version{}, false
	}
	var parsed version
	for i, part := range match[1:] {
		value, err := strconv.Atoi(part)
		if err != nil {
			return version{}, false
		}
		parsed[i] = value
	}
	return parsed, true
}

func parseLegacyTag(tag string, inventory releaseInventory) (version, bool) {
	for _, artifact := range inventory.Artifacts {
		prefix, suffix := artifact.Package+"-", "-"+artifact.Flavor
		if strings.HasPrefix(tag, prefix) && strings.HasSuffix(tag, suffix) {
			if parsed, valid := parseTag(strings.TrimSuffix(strings.TrimPrefix(tag, prefix), suffix)); valid {
				return parsed, true
			}
		}
	}
	return version{}, false
}

func compareVersions(left, right version, parts int) int {
	for i := range parts {
		if left[i] < right[i] {
			return -1
		}
		if left[i] > right[i] {
			return 1
		}
	}
	return 0
}

func replaceBlock(body, start, end, content string) (string, error) {
	from, to := strings.Index(body, start), strings.Index(body, end)
	if from < 0 || to <= from || strings.Count(body, start) != 1 || strings.Count(body, end) != 1 {
		return "", fmt.Errorf("release notes must contain exactly one %s / %s block", start, end)
	}
	return body[:from] + start + "\n" + strings.TrimSpace(content) + "\n" + body[to:], nil
}

func renderBody(existing, changelog, inventory string) (string, error) {
	if existing == "" {
		return changelogStart + "\n" + strings.TrimSpace(changelog) + "\n" + changelogEnd +
			"\n\n" + inventoryStart + "\n" + strings.TrimSpace(inventory) + "\n" + inventoryEnd + "\n", nil
	}
	updated, err := replaceBlock(existing, changelogStart, changelogEnd, changelog)
	if err != nil {
		return "", err
	}
	return replaceBlock(updated, inventoryStart, inventoryEnd, inventory)
}

func renderInventory(inventory releaseInventory, tag string) (string, error) {
	current, ok := parseTag(tag)
	if !ok {
		return "", fmt.Errorf("invalid consolidated release tag: %s", tag)
	}
	rows := []string{"## Package inventory", "", "| Package | Flavor | OCI artifact | Signing tag |", "| --- | --- | --- | --- |"}
	if len(inventory.Artifacts) == 0 {
		return "", fmt.Errorf("release inventory has no artifacts")
	}
	seen := make(map[string]bool)
	for _, artifact := range inventory.Artifacts {
		key := artifact.Package + "/" + artifact.Flavor
		if !namePattern.MatchString(artifact.Package) || !namePattern.MatchString(artifact.Flavor) || seen[key] {
			return "", fmt.Errorf("invalid or duplicate package/flavor: %s", key)
		}
		seen[key] = true
		artifactVersion, valid := parseTag(artifact.Version)
		if !valid || compareVersions(artifactVersion, current, 3) != 0 ||
			compareVersions(artifactVersion, current, 4) > 0 {
			return "", fmt.Errorf("%s has invalid release version %q", key, artifact.Version)
		}
		ref, err := url.Parse(artifact.OCIRef)
		if err != nil || ref.Scheme != "oci" || ref.Host == "" || ref.Path == "" ||
			!strings.HasSuffix(artifact.OCIRef, ":"+artifact.Version+"-"+artifact.Flavor) ||
			strings.ContainsAny(artifact.OCIRef, " `|\r\n\t") ||
			artifact.SigningTag == "" || strings.ContainsAny(artifact.SigningTag, " `|\r\n\t") {
			return "", fmt.Errorf("%s has invalid OCI reference or signing tag", key)
		}
		rows = append(rows, fmt.Sprintf("| `%s` | `%s` | `%s` | `%s` |",
			artifact.Package, artifact.Flavor, artifact.OCIRef, artifact.SigningTag))
	}
	return strings.Join(rows, "\n"), nil
}

type candidate struct {
	release      githubRelease
	version      version
	consolidated bool
}

func releaseTime(release githubRelease) time.Time {
	value := release.PublishedAt
	if value == "" {
		value = release.CreatedAt
	}
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}

func updateRelease(ctx context.Context, client releaseClient, repo, tag string, inventory releaseInventory) (string, error) {
	current, ok := parseTag(tag)
	if !ok {
		return "", fmt.Errorf("invalid consolidated release tag: %s", tag)
	}
	if err := client.GetTag(ctx, repo, tag); err != nil {
		return "", fmt.Errorf("verify tag %s: %w", tag, err)
	}
	releases, err := client.ListReleases(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("list releases: %w", err)
	}
	var previous []candidate
	var existing *githubRelease
	var sameLine int
	newer := false
	for _, release := range releases {
		if parsed, valid := parseTag(release.TagName); valid {
			entry := candidate{release: release, version: parsed, consolidated: true}
			if compareVersions(parsed, current, 3) == 0 {
				sameLine++
				matchedRelease := release
				existing = &matchedRelease
			}
			if !release.Draft && compareVersions(parsed, current, 4) > 0 {
				newer = true
			}
			if !release.Draft && compareVersions(parsed, current, 3) < 0 {
				previous = append(previous, entry)
			}
		} else {
			if parsed, valid := parseLegacyTag(release.TagName, inventory); valid && !release.Draft &&
				compareVersions(parsed, current, 3) < 0 {
				previous = append(previous, candidate{release: release, version: parsed})
			}
		}
	}
	if sameLine > 1 {
		return "", fmt.Errorf("multiple GitHub Releases already exist for %s", strings.Split(tag, "-uds.")[0])
	}
	if existing != nil {
		if existing.Draft {
			return "", fmt.Errorf("release %s is still a draft", existing.TagName)
		}
		if prior, valid := parseTag(existing.TagName); valid && compareVersions(prior, current, 4) >= 0 {
			return existing.TagName + " is already the current or newer release; skipping", nil
		}
		if existing.Immutable {
			return "", fmt.Errorf("release %s is immutable", existing.TagName)
		}
	}
	if current[3] > 0 {
		if existing == nil {
			return "", fmt.Errorf("publish the preceding %s revision first", strings.Split(tag, "-uds.")[0])
		}
		prior, _ := parseTag(existing.TagName)
		if prior[3] != current[3]-1 {
			return "", fmt.Errorf("publish the preceding %s revision first", strings.Split(tag, "-uds.")[0])
		}
	}
	if len(previous) == 0 {
		return "", fmt.Errorf("no prior GitHub Release provides a changelog baseline for %s", tag)
	}
	sort.Slice(previous, func(i, j int) bool {
		if comparison := compareVersions(previous[i].version, previous[j].version, 4); comparison != 0 {
			return comparison > 0
		}
		if previous[i].consolidated != previous[j].consolidated {
			return previous[i].consolidated
		}
		return releaseTime(previous[i].release).After(releaseTime(previous[j].release))
	})
	previousTag := ""
	if len(previous) > 0 {
		previousTag = previous[0].release.TagName
	}
	changelog, err := client.GenerateNotes(ctx, repo, tag, previousTag)
	if err != nil {
		return "", fmt.Errorf("generate release notes: %w", err)
	}
	inventoryBody, err := renderInventory(inventory, tag)
	if err != nil {
		return "", err
	}
	oldBody := ""
	if existing != nil {
		oldBody = existing.Body
	}
	body, err := renderBody(oldBody, changelog, inventoryBody)
	if err != nil {
		return "", err
	}
	latest := "true"
	if newer {
		latest = "false"
	}
	request := releaseRequest{TagName: tag, Name: tag, Body: body, MakeLatest: latest}
	if existing != nil {
		if err := client.UpdateRelease(ctx, repo, existing.ID, request); err != nil {
			return "", fmt.Errorf("advance release %d: %w", existing.ID, err)
		}
		return fmt.Sprintf("advanced GitHub Release %d from %s to %s", existing.ID, existing.TagName, tag), nil
	}
	id, err := client.CreateRelease(ctx, repo, request)
	if err != nil {
		return "", fmt.Errorf("create release: %w", err)
	}
	return fmt.Sprintf("created GitHub Release %d for %s", id, tag), nil
}
