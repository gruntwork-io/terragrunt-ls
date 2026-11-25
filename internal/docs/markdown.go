package docs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// MaxIndexItems is the most items [ParseIndex] reads from an index page.
const MaxIndexItems = 1000

var (
	// ErrEmptyIndex is returned by [ParseIndex] and [ParseVariable] for a page that lists no items.
	ErrEmptyIndex = errors.New("index page lists no HCL items")
	// ErrTooManyItems is returned by [ParseIndex] and [ParseVariable] for a page that lists more
	// than [MaxIndexItems] items.
	ErrTooManyItems = fmt.Errorf("index page lists more than %d HCL items", MaxIndexItems)
)

// indexRow matches a table row of an index page, e.g.
// | [`get_env`](/reference/hcl/functions/get_env/) | Returns the value of an environment variable. |
var indexRow = regexp.MustCompile(
	"^\\|\\s*\\[`([a-z][a-z0-9_]{0,127})`\\]\\(/reference/hcl/(blocks|attributes|functions)/([a-z][a-z0-9_]{0,127})/?\\)" +
		"\\s*\\|\\s*(.*?)\\s*\\|\\s*$",
)

// fieldRow matches a row of the Fields list of a variable's page, e.g. "- [`path.unit`](...): ...".
var fieldRow = regexp.MustCompile(
	"^-\\s+\\[`([a-z][a-z0-9_]{0,127})\\.([a-z][a-z0-9_]{0,127})`\\]" +
		"\\(/reference/hcl/variables/([a-z][a-z0-9_]{0,127})/([a-z][a-z0-9_]{0,127})/?\\):\\s*(.*?)\\s*$",
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

// ParseVariable returns v and its fields from its page, or [ErrEmptyIndex] or [ErrTooManyItems] for a bad page.
func ParseVariable(v Variable, page []byte) ([]Item, error) {
	var (
		description, section string
		fields               []Item
	)

	scanner := bufio.NewScanner(bytes.NewReader(page))
	for scanner.Scan() {
		line := scanner.Text()

		if heading, ok := strings.CutPrefix(line, "## "); ok {
			section = strings.TrimSpace(heading)

			continue
		}

		if quote, ok := strings.CutPrefix(line, "> "); ok && section == "" && description == "" {
			description = markdownEscape.ReplaceAllString(strings.TrimSpace(quote), "$1")

			continue
		}

		if section != "Fields" {
			continue
		}

		m := fieldRow.FindStringSubmatch(line)
		if m == nil || m[1] != v.Name || m[3] != v.Name || m[2] != m[4] {
			continue
		}

		if len(fields) == MaxIndexItems {
			return nil, ErrTooManyItems
		}

		fields = append(fields, Item{
			Kind:        KindVariable,
			Name:        v.Name + "." + m[2],
			Description: markdownEscape.ReplaceAllString(m[5], "$1"),
			Experiment:  v.Experiment,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if description == "" || len(fields) == 0 {
		return nil, ErrEmptyIndex
	}

	variable := Item{Kind: KindVariable, Name: v.Name, Description: description, Experiment: v.Experiment}

	return append([]Item{variable}, fields...), nil
}
