package tomlconf

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
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

var nullConn = ConfigHolder{Type: "null"}

type confTest struct {
	name          string
	tomlStr       string
	IsValid       bool
	expectedError string
	expectedConf  *Config
}

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

			config.binary = "asd"
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
					Chat: Chat{
						BridgeChat:    true,
						AllowForwards: true,
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
			config.binary = "1337ThisDoesntExistAndWillProbablyNeverExist"
		`,

		expectedConf: &Config{
			Connection: ConfigHolder{Type: "null"},
			FormatTemplates: map[string]FormatSet{
				"test": {
					Message: makeStrPtr("message template test"),
					Join:    makeStrPtr("join template test"),
					Part:    makeStrPtr("part template test"),
					Nick:    makeStrPtr("nick template test"),
					Quit:    makeStrPtr("quit template test"),
					Kick:    makeStrPtr("kick template test"),
					Extra:   map[string]string{"test_one": "test_one: asd"},
				},
			},
			RegexpTemplates: map[string][]Regexp{
				"test_regexps1": {
					{
						Name:         "test_regexp",
						Format:       "this is a regexp test",
						Regexp:       "this regexp test has a regexp",
						Eat:          true,
						SendToChan:   true,
						SendToOthers: true,
					},
				},
			},
			Games: []*Game{
				{
					Name: "whatever",
					Chat: Chat{
						Formats: FormatSet{
							Message: makeStrPtr("message template test"),
							Join:    makeStrPtr("join template test"),
							Part:    makeStrPtr("part template test"),
							Nick:    makeStrPtr("nick template test"),
							Quit:    makeStrPtr("quit template test"),
							Kick:    makeStrPtr("kick template test"),
							Extra:   map[string]string{"test_one": "test_one: asd"},
						},

						ImportFormat:  &strTest,
						AllowForwards: true,
						BridgeChat:    true,
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
							Name:         "test_regexp",
							Format:       "this is a regexp test",
							Regexp:       "this regexp test has a regexp",
							Eat:          true,
							SendToChan:   true,
							SendToOthers: true,
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
			config.whatever = "asd"

		`,
		IsValid:       true,
		expectedError: `unable to resolve imports for "test": could not resolve regexp import "this_doesn't_exist" as it does not exist`, //nolint:lll // Its a string
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
			config.asd = "asd"
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
				Chat: Chat{
					BridgeChat:    true,
					AllowForwards: true,
				},
			}},
		},
	}, {
		name:          "bad command import",
		IsValid:       true,
		expectedError: `unable to resolve imports for "": could not resolve command import "this doesn't exist" as it does not exist`, //nolint:lll // Its a string
		tomlStr: minViableToml + `
		[[game]]
		import_commands = ["this doesn't exist"]
			[game.transport]
			type = "process"
			config.asd = "asd"
		`,
	}, {
		name:          "bad format import",
		IsValid:       true,
		expectedError: `unable to resolve imports for "": could not resolve format import "this doesn't exist either" as it does not exist`, //nolint:lll // Its a string
		tomlStr: minViableToml + `
		[[game]]
			[game.chat]
			import_format = "this doesn't exist either"

			[game.transport]
			type = "process"
			config.asd = "asd"
		`,
	}, {
		name: "format import with override",
		tomlStr: minViableToml + `
		[format_templates]
			[format_templates.test]
			message = "asd"
			join = "join"
			part = "part"

		[[game]]
		name = "test"
			[game.chat]
			import_format = "test"

			[game.chat.formats]
			message = "whatever"
			quit = "Im different"

			[game.transport]
			type = "none"
			config.test = "test"
		`,
		IsValid: true,
		expectedConf: &Config{
			OriginalPath: "",
			Connection:   nullConn,
			FormatTemplates: map[string]FormatSet{
				"test": {
					Message:  makeStrPtr("asd"),
					Join:     makeStrPtr("join"),
					Part:     makeStrPtr("part"),
					Nick:     nil,
					Quit:     nil,
					Kick:     nil,
					External: nil,
					Extra:    nil,
				},
			},
			Games: []*Game{
				{
					Name:      "test",
					Transport: ConfigHolder{Type: "none", RealConf: tomlTreeFromMapMust(map[string]interface{}{"test": "test"})},
					Chat: Chat{
						ImportFormat: makeStrPtr("test"),
						Formats: FormatSet{
							Message:  makeStrPtr("whatever"),
							Join:     makeStrPtr("join"),
							Part:     makeStrPtr("part"),
							Nick:     nil,
							Quit:     makeStrPtr("Im different"),
							Kick:     nil,
							External: nil,
							Extra:    nil,
						},
						BridgeChat:    true,
						AllowForwards: true,
					},
				},
			},
		},
	}, /* {
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
	}, */
}

func errStrOr(e error, def string) string {
	if e != nil {
		return e.Error()
	}

	return def
}

func cmpConfig(a, b *Config) (out bool) { //nolint:gocognit // Its a struct compare, its going to be complex
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

func cmpGame(a, b *Game) bool { //nolint:gocognit // Its a comparison
	if (a == nil || b == nil) && a != b {
		return false
	}

	// Sanity check to make sure this wasn't updated/changed
	if gameNumFields != 11 {
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
		!reflect.DeepEqual(a.RegexpImports, b.RegexpImports) ||
		!reflect.DeepEqual(a.Regexps, b.Regexps) { //nolint:go-lint // Its done this way intentionally

		return false
	}

	return true
}

func tomlTreeFromMapMust(in map[string]interface{}) *toml.Tree {
	tree, err := toml.TreeFromMap(in)
	if err != nil {
		panic(err)
	}

	return tree
}

func diff(a, b string) string {
	out := &strings.Builder{}
	aSplit := strings.Split(a, "\n")
	bSplit := strings.Split(b, "\n")

	for i, v := range aSplit {
		if len(bSplit) <= i {
			fmt.Fprintln(out, "+", v)
			fmt.Fprintln(out, "--------------------------")

			continue
		}

		aLine := aSplit[i]
		bLine := bSplit[i]

		if aLine == bLine {
			fmt.Fprintln(out, aLine)
			continue
		}

		fmt.Fprintln(out, "+", aLine)
		fmt.Fprintln(out, "-", bLine)
	}

	return out.String()
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

func TestInclusion(t *testing.T) { //nolint:gocognit // Its a test
	sConf := spew.NewDefaultConfig()
	sConf.DisablePointerAddresses = true
	sConf.SortKeys = true
	sConf.DisableCapacities = true

	for _, tt := range tests {
		tt := tt
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
				result := diff(sConf.Sdump(conf), sConf.Sdump(tt.expectedConf))
				t.Log("\n", result)
				t.Fatalf("Config did not match expected config")
			}
		})
	}
}

func makeStrPtr(x string) *string { return &x }

func dumpExampleConf(t *testing.T) { //nolint:funlen // Must be long
	realConf, err := toml.TreeFromMap(
		map[string]interface{}{
			"path":              "/opt/games/somegame",
			"args":              "-run_without_crashing",
			"working_directory": "/opt/games/",
			"environment":       []string{"NO_REALLY_DONT_CRASH=1"},
			"copy_env":          true,
		},
	)

	if err != nil {
		panic(err)
	}

	exampleGame := &Game{
		Name:        "example_game",
		AutoStart:   true,
		AutoRestart: 1337,
		Comment:     "Example game",
		Transport: ConfigHolder{
			Type:     "process",
			RealConf: realConf,
		},
		PreRoll: struct {
			Regexp  string
			Replace string
		}{
			Regexp:  ".*",
			Replace: "no data for you",
		},
		Chat: Chat{
			BridgedChannel: "#some_channel",
			ImportFormat:   makeStrPtr("some_format"),
			Formats:        FormatSet{},
			BridgeChat:     true,
			DumpStdout:     false,
			DumpStderr:     false,
			AllowForwards:  true,
			Transformer:    &ConfigHolder{Type: "minecraft"},
		},
		CommandImports: []string{"some_template"},
		Commands: map[string]Command{
			"test": {
				Format:        "/foo",
				Help:          "Runs the foo command (admin only)",
				RequiresAdmin: 3,
			},
		},
		RegexpImports: []string{"some_template"},
		Regexps: []Regexp{{
			Name:     "whatever",
			Regexp:   ".{1337}",
			Format:   "1337",
			Priority: 1,
		}},
	}

	c := Config{
		Connection: ConfigHolder{
			Type:     "irc",
			RealConf: &toml.Tree{},
		},
		FormatTemplates: map[string]FormatSet{
			"some_format": {
				Message:  makeStrPtr("message"),
				Join:     makeStrPtr("join"),
				Part:     makeStrPtr("part"),
				Nick:     makeStrPtr("nick"),
				Quit:     makeStrPtr("quit"),
				Kick:     makeStrPtr("kick"),
				External: makeStrPtr("external"),
				Extra: map[string]string{
					"one": "two",
				},
			},
		},
		RegexpTemplates: map[string][]Regexp{
			"some_template": {
				{
					Name:         "example",
					Regexp:       ".*",
					Format:       "No format",
					Priority:     0,
					Eat:          false,
					SendToChan:   false,
					SendToOthers: false,
					SendToLocal:  false,
				},
			},
		},
		CommandTemplates: map[string]map[string]Command{
			"some_template": {
				"cmd_template": {
					Format:        "/frob",
					Help:          "runs the frob command",
					RequiresAdmin: 0,
				},
			},
		},
		Games: []*Game{exampleGame},
	}

	b, err := toml.Marshal(c)
	if err != nil {
		panic(err)
	}

	t.Error(string(b))
}

func Test_Something(t *testing.T) {
	t.SkipNow()
	dumpExampleConf(t)
}
