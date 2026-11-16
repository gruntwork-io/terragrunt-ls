package docs

import (
	"net/url"
	"regexp"
	"strings"
)

// linkTarget matches the target of a Markdown link.
var linkTarget = regexp.MustCompile(`\]\(([^)\s]+)\)`)

// section is a `##` section of a page. The part of the page before the first
// one has no heading.
type section struct {
	heading string
	lines   []string
}

// Summarize returns what a hover shows of an item's page, which is at pageURL:
// the description, the attribute's type, the function's signature, the first
// paragraph of the usage, the function's parameters as a list, and the names of
// the block's attributes and nested blocks. Links resolve against pageURL.
func Summarize(page []byte, pageURL *url.URL) string {
	var parts []string

	for _, s := range splitSections(string(page)) {
		switch s.heading {
		case "":
			parts = append(parts, preamble(s.lines)...)
		case "Usage":
			parts = append(parts, usage(s.lines)...)
		case "Parameters":
			parts = append(parts, parameters(s.lines)...)
		case "Attributes", "Blocks":
			if names := subsectionNames(s.lines); len(names) > 0 {
				parts = append(parts, "**"+s.heading+":** "+strings.Join(names, ", "))
			}
		}
	}

	return resolveLinks(strings.Join(parts, "\n\n"), pageURL)
}

// splitSections splits a page at its `##` headings, leaving out its `#`
// title. Headings in code blocks don't split it.
func splitSections(page string) []section {
	lines := strings.Split(strings.ReplaceAll(page, "\r\n", "\n"), "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		lines = lines[1:]
	}

	sections := []section{{}}
	inFence := false

	for _, line := range lines {
		if isFence(line) {
			inFence = !inFence
		}

		if !inFence && strings.HasPrefix(line, "## ") {
			sections = append(sections, section{heading: strings.TrimSpace(strings.TrimPrefix(line, "## "))})

			continue
		}

		last := &sections[len(sections)-1]
		last.lines = append(last.lines, line)
	}

	return sections
}

// preamble returns the description and the attribute's type. It leaves out a
// function's return type, which its signature shows.
func preamble(lines []string) []string {
	var parts []string

	for _, line := range lines {
		switch {
		case strings.TrimSpace(line) == "", strings.HasPrefix(line, "**Returns:**"):
		case strings.HasPrefix(line, "> "):
			parts = append(parts, strings.TrimPrefix(line, "> "))
		default:
			parts = append(parts, line)
		}
	}

	return parts
}

// usage returns the code block the Usage section opens with, when it is a
// function's one-line signature, and the section's first paragraph.
func usage(lines []string) []string {
	var parts []string

	i := skipBlank(lines, 0)
	if i+2 < len(lines) && isFence(lines[i]) && isFence(lines[i+2]) && strings.Contains(lines[i+1], " --> ") {
		parts = append(parts, strings.Join(lines[i:i+3], "\n"))
		i += 3
	}

	for i = skipBlank(lines, i); i < len(lines); i = skipBlank(lines, i+1) {
		line := lines[i]

		switch {
		case strings.HasPrefix(line, "### "):
			return parts
		case isFence(line):
			i = fenceEnd(lines, i)
		case !strings.HasPrefix(line, "|"):
			end := i
			for end < len(lines) && strings.TrimSpace(lines[end]) != "" {
				end++
			}

			return append(parts, strings.Join(lines[i:end], "\n"))
		}
	}

	return parts
}

// parameters returns the rows of the Parameters table as a list.
func parameters(lines []string) []string {
	var items []string

	rows := 0

	for _, line := range lines {
		if !strings.HasPrefix(line, "|") {
			continue
		}

		rows++

		// The first two rows are the header and its separator.
		cells := tableCells(line)
		if rows <= 2 || len(cells) != 4 {
			continue
		}

		name, paramType, required, description := cells[0], cells[1], cells[2], cells[3]
		if required == "No" {
			paramType += ", optional"
		}

		items = append(items, "- "+name+" ("+paramType+"): "+description)
	}

	if len(items) == 0 {
		return nil
	}

	return []string{"**Parameters:**\n" + strings.Join(items, "\n")}
}

// subsectionNames returns the names of the `### `name“ subsections of a
// section, e.g. the attributes of a block.
func subsectionNames(lines []string) []string {
	var names []string

	inFence := false

	for _, line := range lines {
		if isFence(line) {
			inFence = !inFence
		}

		name := strings.TrimPrefix(line, "### ")
		if !inFence && name != line && strings.HasPrefix(name, "`") && strings.HasSuffix(name, "`") {
			names = append(names, name)
		}
	}

	return names
}

// tableCells returns the cells of a Markdown table row, unescaping `\|`.
func tableCells(row string) []string {
	const escapedPipe = "\x00"

	row = strings.TrimSpace(row)
	row = strings.TrimSuffix(strings.TrimPrefix(row, "|"), "|")
	row = strings.ReplaceAll(row, `\|`, escapedPipe)

	cells := strings.Split(row, "|")
	for i, cell := range cells {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(cell, escapedPipe, "|"))
	}

	return cells
}

// resolveLinks resolves the relative link targets in text against base.
func resolveLinks(text string, base *url.URL) string {
	return linkTarget.ReplaceAllStringFunc(text, func(link string) string {
		ref, err := url.Parse(linkTarget.FindStringSubmatch(link)[1])
		if err != nil || ref.IsAbs() {
			return link
		}

		return "](" + base.ResolveReference(ref).String() + ")"
	})
}

func isFence(line string) bool {
	return strings.HasPrefix(line, "```")
}

// fenceEnd returns the index of the line that closes the code block opening at
// start, or the last line when nothing closes it.
func fenceEnd(lines []string, start int) int {
	for i := start + 1; i < len(lines); i++ {
		if isFence(lines[i]) {
			return i
		}
	}

	return len(lines) - 1
}

func skipBlank(lines []string, i int) int {
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	return i
}
