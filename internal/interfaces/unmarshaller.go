package interfaces

// Unmarshaler refers to anything that can unmarshal data into an interface{}
type Unmarshaler interface {
	// Unmarshal unmarshals data from a config or other source into an interface{}
	Unmarshal(interface{}) error
}
