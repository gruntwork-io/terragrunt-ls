package ast

import (
	"github.com/hashicorp/hcl/v2"
	"go.lsp.dev/protocol"
)

// FromHCLRange converts a hcl.Range to a LSP protocol.Range.
func FromHCLRange(s hcl.Range) protocol.Range {
	return protocol.Range{
		Start: FromHCLPos(s.Start),
		End:   FromHCLPos(s.End),
	}
}

// FromHCLPos converts a hcl.Pos to a LSP protocol.Position.
func FromHCLPos(s hcl.Pos) protocol.Position {
	return protocol.Position{
		Line:      uint32(max(s.Line-1, 0)),
		Character: uint32(max(s.Column-1, 0)),
	}
}

// ToHCLRange converts a LSP protocol.Range to a hcl.Range.
func ToHCLRange(s protocol.Range) hcl.Range {
	return hcl.Range{
		Filename: "",
		Start:    ToHCLPos(s.Start),
		End:      ToHCLPos(s.End),
	}
}

// ToHCLPos converts a LSP protocol.Position to a hcl.Pos.
func ToHCLPos(s protocol.Position) hcl.Pos {
	return hcl.Pos{
		Line:   int(s.Line + 1),
		Column: int(s.Character + 1),
		Byte:   0,
	}
}