// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

// Package main updates the GitHub Release for a successfully published Zarf version.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type githubAPI struct {
	client  *http.Client
	baseURL string
	token   string
}

func newGitHubAPI(token string) *githubAPI {
	return &githubAPI{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://api.github.com",
		token:   token,
	}
}

func (api *githubAPI) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, api.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("prepare GitHub request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+api.token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "delivery-zarf-init-release")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub request failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&apiError)
		return fmt.Errorf("GitHub API %s %s returned %d: %s", method, path,
			response.StatusCode, apiError.Message)
	}
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 10<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}
	return nil
}

func (api *githubAPI) GetTag(ctx context.Context, repo, tag string) error {
	return api.request(ctx, http.MethodGet, "/repos/"+repo+"/git/ref/tags/"+url.PathEscape(tag), nil, nil)
}

func (api *githubAPI) ListReleases(ctx context.Context, repo string) ([]githubRelease, error) {
	var all []githubRelease
	for page := 1; page <= 100; page++ {
		var releases []githubRelease
		path := "/repos/" + repo + "/releases?per_page=100&page=" + strconv.Itoa(page)
		if err := api.request(ctx, http.MethodGet, path, nil, &releases); err != nil {
			return nil, err
		}
		if releases == nil {
			return nil, fmt.Errorf("GitHub returned a non-array release listing")
		}
		for _, release := range releases {
			if release.ID <= 0 || release.TagName == "" {
				return nil, fmt.Errorf("GitHub returned a release without an ID or tag")
			}
			for _, timestamp := range []string{release.PublishedAt, release.CreatedAt} {
				if timestamp != "" {
					if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
						return nil, fmt.Errorf("GitHub returned an invalid release timestamp: %w", err)
					}
				}
			}
		}
		all = append(all, releases...)
		if len(releases) < 100 {
			return all, nil
		}
	}
	return nil, fmt.Errorf("GitHub release listing exceeded 100 pages")
}

func (api *githubAPI) GenerateNotes(ctx context.Context, repo, tag, previous string) (string, error) {
	input := map[string]string{"tag_name": tag}
	if previous != "" {
		input["previous_tag_name"] = previous
	}
	var output struct {
		Body string `json:"body"`
	}
	if err := api.request(ctx, http.MethodPost, "/repos/"+repo+"/releases/generate-notes", input, &output); err != nil {
		return "", err
	}
	return output.Body, nil
}

func (api *githubAPI) CreateRelease(ctx context.Context, repo string, input releaseRequest) (int64, error) {
	var output struct {
		ID int64 `json:"id"`
	}
	if err := api.request(ctx, http.MethodPost, "/repos/"+repo+"/releases", input, &output); err != nil {
		return 0, err
	}
	if output.ID <= 0 {
		return 0, fmt.Errorf("GitHub returned a release without an ID")
	}
	return output.ID, nil
}

func (api *githubAPI) UpdateRelease(ctx context.Context, repo string, id int64, input releaseRequest) error {
	if id <= 0 {
		return fmt.Errorf("invalid release ID %d", id)
	}
	return api.request(ctx, http.MethodPatch, "/repos/"+repo+"/releases/"+strconv.FormatInt(id, 10), input, nil)
}

func validateRepo(repo string) error {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || !namePattern.MatchString(strings.ToLower(parts[0])) ||
		!namePattern.MatchString(strings.ToLower(parts[1])) {
		return fmt.Errorf("invalid GitHub repository %q", repo)
	}
	return nil
}
