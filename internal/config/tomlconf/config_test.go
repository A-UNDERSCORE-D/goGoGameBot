package tomlconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml"
)

var (
	// String pointers (Must be used for some tests)
	strTest = "test"
)

const (
	minViableToml = `
	[connection]
	type = "null"
	`
)

type confTest struct {
	name          string
	tomlStr       string
	IsValid       bool
	dumpJSON      bool
	expectedError string
	expectedConf  *Config
}

func strPointer(in string) *string { return &in }

var tests = []confTest{
	{
		name:    "minimum valid",
		IsValid: true,
		tomlStr: minViableToml,
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
		
		[[game]]
		name = "test"
		`,
	}, {
		name:    "valid with game",
		IsValid: true,
		tomlStr: `
		[connection]
			type = "null"
		
		[[game]]
			name = "test"
			
			[game.transport]
			type = "process"

			conf.binary = "asd"
		`,
		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
			Games: []*Game{
				{
					Name: "test",
					Transport: ConfigHolder{
						Type: "process",
						RealConf: tomlTreeFromMapMust(map[string]interface{}{
							"binary": "asd",
						}),
					},
				},
			},
		},
	},

	{
		name:    "import simple",
		IsValid: true,
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

		[[regexp_templates.test_regexps1]]
			name   = "test_regexp"
			format = "this is a regexp test"
			regexp = "this regexp test has a regexp"


		[[game]]
		name = "whatever"
		import_regexps = ["test_regexps1"]
		
			[game.chat]
			import_format = "test"

			[game.transport]
			type = "process"
			conf.binary = "1337ThisDoesntExistAndWillProbablyNeverExist"
		`,

		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
			FormatTemplates: map[string]FormatSet{
				"test": {
					Message: strPointer("message template test"),
					Join:    strPointer("join template test"),
					Part:    strPointer("part template test"),
					Nick:    strPointer("nick template test"),
					Quit:    strPointer("quit template test"),
					Kick:    strPointer("kick template test"),
					Extra:   map[string]string{"test_one": "test_one: asd"},
				},
			},
			RegexpTemplates: map[string][]Regexp{
				"test_regexps1": {
					{
						Name:   "test_regexp",
						Format: "this is a regexp test",
						Regexp: "this regexp test has a regexp",
					},
				},
			},
			Games: []*Game{
				{
					Name: "whatever",
					Chat: Chat{
						Formats: FormatSet{
							Message: strPointer("message template test"),
							Join:    strPointer("join template test"),
							Part:    strPointer("part template test"),
							Nick:    strPointer("nick template test"),
							Quit:    strPointer("quit template test"),
							Kick:    strPointer("kick template test"),
							Extra:   map[string]string{"test_one": "test_one: asd"},
						},

						ImportFormat: &strTest,
					},
					Transport: ConfigHolder{
						Type: "process",
						RealConf: tomlTreeFromMapMust(
							map[string]interface{}{"binary": "1337ThisDoesntExistAndWillProbablyNeverExist"},
						),
					},

					RegexpImports: []string{"test_regexps1"},
					Regexps: []Regexp{
						{
							Name:   "test_regexp",
							Format: "this is a regexp test",
							Regexp: "this regexp test has a regexp",
						},
					},

					// map[string]Regexp{
					// 	"test_regexp": {
					// 		Format: "this is a regexp test",
					// 		Regexp: "this regexp test has a regexp",
					// 	},
					// },
				},
			},
		},
	}, {
		name: "invalid import",
		tomlStr: minViableToml + `
		
		[[game]]
		name = "test"
		import_regexps = ["this_doesn't_exist"]
			[game.transport]
			type = "process"
			conf.whatever = "asd"

		`,
		IsValid:       true,
		expectedError: `unable to resolve imports for "test": could not resolve regexp import "this_doesn't_exist" as it does not exist`,
		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
			Games: []*Game{{
				Name: "test",
				Transport: ConfigHolder{
					Type: "process",
					RealConf: tomlTreeFromMapMust(map[string]interface{}{
						"whatever": "asd",
					}),
				},
				RegexpImports: []string{
					"this_doesn't_exist",
				}},
			},
		},
	}, {
		name:          "invalid TOML",
		tomlStr:       `this doesn't work`,
		IsValid:       false,
		expectedError: `(1, 6): was expecting token =, but got unclosed string instead`,
	}, {
		name:    "command import",
		IsValid: true,
		tomlStr: minViableToml + `
		[command_templates.root.one]
		format = "test"
		help = "tests things"
		requires_admin = 1337

		[[game]]
		name = "test"
		import_commands = ["root"]
		
			[game.transport]
			type = "process"
			conf.asd = "asd"
		`,
		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
			CommandTemplates: map[string]map[string]Command{
				"root": {
					"one": {
						Format:        "test",
						Help:          "tests things",
						RequiresAdmin: 1337,
					},
				},
			},
			Games: []*Game{{
				Name: "test",
				Transport: ConfigHolder{Type: "process", RealConf: tomlTreeFromMapMust(
					map[string]interface{}{"asd": "asd"},
				)},
				CommandImports: []string{"root"},
				Commands: map[string]Command{
					"one": {Format: "test", Help: "tests things", RequiresAdmin: 1337},
				},
			}},
		},
	}, {
		name:          "bad command import",
		IsValid:       true,
		expectedError: `unable to resolve imports for "": could not resolve command import "this doesn't exist" as it does not exist`,
		tomlStr: minViableToml + `
		[[game]]
		import_commands = ["this doesn't exist"]
			[game.transport]
			type = "process"
			conf.asd = "asd"
		`,
	}, {
		name:          "bad format import",
		IsValid:       true,
		expectedError: `unable to resolve imports for "": could not resolve format import "this doesn't exist either" as it does not exist`,
		tomlStr: minViableToml + `
		[[game]]
			[game.chat]
			import_format = "this doesn't exist either"

			[game.transport]
			type = "process"
			conf.asd = "asd"
		`,
	}, {
		name:    "large complex",
		IsValid: true,
		tomlStr: `
		[connection]
		type = "IRC"
			[connection.conf]
			nick = "goGoGameBot"
			ident = "gggb"
			gecos = "golang rocks"

			host = "irc.goGoGameBot.golangrocks"
			port = 1337

		[[connection.admin]]
		mask = "*!*@golang_rocks"
		# TODO: the rest of this, 2 games, multiple imports of different types

			
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
		} else if err != nil {
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

outer:
	for _, g := range a.Games {
		for _, g2 := range b.Games {
			if cmpGame(g, g2) {
				continue outer
			}
		}
		return false
	}

	return true
}

var gameNumFields = func() int { return reflect.TypeOf(Game{}).NumField() }()

func cmpGame(a, b *Game) bool {
	if (a == nil || b == nil) && a != b {
		return false
	}

	// Sanity check to make sure this wasn't updated/changed
	if reflect.TypeOf(a).Elem().NumField() != 11 {
		panic(errors.New("tomlconf.Game updated but tests not"))
	}

	// Manual: Transport
	// DeepEqualled: Chat, CommandImports, Commands, RegexpImports, Regexps
	if a.Name != b.Name ||
		a.Comment != b.Comment ||
		a.AutoStart != b.AutoStart ||
		a.AutoRestart != b.AutoRestart ||
		a.PreRoll != b.PreRoll ||
		a.Transport.Type != b.Transport.Type ||
		a.Transport.RealConf.String() != b.Transport.RealConf.String() ||
		!reflect.DeepEqual(a.Chat, b.Chat) ||
		!reflect.DeepEqual(a.CommandImports, b.CommandImports) ||
		!reflect.DeepEqual(a.Commands, b.Commands) ||
		!reflect.DeepEqual(a.RegexpImports, a.RegexpImports) ||
		!reflect.DeepEqual(a.Regexps, b.Regexps) {

		return false
	}
	return true
}

func jsonMust(res []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return res
}

func jsonMustMarshalIndent(in interface{}) string {
	return string(jsonMust(json.MarshalIndent(in, "", "    ")))
}

func tomlTreeFromMapMust(in map[string]interface{}) *toml.Tree {
	tree, err := toml.TreeFromMap(in)
	if err != nil {
		panic(err)
	}
	return tree
}

func diff(a, b string) {
	aSplit := strings.Split(a, "\n")
	bSplit := strings.Split(b, "\n")

	for i, v := range aSplit {
		if len(bSplit) <= i {
			fmt.Println("+", v)
			fmt.Println("--------------------------")
			continue
		}

		lineDiff(v, bSplit[i])

	}
}

func lineDiff(a, b string) {
	if a == b {
		fmt.Println(a)
		return
	}

	fmt.Println("+", a)
	fmt.Println("-", b)
}
func TestValidateConfig(t *testing.T) { //nolint:gocognit // Its just one func
	for _, v := range tests {
		//nolint:scopelint // tests run like this are safe iteration wise.
		t.Run(v.name, func(t *testing.T) {
			x, err := toml.Load(v.tomlStr)
			if err != nil {
				if v.expectedError == errStrOr(err, "") {
					return
				}
				t.Fatal(err)
			}
			conf, err := configFromTree(x)
			if err != nil {
				t.Fatal(err)
			}

			validationError := validateConfig(conf)

			if validationError == nil && v.IsValid {
				// We are valid, and expected to be such
				return
			} else if validationError != nil && v.expectedError == errStrOr(validationError, "") {
				// we are invalid, and the error we expected matches what we got
				return
			}

			t.Fatalf(
				"Config validity not as expected. Expected validity %t, got %t. Expected error %q, got %q",
				v.IsValid, validationError == nil,
				v.expectedError, validationError,
			)
		})
	}
}

func TestInclusion(t *testing.T) {
	for _, tt := range tests {
		if !tt.IsValid {
			continue // Cant test inclusion on invalid configs.
		}
		t.Run(tt.name, func(t *testing.T) {
			tree, err := toml.Load(tt.tomlStr)
			if err != nil {
				t.Fatalf("Could not parse toml string: %s", err)
			}

			conf, err := configFromTree(tree)
			if err != nil {
				t.Fatalf("could not unmarshal tree into config: %s", err)
			}
			err = conf.resolveImports()
			if err != nil {
				if tt.expectedError != "" && err.Error() == tt.expectedError {
					return // It behaved as expected
				}
				t.Fatalf("Could not resolve imports: %s", err)
			}
			if !cmpConfig(conf, tt.expectedConf) {
				diff(jsonMustMarshalIndent(conf), jsonMustMarshalIndent(tt.expectedConf))
				t.Fatalf("Config did not match expected config")
			}
		})
	}
}
