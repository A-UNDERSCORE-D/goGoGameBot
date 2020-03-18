package tomlconf

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml"
)

var tests = []struct {
	name          string
	tomlStr       string
	IsValid       bool
	dumpJSON      bool
	expectedError string
	expectedConf  *Config
}{
	{
		name:    "minimum valid",
		IsValid: true,
		tomlStr: `
		[connection]
		type = "null"
		`,
		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
		},
	}, {
		name:          "empty",
		IsValid:       false,
		expectedError: "invalid config for connection type \"\", missing config",
	},
	{
		name:    "bad conn",
		IsValid: false,
		tomlStr: `
		[connection]
		type = "IRC"
		`,
		expectedError: "invalid config for connection type \"IRC\", missing config",
	}, {
		name:          "invalid with game",
		IsValid:       false,
		expectedError: `invalid config for game "test". Missing transport`,
		tomlStr: `
		[connection]
		type = "null"
		
		[games.test]
		`,
	}, {
		name:     "valid with game",
		IsValid:  true,
		dumpJSON: true,
		tomlStr: `
		[connection]
			type = "null"
		
		[games.test]
			[games.test.transport]
			type = "process"
			exec.binary = "asd"
		`,
		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
			Games: map[string]*Game{
				"test": &Game{
					Transport: ConfigHolder{
						Type: "process",
						RealConf: tomlTreeFromMapMust(map[string]interface{}{
							"type": "process",
							"exec": map[string]interface{}{
								"binary": "asd",
							},
						}),
					},
				},
			},
		},
	},

	{
		name:     "import simple",
		IsValid:  true,
		dumpJSON: true,
		tomlStr: `
		[connection]
			type = "null"

		[format_templates.test]
			message = "message template test"
			join = "join template test"
			part = "part template test"
			nick = "nick template test"
			quit = "quit template test"
			kick = "kick template test"
			extra.test_one = "test_one: asd"

		[regexp_templates.testSet.test_regexp]
			format = "this is a regexp test"
			regexp = "this regexp test has a regexp"

		[games.whatever]
			import_format = "test"
			import_regexps = ["test_regexp"]

			[games.whatever.transport]
				type = "process"
				execute.binary = "1337ThisDoesntExistAndWillProbablyNeverExist"
		`,
	}, {
		name:    "big valid",
		IsValid: true,
		// TODO: more of this, needs two games, and a complex conn
		tomlStr: `
		[connection]
		type = "null"
		`,
	},
}

func errStrOr(e error, def string) string {
	if e != nil {
		return e.Error()
	}

	return def
}

func all(cmps ...bool) bool {
	for _, c := range cmps {
		if !c {
			return false
		}
	}
	return true
}

func cmpConfig(a, b *Config) (out bool) {
	defer func() {
		if err := recover(); err != nil && strings.Contains(err.(error).Error(), "nil pointer dereference") {
			out = false
		} else {
			panic(err)
		}
	}()

	if a == nil && b == nil {
		return true
	}

	if a.OriginalPath != b.OriginalPath {
		return false
	}

	if a.Connection.Type != b.Connection.Type {
		return false
	}

	if a.Connection.RealConf != nil && b.Connection.RealConf != nil {
		if a.Connection.RealConf.String() != b.Connection.RealConf.String() {
			return false
		}
	}

	if !reflect.DeepEqual(a.FormatTemplates, b.FormatTemplates) {
		return false
	}

	if !reflect.DeepEqual(a.RegexpTemplates, b.RegexpTemplates) {
		return false
	}

	if len(a.Games) != len(b.Games) || (a.Games == nil) != (b.Games == nil) {
		return false
	}

	for name, g := range a.Games {
		other, exists := b.Games[name]
		if !exists {
			return false
		}
		if other != g {
			return false
		}
	}
	return false
}

func cmpGame(a, b *Game) bool {
	if a == nil && a != b {
		return false
	}
	return false
}

func jsonMust(res []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return res
}

func tomlTreeFromMapMust(in map[string]interface{}) *toml.Tree {
	tree, err := toml.TreeFromMap(in)
	if err != nil {
		panic(err)
	}
	return tree
}

func TestValidateConfig(t *testing.T) { //nolint:gocognit // Its just one func
	for _, v := range tests {
		//nolint:scopelint // tests run like this are safe iteration wise.
		t.Run(v.name, func(t *testing.T) {
			x, err := toml.Load(v.tomlStr)
			if err != nil {
				t.Fatal(err)
			}
			conf, err := makeConfig(x)
			if err != nil {
				t.Fatal(err)
			}
			valid := validateConfig(conf)
			isValid := valid != nil

			if v.dumpJSON {
				res, err := json.MarshalIndent(conf, "", "    ")
				if err != nil {
					panic(err)
				}
				t.Log(string(res))
			}

			// if we are valid and expected to be, hop out, otherwise, check
			// that we are invalid in the expected way
			if isValid == v.IsValid || v.expectedError != errStrOr(valid, "") {
				t.Fatalf(
					"Config validity not as expected: valid: %t, expected %t, error: %q, expected %q",
					valid == nil, v.IsValid,
					errStrOr(valid, ""), v.expectedError,
				)
			} else if v.expectedConf != nil && !cmpConfig(v.expectedConf, conf) {
				t.Fatalf(
					"Expected config and resulting config did not match:\n%#v\n------\ndoes not equal\n------\n%#v",
					v.expectedConf,
					conf,
				)
			}
		})
	}
}

func TestInclusion(t *testing.T) {
	panic("so I heard you like panics")
}
