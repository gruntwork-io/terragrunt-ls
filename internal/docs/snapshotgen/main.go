// Command snapshotgen updates the index pages compiled into terragrunt-ls from
// the docs site. `go generate ./internal/docs` runs it from the docs package's
// directory.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"terragrunt-ls/internal/docs"
)

const snapshotPerm = 0o644

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	base, err := url.Parse(docs.DefaultBaseURL)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: docs.FetchTimeout}

	for _, cfg := range docs.Configs {
		urlPath := "/reference/hcl/" + string(cfg) + ".md"

		page, err := docs.Fetch(context.Background(), client, base, urlPath, "")
		if err != nil {
			return err
		}

		if _, err := docs.ParseIndex(page.Body); err != nil {
			return fmt.Errorf("%s: %w", urlPath, err)
		}

		if err := os.WriteFile(filepath.Join("snapshot", string(cfg)+".md"), page.Body, snapshotPerm); err != nil {
			return err
		}
	}

	return nil
}
