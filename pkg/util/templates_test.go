package util

import (
	"reflect"
	"testing"
	"text/template"
)

var fmtStringCleanupTests = []struct {
	name            string
	stripWhitespace bool
	stripNewlines   bool
	formatString    string
	want            string
}{
	{
		"no stripping, single line",
		false,
		false,
		"test string",
		"test string",
	},
	{
		"no stripping, multiline",
		false,
		false,
		"test string\nwith newlines",
		"test string\nwith newlines",
	},
	{
		"all stripping stripping, single line",
		true,
		true,
		"test string",
		"test string",
	},
	{
		"newline stripping, single line",
		false,
		true,
		"test string",
		"test string",
	},
	{
		"whitespace stripping, single line",
		true,
		false,
		"test string",
		"test string",
	},
	{
		"newline stripping, multiline",
		false,
		true,
		"    test\n string\n with\nnewlines",
		"    test string withnewlines",
	},
	{
		"all stripping, multiline with whitespace",
		true,
		true,
		"test \n     string\n with newlines\n and\n spaces",
		"teststringwith newlinesandspaces",
	},
}

func TestFormat_doFmtStringCleanup(t *testing.T) {
	for _, tt := range fmtStringCleanupTests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Format{
				FormatString:    tt.formatString,
				StripWhitespace: tt.stripWhitespace,
				StripNewlines:   tt.stripNewlines,
			}
			f.doFmtStringCleanup()
			if f.FormatString != tt.want {
				t.Errorf("Format.doFmtStringCleanup() = %q, want %q", f.FormatString, tt.want)
			}
		})
	}
}

func Benchmark_doFmtStringCleanup(b *testing.B) {
	for _, tt := range fmtStringCleanupTests {
		b.Run(tt.name, func(b *testing.B) {
			f := &Format{}
			for i := 0; i < b.N; i++ {
				f.FormatString = tt.formatString
				f.StripWhitespace = tt.stripWhitespace
				f.StripNewlines = tt.stripNewlines
				f.doFmtStringCleanup()
			}
		})
	}
}

var compileColourTests = []struct {
	name string
	f    *Format
	args []template.FuncMap
	want string
}{
	{
		"simple case",
		&Format{FormatString: "test string$c[red]"},
		nil,
		"test string\x034",
	},
	{
		"simple with funcmap",
		&Format{FormatString: "test $bstring"},
		[]template.FuncMap{
			{"test": func() string { return "test" }},
		},
		"test \x02string",
	},
}

func TestFormat_CompileColour(t *testing.T) {
	for _, tt := range compileColourTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test", true, tt.args...); err != nil {
				t.Errorf("Format.Compile(evalColour = true) error = %v", err)
			} else if tt.f.FormatString != tt.want {
				t.Errorf("Format.Compile(evalColour = true) formatString = %q, want %q", tt.f.FormatString, tt.want)
			}
		})
	}
}

func BenchmarkFormat_CompileColour(b *testing.B) {
	for _, tt := range compileColourTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.f.compiled = false
				_ = tt.f.Compile(tt.name, true, tt.args...)
			}
		})
	}
}


var formatCompileTests = []struct {
	name    string
	f       *Format
	args    []template.FuncMap
	wantErr bool
}{
	{
		"simple case",
		&Format{FormatString: "test string"},
		nil,
		false,
	},
	{
		"simple with funcmap",
		&Format{FormatString: "test string"},
		[]template.FuncMap{
			{"test": func() string { return "test" }},
		},
		false,
	},
	{
		"empty format",
		&Format{},
		nil,
		true,
	},
	{
		"bad template format",
		&Format{FormatString: "{{"},
		nil,
		true,
	},
	{
		"format with actual calls",
		&Format{FormatString: "{{zwsp .}}"},
		nil,
		false,
	},
}

func TestFormat_Compile(t *testing.T) {
	for _, tt := range formatCompileTests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test", false, tt.args...); (err != nil) != tt.wantErr {
				t.Errorf("Format.Compile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkFormat_Compile(b *testing.B) {
	for _, tt := range formatCompileTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.f.compiled = false
				_ = tt.f.Compile(tt.name, false, tt.args...)
			}
		})
	}
}

var testsForExecute = []struct {
	name    string
	f       *Format
	args    interface{}
	want    string
	wantErr bool
}{
	{
		"simple string",
		&Format{FormatString: "test string"},
		nil,
		"test string",
		false,
	},
	{
		"accessing values",
		&Format{FormatString: "{{.}}"},
		"test",
		"test",
		false,
	},
	{
		"bad value access",
		&Format{FormatString: "{{.something}}"},
		"test",
		"",
		true,
	},
	{
		"function call",
		&Format{FormatString: "{{zwsp \"test\"}}"},
		nil,
		"t\u200best",
		false,
	},
	{
		"bad function call",
		&Format{FormatString: "{{zwsp}}"},
		"",
		"",
		true,
	},
}

func cleanExecuteTests() {
	for _, tt := range testsForExecute {
		tt.f.compiled = false
		tt.f.CompiledFormat = nil
	}
}

func TestFormat_ExecuteBytes(t *testing.T) {
	cleanExecuteTests()
	for _, tt := range testsForExecute {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test_"+tt.name, false); err != nil {
				t.Error(err)
				return
			}
			got, err := tt.f.ExecuteBytes(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format.ExecuteBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			byteWant := []byte(tt.want)
			if tt.want == "" {
				byteWant = []byte(nil)
			}
			if !reflect.DeepEqual(got, byteWant) {
				t.Errorf("Format.ExecuteBytes() = %#v, want %#v", got, byteWant)
			}
		})
	}
}

func BenchmarkFormat_ExecuteBytes(b *testing.B) {
	cleanExecuteTests()
	for _, tt := range testsForExecute {
		b.Run(tt.name, func(b *testing.B) {
			b.StopTimer()
			if !tt.f.compiled {
				_ = tt.f.Compile("test_"+tt.name, false)
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				_, _ = tt.f.ExecuteBytes(tt.args)
			}
		})
	}
}

func TestFormat_Execute(t *testing.T) {
	cleanExecuteTests()
	for _, tt := range testsForExecute {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test_"+tt.name, false); err != nil {
				t.Error(err)
				return
			}
			got, err := tt.f.Execute(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Format.Execute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkFormat_Execute(b *testing.B) {
	cleanExecuteTests()
	for _, tt := range testsForExecute {
		b.Run(tt.name, func(b *testing.B) {
			b.StopTimer()
			if !tt.f.compiled {
				_ = tt.f.Compile("test_"+tt.name, false)
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				_, _ = tt.f.Execute(tt.args)
			}
		})
	}
}
