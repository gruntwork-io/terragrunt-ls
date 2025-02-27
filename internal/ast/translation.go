package ast

import (
	"github.com/hashicorp/hcl/v2"
	"go.lsp.dev/protocol"
)

func FromHCLRange(s hcl.Range) protocol.Range {
	return protocol.Range{
		Start: FromHCLPos(s.Start),
		End:   FromHCLPos(s.End),
	}
}

func FromHCLPos(s hcl.Pos) protocol.Position {
	return protocol.Position{
		Line:      uint32(max(s.Line-1, 0)),
		Character: uint32(max(s.Column-1, 0)),
	}
}

func ToHCLRange(s protocol.Range) hcl.Range {
	return hcl.Range{
		Filename: "",
		Start:    ToHclPos(s.Start),
		End:      ToHclPos(s.End),
	}
}

func ToHclPos(s protocol.Position) hcl.Pos {
	return hcl.Pos{
		Line:   int(s.Line + 1),
		Column: int(s.Character + 1),
		Byte:   0,
	}
}
