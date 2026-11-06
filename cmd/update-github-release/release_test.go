// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const testRepo = "defenseunicorns/delivery-zarf-init"

type noteCall struct {
	tag      string
	previous string
}

type updateCall struct {
	id      int64
	request releaseRequest
}

type fakeClient struct {
	releases  []githubRelease
	notes     []noteCall
	creates   []releaseRequest
	updates   []updateCall
	listErr   error
	notesErr  error
	createErr error
	updateErr error
}

func (fake *fakeClient) ListReleases(_ context.Context, _ string) ([]githubRelease, error) {
	return fake.releases, fake.listErr
}

func (fake *fakeClient) GenerateNotes(_ context.Context, _, tag, previous string) (string, error) {
	fake.notes = append(fake.notes, noteCall{tag: tag, previous: previous})
	if fake.notesErr != nil {
		return "", fake.notesErr
	}
	return "## What's Changed\n\n- a change", nil
}

func (fake *fakeClient) CreateRelease(_ context.Context, _ string, request releaseRequest) (int64, error) {
	fake.creates = append(fake.creates, request)
	return 42, fake.createErr
}

func (fake *fakeClient) UpdateRelease(_ context.Context, _ string, id int64, request releaseRequest) error {
	fake.updates = append(fake.updates, updateCall{id: id, request: request})
	return fake.updateErr
}

func fixture(version, unicornVersion string) releaseInventory {
	inventory := releaseInventory{}
	for _, name := range []string{"init", "init-agent-only", "init-gitea"} {
		for _, flavorName := range []string{"upstream", "registry1", "unicorn"} {
			artifactVersion := version
			if flavorName == "unicorn" && unicornVersion != "" {
				artifactVersion = unicornVersion
			}
			registry := "ghcr.io/defenseunicorns/delivery-zarf-init"
			if flavorName == "unicorn" {
				registry = "ghcr.io/defenseunicorns/packages/private/delivery-zarf-init"
			}
			inventory.Artifacts = append(inventory.Artifacts, releaseArtifact{
				Package: name, Flavor: flavorName, Version: artifactVersion,
				OCIRef:     "oci://" + registry + "/" + name + ":" + artifactVersion + "-" + flavorName,
				SigningTag: artifactVersion,
			})
		}
	}
	return inventory
}

func publish(t *testing.T, fake *fakeClient, tag string, inventory releaseInventory) string {
	t.Helper()
	message, err := updateRelease(context.Background(), fake, testRepo, tag, inventory)
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func TestFirstRollingRevision(t *testing.T) {
	fake := &fakeClient{releases: []githubRelease{{ID: 8, TagName: "v0.88.0-uds.1"}}}
	publish(t, fake, "v0.89.0-uds.0", fixture("v0.89.0-uds.0", ""))
	if len(fake.creates) != 1 || len(fake.updates) != 0 || len(fake.notes) != 1 {
		t.Fatalf("unexpected calls: creates=%d updates=%d notes=%d", len(fake.creates), len(fake.updates), len(fake.notes))
	}
	if fake.notes[0].previous != "v0.88.0-uds.1" || fake.creates[0].MakeLatest != "true" {
		t.Fatalf("incorrect baseline or Latest: notes=%+v request=%+v", fake.notes[0], fake.creates[0])
	}
	wantBody := strings.Join([]string{
		"<!-- generated-changelog:start -->",
		"## What's Changed",
		"",
		"- a change",
		"<!-- generated-changelog:end -->",
		"",
		"<!-- package-inventory:start -->",
		"## Package inventory",
		"",
		"| Package | Flavor | OCI artifact | Signing tag |",
		"| --- | --- | --- | --- |",
		"| `init` | `upstream` | `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.89.0-uds.0-upstream` | `v0.89.0-uds.0` |",
		"| `init` | `registry1` | `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init:v0.89.0-uds.0-registry1` | `v0.89.0-uds.0` |",
		"| `init` | `unicorn` | `oci://ghcr.io/defenseunicorns/packages/private/delivery-zarf-init/init:v0.89.0-uds.0-unicorn` | `v0.89.0-uds.0` |",
		"| `init-agent-only` | `upstream` | `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init-agent-only:v0.89.0-uds.0-upstream` | `v0.89.0-uds.0` |",
		"| `init-agent-only` | `registry1` | `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init-agent-only:v0.89.0-uds.0-registry1` | `v0.89.0-uds.0` |",
		"| `init-agent-only` | `unicorn` | `oci://ghcr.io/defenseunicorns/packages/private/delivery-zarf-init/init-agent-only:v0.89.0-uds.0-unicorn` | `v0.89.0-uds.0` |",
		"| `init-gitea` | `upstream` | `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init-gitea:v0.89.0-uds.0-upstream` | `v0.89.0-uds.0` |",
		"| `init-gitea` | `registry1` | `oci://ghcr.io/defenseunicorns/delivery-zarf-init/init-gitea:v0.89.0-uds.0-registry1` | `v0.89.0-uds.0` |",
		"| `init-gitea` | `unicorn` | `oci://ghcr.io/defenseunicorns/packages/private/delivery-zarf-init/init-gitea:v0.89.0-uds.0-unicorn` | `v0.89.0-uds.0` |",
		"<!-- package-inventory:end -->",
		"",
	}, "\n")
	if fake.creates[0].Body != wantBody {
		t.Fatalf("incorrect rolling release body:\nwant:\n%s\ngot:\n%s", wantBody, fake.creates[0].Body)
	}
}

func TestRollingReleaseRequiresBaseline(t *testing.T) {
	fake := &fakeClient{releases: []githubRelease{{ID: 7, TagName: "init-v0.88.0-uds.1-upstream"}}}
	_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.0", fixture("v0.89.0-uds.0", ""))
	if err == nil || !strings.Contains(err.Error(), "changelog baseline") || len(fake.creates) != 0 {
		t.Fatalf("expected missing baseline failure, got %v", err)
	}
}

func TestFirstReleaseUsesNewestConsolidatedBaseline(t *testing.T) {
	fake := &fakeClient{releases: []githubRelease{
		{ID: 7, TagName: "v0.87.0-uds.2"},
		{ID: 8, TagName: "v0.88.0-uds.1"},
		{ID: 9, TagName: "init-v0.88.0-uds.2-upstream"},
	}}
	publish(t, fake, "v0.89.0-uds.0", fixture("v0.89.0-uds.0", ""))
	if fake.notes[0].previous != "v0.88.0-uds.1" {
		t.Fatalf("wrong consolidated baseline: %+v", fake.notes[0])
	}
}

func TestLaterRevisionPreservesAuthoredGuidance(t *testing.T) {
	managed, err := renderBody("", "Old changes", "Old inventory")
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeClient{releases: []githubRelease{
		{ID: 8, TagName: "v0.88.0-uds.1"},
		{ID: 9, TagName: "v0.89.0-uds.0", Body: "## Upgrade guidance\n\nKeep this guidance.\n\n" + managed},
	}}
	publish(t, fake, "v0.89.0-uds.1", fixture("v0.89.0-uds.0", "v0.89.0-uds.1"))
	if len(fake.creates) != 0 || len(fake.updates) != 1 || fake.updates[0].id != 9 ||
		fake.notes[0].previous != "v0.88.0-uds.1" {
		t.Fatalf("wrong release update: %+v", fake)
	}
	body := fake.updates[0].request.Body
	for _, required := range []string{
		"Keep this guidance", "init:v0.89.0-uds.1-unicorn` | `v0.89.0-uds.1`",
		"init:v0.89.0-uds.0-upstream` | `v0.89.0-uds.0`",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("missing %q from %s", required, body)
		}
	}
	if strings.Contains(body, "Old changes") || strings.Contains(body, "Old inventory") {
		t.Fatalf("stale managed content: %s", body)
	}
}

func TestOlderAndDuplicateRunsDoNotRewriteRelease(t *testing.T) {
	for _, tag := range []string{"v0.89.0-uds.1", "v0.89.0-uds.2"} {
		t.Run(tag, func(t *testing.T) {
			fake := &fakeClient{releases: []githubRelease{{ID: 9, TagName: "v0.89.0-uds.2"}}}
			publish(t, fake, tag, fixture(tag, ""))
			if len(fake.notes)+len(fake.creates)+len(fake.updates) != 0 {
				t.Fatalf("duplicate or older run mutated release: %+v", fake)
			}
		})
	}
}

func TestNewZarfVersionCreatesAnotherRelease(t *testing.T) {
	fake := &fakeClient{releases: []githubRelease{{ID: 9, TagName: "v0.89.0-uds.2"}}}
	publish(t, fake, "v0.89.1-uds.0", fixture("v0.89.1-uds.0", ""))
	if len(fake.creates) != 1 || len(fake.updates) != 0 || fake.notes[0].previous != "v0.89.0-uds.2" {
		t.Fatalf("wrong new-line release: %+v", fake)
	}
}

func TestOlderLineCannotBecomeLatest(t *testing.T) {
	body, err := renderBody("", "Old", "Old")
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeClient{releases: []githubRelease{
		{ID: 8, TagName: "v0.88.0-uds.1"},
		{ID: 9, TagName: "v0.89.0-uds.0", Body: body},
		{ID: 10, TagName: "v0.89.1-uds.0"},
	}}
	publish(t, fake, "v0.89.0-uds.1", fixture("v0.89.0-uds.1", ""))
	if len(fake.updates) != 1 || fake.updates[0].request.MakeLatest != "false" {
		t.Fatalf("older line changed Latest: %+v", fake.updates)
	}
}

func TestImmutableAndDuplicateSameLineFail(t *testing.T) {
	for _, test := range []struct {
		name     string
		releases []githubRelease
	}{
		{name: "immutable", releases: []githubRelease{{ID: 9, TagName: "v0.89.0-uds.0", Immutable: true}}},
		{name: "duplicate line", releases: []githubRelease{
			{ID: 9, TagName: "v0.89.0-uds.0"}, {ID: 10, TagName: "v0.89.0-uds.1"},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{releases: test.releases}
			_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.2", fixture("v0.89.0-uds.2", ""))
			if err == nil || len(fake.creates)+len(fake.updates) != 0 {
				t.Fatalf("expected pre-mutation failure, got %v", err)
			}
		})
	}
}

func TestRollingRevisionsMustPublishInOrder(t *testing.T) {
	for _, test := range []struct {
		name     string
		releases []githubRelease
	}{
		{name: "no current line", releases: []githubRelease{{ID: 8, TagName: "v0.88.0-uds.1"}}},
		{name: "missing prior revision", releases: []githubRelease{{ID: 9, TagName: "v0.89.0-uds.0"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeClient{releases: test.releases}
			_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.2", fixture("v0.89.0-uds.2", ""))
			if err == nil || !strings.Contains(err.Error(), "preceding") || len(fake.creates)+len(fake.updates) != 0 {
				t.Fatalf("expected revision-order failure, got %v", err)
			}
		})
	}
}

func TestEarlierVersionHasNoSpecialCutoff(t *testing.T) {
	fake := &fakeClient{releases: []githubRelease{{ID: 8, TagName: "v0.87.0-uds.2"}}}
	publish(t, fake, "v0.88.0-uds.0", fixture("v0.88.0-uds.0", ""))
	if len(fake.creates) != 1 || fake.notes[0].previous != "v0.87.0-uds.2" {
		t.Fatalf("earlier release line was rejected: %+v", fake)
	}
}

func TestMalformedNotesAndVersionsFailClosed(t *testing.T) {
	if _, err := renderBody("Hand-written notes", "New", "Inventory"); err == nil {
		t.Fatal("accepted missing managed sections")
	}
	if _, err := renderInventory(fixture("v0.88.0-uds.1", ""), "v0.89.0-uds.0"); err == nil {
		t.Fatal("accepted mismatched artifact versions")
	}
	if _, err := renderInventory(fixture("v0.89.0-uds.0", ""), "v0.89.0-uds.0"); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryUsesExplicitArtifactMetadata(t *testing.T) {
	inventory := fixture("v0.89.0-uds.0", "")
	inventory.Artifacts[0].OCIRef = "oci://registry.example/packages/init:v0.89.0-uds.0-upstream"
	inventory.Artifacts[0].SigningTag = "init-v0.89.0-uds.0-upstream"
	body, err := renderInventory(inventory, "v0.89.0-uds.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "| `init` | `upstream` | `oci://registry.example/packages/init:v0.89.0-uds.0-upstream` | `init-v0.89.0-uds.0-upstream` |") {
		t.Fatalf("explicit artifact metadata was ignored: %s", body)
	}
	inventory.Artifacts[0].OCIRef = "oci:///init:v0.89.0-uds.0-upstream"
	if _, err := renderInventory(inventory, "v0.89.0-uds.0"); err == nil {
		t.Fatal("accepted OCI reference without a registry")
	}
}

func TestListingFailureStopsBeforeMutation(t *testing.T) {
	listErr := errors.New("listing unavailable")
	fake := &fakeClient{listErr: listErr}
	_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.0", fixture("v0.89.0-uds.0", ""))
	if !errors.Is(err, listErr) || !strings.Contains(err.Error(), "list releases") {
		t.Fatalf("expected listing failure, got %v", err)
	}
	if len(fake.notes) != 0 || len(fake.creates) != 0 || len(fake.updates) != 0 {
		t.Fatalf("listing failure reached later operations: %+v", fake)
	}
}

func TestNoteGenerationFailureStopsBeforeMutation(t *testing.T) {
	notesErr := errors.New("notes unavailable")
	fake := &fakeClient{
		releases: []githubRelease{{ID: 8, TagName: "v0.88.0-uds.1"}},
		notesErr: notesErr,
	}
	_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.0", fixture("v0.89.0-uds.0", ""))
	if !errors.Is(err, notesErr) || !strings.Contains(err.Error(), "generate release notes") {
		t.Fatalf("expected note-generation failure, got %v", err)
	}
	if len(fake.notes) != 1 || len(fake.creates) != 0 || len(fake.updates) != 0 {
		t.Fatalf("note-generation failure reached mutation: %+v", fake)
	}
}

func TestCreationFailureIsReturned(t *testing.T) {
	createErr := errors.New("creation unavailable")
	fake := &fakeClient{
		releases:  []githubRelease{{ID: 8, TagName: "v0.88.0-uds.1"}},
		createErr: createErr,
	}
	_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.0", fixture("v0.89.0-uds.0", ""))
	if !errors.Is(err, createErr) || !strings.Contains(err.Error(), "create release") {
		t.Fatalf("expected creation failure, got %v", err)
	}
	if len(fake.creates) != 1 || len(fake.updates) != 0 {
		t.Fatalf("unexpected calls after creation failure: %+v", fake)
	}
}

func TestUpdateFailureIsReturned(t *testing.T) {
	updateErr := errors.New("update unavailable")
	fake := &fakeClient{
		releases: []githubRelease{
			{ID: 8, TagName: "v0.88.0-uds.1"},
			{ID: 9, TagName: "v0.89.0-uds.0"},
		},
		updateErr: updateErr,
	}
	_, err := updateRelease(context.Background(), fake, testRepo, "v0.89.0-uds.1", fixture("v0.89.0-uds.1", ""))
	if !errors.Is(err, updateErr) || !strings.Contains(err.Error(), "advance release 9") {
		t.Fatalf("expected update failure, got %v", err)
	}
	if len(fake.creates) != 0 || len(fake.updates) != 1 || fake.updates[0].id != 9 {
		t.Fatalf("unexpected calls after update failure: %+v", fake)
	}
}
