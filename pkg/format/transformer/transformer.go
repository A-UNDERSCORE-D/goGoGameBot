package transformer

import (
	"errors"
	"fmt"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config/tomlconf"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/minecraft"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/simple"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/strip"
)

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

// GetTransformer Instantiates a Transformer implementation given a config
func GetTransformer(conf *tomlconf.ConfigHolder) (Transformer, error) {
	if conf == nil {
		return nil, errors.New("cannot get a transformer from a nil config")
	}

	x := strings.ToLower(conf.Type)
	switch x {
	case "strip":
		return new(strip.Transformer), nil
	case "simple":
		stc := new(simple.Conf)
		if err := conf.RealConf.Unmarshal(stc); err != nil {
			return nil, fmt.Errorf("could not create new SimpleTransformer: %w", err)
		}

		return simple.New(stc.MakeMaps()), nil
	case "minecraft":
		return minecraft.Transformer{}, nil
	}

	return nil, fmt.Errorf("unknown transformer type %q", x)
}
