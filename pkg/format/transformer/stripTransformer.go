package transformer

// StripTransformer is a simple transformer that simply removes all intermediate form formatting codes it sees
type StripTransformer struct{}

// Transform strips all formatting codes from the passed string
func (s StripTransformer) Transform(in string) string { return Strip(in) }

// MakeIntermediate strips all formatting codes from the passed string
func (s StripTransformer) MakeIntermediate(in string) string { return Strip(in) }
