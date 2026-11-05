// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	repo := flag.String("repo", "", "GitHub owner/repository")
	tag := flag.String("tag", "", "consolidated release tag")
	flag.Parse()
	if err := run(context.Background(), os.Stdin, os.Stdout, *repo, *tag); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, input io.Reader, output io.Writer, repo, tag string) error {
	if err := validateRepo(repo); err != nil {
		return err
	}
	if _, ok := parseTag(tag); !ok {
		return fmt.Errorf("invalid consolidated release tag: %s", tag)
	}
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		return fmt.Errorf("GH_TOKEN is required")
	}
	var inventory releaseInventory
	if err := json.NewDecoder(io.LimitReader(input, 1<<20)).Decode(&inventory); err != nil {
		return fmt.Errorf("decode release inventory: %w", err)
	}
	message, err := updateRelease(ctx, newGitHubAPI(token), repo, tag, inventory)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, message)
	return err
}
