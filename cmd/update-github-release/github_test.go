// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func testAPI(t *testing.T, handler http.HandlerFunc) *githubAPI {
	t.Helper()
	api := newGitHubAPI("test-token")
	api.baseURL = "https://api.github.test"
	api.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer test-token" ||
			request.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("missing GitHub API authentication or version header")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Result(), nil
	})}
	return api
}

func TestGitHubAPIListReleasesPagination(t *testing.T) {
	var pages int
	api := testAPI(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/repos/defenseunicorns/delivery-zarf-init/releases" ||
			request.URL.Query().Get("per_page") != "100" {
			t.Errorf("unexpected release listing request: %s %s", request.Method, request.URL)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		pages++
		if request.URL.Query().Get("page") == "1" {
			page := make([]githubRelease, 100)
			for i := range page {
				page[i] = githubRelease{ID: int64(i + 1), TagName: "v0.88.0-uds.1"}
			}
			if err := json.NewEncoder(writer).Encode(page); err != nil {
				t.Error(err)
			}
			return
		}
		if request.URL.Query().Get("page") != "2" {
			t.Errorf("unexpected release listing page: %s", request.URL)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := fmt.Fprint(writer, `[]`); err != nil {
			t.Error(err)
		}
	}))
	releases, err := api.ListReleases(context.Background(), testRepo)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 100 {
		t.Fatalf("incorrect release count: %d", len(releases))
	}
	if pages != 2 || releases[0].ID != 1 || releases[99].ID != 100 {
		t.Fatalf("incorrect pagination: first=%+v last=%+v pages=%d", releases[0], releases[99], pages)
	}
}

func TestGitHubAPIGenerateNotes(t *testing.T) {
	api := testAPI(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost ||
			request.URL.Path != "/repos/defenseunicorns/delivery-zarf-init/releases/generate-notes" {
			t.Errorf("unexpected note request: %s %s", request.Method, request.URL)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		var input map[string]string
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Error(err)
			return
		}
		if input["tag_name"] != "v0.89.0-uds.0" || input["previous_tag_name"] != "v0.88.0-uds.1" {
			t.Errorf("incorrect note request: %+v", input)
		}
		if _, err := fmt.Fprint(writer, `{"body":"generated notes"}`); err != nil {
			t.Error(err)
		}
	}))
	notes, err := api.GenerateNotes(context.Background(), testRepo, "v0.89.0-uds.0", "v0.88.0-uds.1")
	if err != nil {
		t.Fatal(err)
	}
	if notes != "generated notes" {
		t.Fatalf("incorrect notes: %q", notes)
	}
}

func TestGitHubAPICreateRelease(t *testing.T) {
	want := releaseRequest{TagName: "v0.89.0-uds.0", Name: "v0.89.0-uds.0", Body: "notes", MakeLatest: "true"}
	api := testAPI(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/repos/defenseunicorns/delivery-zarf-init/releases" {
			t.Errorf("unexpected create request: %s %s", request.Method, request.URL)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		var input releaseRequest
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Error(err)
			return
		}
		if input != want {
			t.Errorf("incorrect create request: got %+v, want %+v", input, want)
		}
		if _, err := fmt.Fprint(writer, `{"id":42}`); err != nil {
			t.Error(err)
		}
	}))
	id, err := api.CreateRelease(context.Background(), testRepo, want)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("incorrect created release ID: %d", id)
	}
}

func TestGitHubAPIUpdateRelease(t *testing.T) {
	want := releaseRequest{TagName: "v0.89.0-uds.1", Name: "v0.89.0-uds.1", Body: "updated", MakeLatest: "true"}
	api := testAPI(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch || request.URL.Path != "/repos/defenseunicorns/delivery-zarf-init/releases/42" {
			t.Errorf("unexpected update request: %s %s", request.Method, request.URL)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		var input releaseRequest
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Error(err)
			return
		}
		if input != want {
			t.Errorf("incorrect update request: got %+v, want %+v", input, want)
		}
		if _, err := fmt.Fprint(writer, `{}`); err != nil {
			t.Error(err)
		}
	}))
	if err := api.UpdateRelease(context.Background(), testRepo, 42, want); err != nil {
		t.Fatal(err)
	}
}

func TestGitHubAPIFailsClosed(t *testing.T) {
	api := testAPI(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
		if _, err := fmt.Fprint(writer, `{"message":"permission denied"}`); err != nil {
			t.Error(err)
		}
	}))
	_, err := api.ListReleases(context.Background(), testRepo)
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "test-token") {
		t.Fatalf("incorrect API error: %v", err)
	}
}

func TestGitHubAPIRejectsMalformedListing(t *testing.T) {
	api := testAPI(t, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if _, err := fmt.Fprint(writer, `null`); err != nil {
			t.Error(err)
		}
	}))
	_, err := api.ListReleases(context.Background(), testRepo)
	if err == nil || !strings.Contains(err.Error(), "non-array release listing") {
		t.Fatalf("expected malformed listing failure, got %v", err)
	}
}

func TestValidateRepo(t *testing.T) {
	if err := validateRepo(testRepo); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"", "owner", "owner/repo/extra", "owner/repo?per_page=1"} {
		t.Run(invalid, func(t *testing.T) {
			if err := validateRepo(invalid); err == nil {
				t.Fatalf("accepted invalid repository: %q", invalid)
			}
		})
	}
}
