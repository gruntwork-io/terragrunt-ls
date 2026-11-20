package ast

import (
	"strings"
	"unicode/utf16"

	"github.com/hashicorp/hcl/v2"
	"go.lsp.dev/protocol"
)

// FromHCLRange converts r, a range in doc, to an LSP range. See [FromHCLPos].
func FromHCLRange(doc string, r hcl.Range) protocol.Range {
	return protocol.Range{
		Start: FromHCLPos(doc, r.Start),
		End:   FromHCLPos(doc, r.End),
	}
}

// FromHCLPos converts pos, a position in doc, to an LSP position. LSP counts characters in
// UTF-16 code units, so the conversion measures the text before pos.Byte on its line.
//
// A position whose byte offset is not on its line in doc, such as the zero hcl.Pos that
// some diagnostics carry, has nothing to measure and converts by line and column alone.
func FromHCLPos(doc string, pos hcl.Pos) protocol.Position {
	start, line, ok := lineAt(doc, pos.Line-1)
	if !ok || pos.Byte < start || pos.Byte > start+len(line) {
		return protocol.Position{
			Line:      uint32(max(pos.Line-1, 0)),
			Character: uint32(max(pos.Column-1, 0)),
		}
	}

	return protocol.Position{
		Line:      uint32(pos.Line - 1),
		Character: uint32(utf16Len(doc[start:pos.Byte])),
	}
}

// ByteOffset returns the byte offset in doc of p, an LSP position. Compare it with the Byte
// of an HCL position, never the Column: HCL counts the grapheme clusters of each token on its
// own, which no conversion from an LSP position reproduces.
//
// A character between the halves of a surrogate pair moves to the start of the pair. A
// character past the end of its line moves to the end of the line, as LSP requires. A line
// past the end of doc maps to the end of doc.
func ByteOffset(doc string, p protocol.Position) int {
	start, line, ok := lineAt(doc, int(p.Line))
	if !ok {
		return len(doc)
	}

	return start + LineByteOffset(line, p.Character)
}

// LineByteOffset returns the byte offset in line of the LSP character, which counts UTF-16
// code units. A character between the halves of a surrogate pair moves to the start of the
// pair, and a character past the end of the line moves to the end of the line.
func LineByteOffset(line string, character uint32) int {
	units := 0

	for i, r := range line {
		if units >= int(character) {
			return i
		}

		units += utf16.RuneLen(r)
		if units > int(character) {
			return i
		}
	}

	return len(line)
}

// lineAt returns the byte offset and text of the zero-based line n of doc, without its line
// ending. Lines end at "\n" or "\r\n", as HCL counts them.
func lineAt(doc string, n int) (int, string, bool) {
	if n < 0 {
		return 0, "", false
	}

	start := 0

	for range n {
		i := strings.IndexByte(doc[start:], '\n')
		if i < 0 {
			return 0, "", false
		}

		start += i + 1
	}

	line, _, found := strings.Cut(doc[start:], "\n")
	if found {
		line = strings.TrimSuffix(line, "\r")
	}

	return start, line, true
}

// utf16Len returns the length of s in UTF-16 code units.
func utf16Len(s string) int {
	units := 0
	for _, r := range s {
		units += utf16.RuneLen(r)
	}

	return units
}
