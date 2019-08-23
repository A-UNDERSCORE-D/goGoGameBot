package formatTransformer

// Transformer refers to a string transformer. String Transformers convert messages from an intermediate format
// to a protocol specific format
type Transformer interface {
	// Transform takes a string in the intermediate format and converts it to its specific format
	Transform(in string) string
	// MakeIntermediate takes a string in a Transformer specific format and converts it to the Intermediate  format
	MakeIntermediate(in string) string
}
