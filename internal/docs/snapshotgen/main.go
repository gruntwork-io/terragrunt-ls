// Command snapshotgen updates the index pages and variable pages compiled into terragrunt-ls from
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
		if err := update(client, base, string(cfg), docs.ParseIndex); err != nil {
			return err
		}
	}

	for _, v := range docs.Variables {
		parse := func(page []byte) ([]docs.Item, error) { return docs.ParseVariable(v, page) }

		if err := update(client, base, string(docs.KindVariable)+"/"+v.Name, parse); err != nil {
			return err
		}
	}

	return nil
}

// update writes the page at `/reference/hcl/<name>.md` to `snapshot/<name>.md` once parse accepts it.
func update(client *http.Client, base *url.URL, name string, parse func(page []byte) ([]docs.Item, error)) error {
	urlPath := "/reference/hcl/" + name + ".md"

	page, err := docs.Fetch(context.Background(), client, base, urlPath, "")
	if err != nil {
		return err
	}

	if _, err := parse(page.Body); err != nil {
		return fmt.Errorf("%s: %w", urlPath, err)
	}

	return os.WriteFile(filepath.Join("snapshot", filepath.FromSlash(name)+".md"), page.Body, snapshotPerm)
}
