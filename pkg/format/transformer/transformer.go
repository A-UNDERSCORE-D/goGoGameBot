package transformer

// Transformer refers to a string transformer. String Transformers convert messages from an intermediate format
// to a protocol specific format
type Transformer interface {
	// Transform takes a string in the intermediate format and converts it to its specific format.
	// When the implementation of Transformer does not support a given format type, it can either eat it entirely, or
	// use a pure ascii notation to indicate that it was there. For example: strike though could be replaced with ~~
	Transform(in string) string
	// MakeIntermediate takes a string in a Transformer specific format and converts it to the Intermediate format.
	// Any existing sentinels in the string SHOULD be escaped
	MakeIntermediate(in string) string
}
