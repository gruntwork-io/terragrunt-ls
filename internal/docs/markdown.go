package docs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
)

// MaxIndexItems is the most items [ParseIndex] reads from an index page.
const MaxIndexItems = 1000

var (
	// ErrEmptyIndex is returned by [ParseIndex] for a page that lists no items.
	ErrEmptyIndex = errors.New("index page lists no HCL items")
	// ErrTooManyItems is returned by [ParseIndex] for a page that lists more
	// than [MaxIndexItems] items.
	ErrTooManyItems = fmt.Errorf("index page lists more than %d HCL items", MaxIndexItems)
)

// indexRow matches a table row of an index page, e.g.
// | [`get_env`](/reference/hcl/functions/get_env/) | Returns the value of an environment variable. |
var indexRow = regexp.MustCompile(
	"^\\|\\s*\\[`([a-z][a-z0-9_]{0,127})`\\]\\(/reference/hcl/(blocks|attributes|functions)/([a-z][a-z0-9_]{0,127})/?\\)" +
		"\\s*\\|\\s*(.*?)\\s*\\|\\s*$",
)

// markdownEscape matches a backslash escape of ASCII punctuation.
var markdownEscape = regexp.MustCompile(`\\([!-/:-@\[-` + "`" + `{-~])`)

// ParseIndex returns the items an index page lists, such as
// `/reference/hcl/units.md`, in page order.
//
// Returns [ErrEmptyIndex] when the page lists none, and [ErrTooManyItems]
// when it lists more than [MaxIndexItems].
func ParseIndex(page []byte) ([]Item, error) {
	var items []Item

	scanner := bufio.NewScanner(bytes.NewReader(page))
	for scanner.Scan() {
		m := indexRow.FindStringSubmatch(scanner.Text())
		if m == nil || m[1] != m[3] {
			continue
		}

		if len(items) == MaxIndexItems {
			return nil, ErrTooManyItems
		}

		items = append(items, Item{
			Kind:        Kind(m[2]),
			Name:        m[1],
			Description: markdownEscape.ReplaceAllString(m[4], "$1"),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, ErrEmptyIndex
	}

	return items, nil
}
