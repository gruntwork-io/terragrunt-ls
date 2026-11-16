// Package docs provides the Terragrunt HCL reference that hover and completion
// show. Index pages compiled into terragrunt-ls give each block, attribute, and
// function a one-line description. The index pages, and each item's page once
// a hover asks for it, are fetched in the background from the Markdown version
// of the docs site and cached on disk, so no request waits on the network.
package docs

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"terragrunt-ls/internal/logger"
)

//go:generate go run ./snapshotgen

//go:embed snapshot/*.md
var snapshot embed.FS

// Kind is a kind of HCL reference item, as its docs URL names it.
type Kind string

const (
	KindBlock     Kind = "blocks"
	KindAttribute Kind = "attributes"
	KindFunction  Kind = "functions"
)

// Config is a kind of configuration file, as the URL of its index page names
// it.
type Config string

const (
	// ConfigUnit is `terragrunt.hcl` and the files it includes.
	ConfigUnit Config = "units"
	// ConfigStack is `terragrunt.stack.hcl`.
	ConfigStack Config = "stacks"
)

// Configs lists every [Config].
var Configs = []Config{ConfigUnit, ConfigStack}

// Item is an entry of an index page.
type Item struct {
	Kind        Kind
	Name        string
	Description string
}

// FetchingNote ends a hover that shows an item's description while its page is
// fetched.
const FetchingNote = "_Fetching more from the docs…_"

// DefaultBaseURL is the docs site.
const DefaultBaseURL = "https://docs.terragrunt.com"

const (
	// DefaultMaxAge is how old a page can get before it is checked for changes
	// again. A check of an unchanged page downloads nothing.
	DefaultMaxAge = time.Hour
	// DefaultRetryAfter is how long to wait after a failed fetch before
	// fetching the same page again.
	DefaultRetryAfter = 5 * time.Minute
	// FetchTimeout bounds each fetch.
	FetchTimeout = 10 * time.Second

	// CacheVersion names the subdirectory of [Options.CacheDir] that pages are
	// cached in. Change it whenever the cache's layout or content changes in a
	// way that an older terragrunt-ls can't read, so that each version keeps
	// its own cache.
	CacheVersion = "v1"

	maxConcurrentFetches = 2
	cacheDirPerm         = 0o755
)

// Options configures [New].
type Options struct {
	// Snapshot holds the index pages to start from, at `units.md` and
	// `stacks.md`, e.g. [SnapshotFS].
	Snapshot fs.FS
	// BaseURL is the docs site, which links point at and pages are fetched
	// from.
	BaseURL *url.URL
	// Client fetches pages from BaseURL. Nil disables fetching.
	Client *http.Client
	// Logger reports failed fetches and cache writes.
	Logger logger.Logger
	// CacheDir is the directory fetched pages are cached in, under a
	// subdirectory named [CacheVersion]. Empty disables the disk cache.
	CacheDir string
	// MaxAge is how old a page can get before it is checked for changes again.
	// Zero means [DefaultMaxAge].
	MaxAge time.Duration
	// RetryAfter is how long to wait after a failed fetch before fetching the
	// same page again. Zero means [DefaultRetryAfter].
	RetryAfter time.Duration
}

// Docs is the HCL reference. Its methods are safe for concurrent use.
type Docs struct {
	indexes  map[Config]map[itemKey]Item
	pages    map[itemKey]page
	inflight map[string]struct{}
	failed   map[string]time.Time
	sem      chan struct{}
	opts     Options
	wg       sync.WaitGroup
	mu       sync.Mutex
}

type itemKey struct {
	kind Kind
	name string
}

// page is the hover summary of an item's page.
type page struct {
	fetched time.Time
	summary string
}

// SnapshotFS returns the index pages compiled into terragrunt-ls. `go
// generate` updates them from the docs site.
func SnapshotFS() fs.FS {
	sub, err := fs.Sub(snapshot, "snapshot")
	if err != nil {
		panic(fmt.Sprintf("snapshot directory is a constant valid path: %v", err))
	}

	return sub
}

// New returns the reference with the index pages from opts.Snapshot, or from
// opts.CacheDir where a previous fetch cached them. It fetches nothing; call
// [Docs.Refresh] for that.
func New(options *Options) (*Docs, error) {
	opts := *options

	if opts.MaxAge == 0 {
		opts.MaxAge = DefaultMaxAge
	}

	if opts.RetryAfter == 0 {
		opts.RetryAfter = DefaultRetryAfter
	}

	if opts.CacheDir != "" {
		opts.CacheDir = filepath.Join(opts.CacheDir, CacheVersion)
	}

	d := &Docs{
		opts:     opts,
		indexes:  map[Config]map[itemKey]Item{},
		pages:    map[itemKey]page{},
		inflight: map[string]struct{}{},
		failed:   map[string]time.Time{},
		sem:      make(chan struct{}, maxConcurrentFetches),
	}

	for _, cfg := range Configs {
		body, err := fs.ReadFile(opts.Snapshot, string(cfg)+".md")
		if err != nil {
			return nil, fmt.Errorf("reading the %s index snapshot: %w", cfg, err)
		}

		items, err := ParseIndex(body)
		if err != nil {
			return nil, fmt.Errorf("parsing the %s index snapshot: %w", cfg, err)
		}

		d.setIndex(cfg, items)

		if cached, _, ok := d.readCache(indexCachePath(cfg)); ok {
			if items, err := ParseIndex(cached); err == nil {
				d.setIndex(cfg, items)
			}
		}
	}

	return d, nil
}

// Refresh fetches, in the background, each index page whose cached copy is
// missing or older than the maximum age. Cancelling ctx cancels the fetches,
// so pass the server's context, not a request's.
func (d *Docs) Refresh(ctx context.Context) {
	for _, cfg := range Configs {
		if _, modified, ok := d.readCache(indexCachePath(cfg)); ok && time.Since(modified) < d.opts.MaxAge {
			continue
		}

		d.fetch(ctx, indexURLPath(cfg), indexCachePath(cfg), func(body []byte) error {
			items, err := ParseIndex(body)
			if err != nil {
				return err
			}

			d.mu.Lock()
			defer d.mu.Unlock()

			d.setIndex(cfg, items)

			return nil
		})
	}
}

// Wait blocks until every background fetch has finished.
func (d *Docs) Wait() {
	d.wg.Wait()
}

// Lookup returns the item of kind with name, when it's valid in cfg.
func (d *Docs) Lookup(cfg Config, kind Kind, name string) (Item, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	item, ok := d.indexes[cfg][itemKey{kind: kind, name: name}]

	return item, ok
}

// Blurb returns the description of the item of kind with name and a link to
// its page, as Markdown, when the item is valid in cfg.
func (d *Docs) Blurb(cfg Config, kind Kind, name string) (string, bool) {
	item, ok := d.Lookup(cfg, kind, name)
	if !ok {
		return "", false
	}

	return d.blurb(item), true
}

// Hover returns the summary of the page of the item of kind with name, as
// Markdown, when the item is valid in cfg. Until the page is fetched, it
// returns the item's [Docs.Blurb], with [FetchingNote] while it fetches the
// page in the background. Cancelling ctx cancels the fetch, so pass the
// server's context, not a request's.
func (d *Docs) Hover(ctx context.Context, cfg Config, kind Kind, name string) (string, bool) {
	item, ok := d.Lookup(cfg, kind, name)
	if !ok {
		return "", false
	}

	key := itemKey{kind: kind, name: name}

	p, found := d.page(key)

	fetching := false
	if !found || time.Since(p.fetched) >= d.opts.MaxAge {
		fetching = d.fetchPage(ctx, key)
	}

	switch {
	case found:
		return p.summary + "\n\n" + d.link(item), true
	case fetching:
		return d.blurb(item) + "\n\n" + FetchingNote, true
	default:
		return d.blurb(item), true
	}
}

func (d *Docs) blurb(item Item) string {
	return item.Description + "\n\n" + d.link(item)
}

// page returns the summary of an item's page from memory, or from the disk
// cache.
func (d *Docs) page(key itemKey) (page, bool) {
	d.mu.Lock()
	p, ok := d.pages[key]
	d.mu.Unlock()

	if ok {
		return p, true
	}

	body, modified, ok := d.readCache(pageCachePath(key))
	if !ok {
		return page{}, false
	}

	p = page{summary: Summarize(body, d.pageURL(key.kind, key.name)), fetched: modified}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.pages[key] = p

	return p, true
}

// fetchPage fetches an item's page in the background, and reports whether a
// fetch of it is running.
func (d *Docs) fetchPage(ctx context.Context, key itemKey) bool {
	return d.fetch(ctx, pageURLPath(key), pageCachePath(key), func(body []byte) error {
		summary := Summarize(body, d.pageURL(key.kind, key.name))

		d.mu.Lock()
		defer d.mu.Unlock()

		d.pages[key] = page{summary: summary, fetched: time.Now()}

		return nil
	})
}

// fetch calls [Docs.refresh] in the background, and reports whether a fetch of
// the page is running. It starts none while one is running, or when the last
// one failed recently.
func (d *Docs) fetch(ctx context.Context, urlPath, cachePath string, apply func(body []byte) error) bool {
	if d.opts.Client == nil {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, running := d.inflight[urlPath]; running {
		return true
	}

	if time.Since(d.failed[urlPath]) < d.opts.RetryAfter {
		return false
	}

	d.inflight[urlPath] = struct{}{}

	d.wg.Go(func() {
		defer func() {
			d.mu.Lock()
			defer d.mu.Unlock()

			delete(d.inflight, urlPath)
		}()

		d.sem <- struct{}{}
		defer func() { <-d.sem }()

		if err := d.refresh(ctx, urlPath, cachePath, apply); err != nil {
			d.mu.Lock()
			d.failed[urlPath] = time.Now()
			d.mu.Unlock()

			d.opts.Logger.Warn("Failed to fetch docs page", "path", urlPath, "error", err)
		}
	})

	return true
}

// refresh fetches the page at urlPath, passes it to apply, and caches it at
// cachePath once apply accepts it. When the copy cached at cachePath is still
// current, it passes that copy to apply and marks it as checked now, or drops
// the copy's ETag if apply rejects it, so that the next fetch downloads the
// page.
func (d *Docs) refresh(ctx context.Context, urlPath, cachePath string, apply func(body []byte) error) error {
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()

	var etag string

	cached, _, hasCached := d.readCache(cachePath)
	if hasCached {
		if tag, _, ok := d.readCache(etagCachePath(cachePath)); ok {
			etag = validETag(string(tag))
		}
	}

	page, err := Fetch(ctx, d.opts.Client, d.opts.BaseURL, urlPath, etag)
	if errors.Is(err, ErrNotModified) {
		if err := apply(cached); err != nil {
			return errors.Join(err, d.removeCache(etagCachePath(cachePath)))
		}

		return d.touchCache(cachePath)
	}

	if err != nil {
		return err
	}

	if err := apply(page.Body); err != nil {
		return err
	}

	if err := d.cachePage(cachePath, page); err != nil {
		d.opts.Logger.Warn("Failed to cache docs page", "path", urlPath, "error", err)
	}

	return nil
}

// cachePage caches a page's body at rel, and its ETag beside it.
func (d *Docs) cachePage(rel string, page Page) error {
	if err := d.writeCache(rel, page.Body); err != nil {
		return err
	}

	if page.ETag == "" {
		return d.removeCache(etagCachePath(rel))
	}

	return d.writeCache(etagCachePath(rel), []byte(page.ETag))
}

// setIndex replaces the index of cfg. The caller holds d.mu, or has not
// shared d yet.
func (d *Docs) setIndex(cfg Config, items []Item) {
	index := make(map[itemKey]Item, len(items))
	for _, item := range items {
		index[itemKey{kind: item.Kind, name: item.Name}] = item
	}

	d.indexes[cfg] = index
}

func (d *Docs) link(item Item) string {
	return "[Terragrunt docs](" + d.pageURL(item.Kind, item.Name).String() + ")"
}

// pageURL returns the URL of an item's page on the docs site.
func (d *Docs) pageURL(kind Kind, name string) *url.URL {
	return d.opts.BaseURL.ResolveReference(&url.URL{Path: itemPagePath(kind, name)})
}

// readCache returns a cached page and when it was cached.
func (d *Docs) readCache(rel string) ([]byte, time.Time, bool) {
	if d.opts.CacheDir == "" {
		return nil, time.Time{}, false
	}

	name := filepath.Join(d.opts.CacheDir, filepath.FromSlash(rel))

	info, err := os.Stat(name)
	if err != nil || info.Size() > MaxPageBytes {
		return nil, time.Time{}, false
	}

	body, err := os.ReadFile(name)
	if err != nil {
		return nil, time.Time{}, false
	}

	return body, info.ModTime(), true
}

// writeCache caches a page, replacing the cached copy in one rename so that a
// reader never sees part of it.
func (d *Docs) writeCache(rel string, body []byte) error {
	if d.opts.CacheDir == "" {
		return nil
	}

	name := filepath.Join(d.opts.CacheDir, filepath.FromSlash(rel))
	dir := filepath.Dir(name)

	if err := os.MkdirAll(dir, cacheDirPerm); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}

	_, writeErr := tmp.Write(body)

	if err := errors.Join(writeErr, tmp.Close()); err != nil {
		return errors.Join(err, os.Remove(tmp.Name()))
	}

	if err := os.Rename(tmp.Name(), name); err != nil {
		return errors.Join(err, os.Remove(tmp.Name()))
	}

	return nil
}

// touchCache marks a cached page as fetched now.
func (d *Docs) touchCache(rel string) error {
	if d.opts.CacheDir == "" {
		return nil
	}

	now := time.Now()

	return os.Chtimes(filepath.Join(d.opts.CacheDir, filepath.FromSlash(rel)), now, now)
}

// removeCache removes a cached file, if there is one.
func (d *Docs) removeCache(rel string) error {
	if d.opts.CacheDir == "" {
		return nil
	}

	err := os.Remove(filepath.Join(d.opts.CacheDir, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}

func etagCachePath(rel string) string {
	return rel + ".etag"
}

func indexURLPath(cfg Config) string {
	return "/reference/hcl/" + string(cfg) + ".md"
}

func indexCachePath(cfg Config) string {
	return string(cfg) + ".md"
}

func pageURLPath(key itemKey) string {
	return "/reference/hcl/" + string(key.kind) + "/" + key.name + ".md"
}

func pageCachePath(key itemKey) string {
	return path.Join(string(key.kind), key.name+".md")
}

func itemPagePath(kind Kind, name string) string {
	return "/reference/hcl/" + string(kind) + "/" + name + "/"
}
