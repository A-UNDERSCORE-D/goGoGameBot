package strip

import (
	"awesome-dragon.science/go/goGoGameBot/pkg/format/transformer/tokeniser"
)

// Transformer is a simple transformer that simply removes all intermediate form formatting codes it sees
type Transformer struct{}

// Transform strips all formatting codes from the passed string
func (s Transformer) Transform(in string) string { return tokeniser.Strip(in) }

// MakeIntermediate strips all formatting codes from the passed string
func (s Transformer) MakeIntermediate(in string) string { return tokeniser.Strip(in) }
