package format

import (
	"reflect"
	"testing"
	"text/template"
)

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
	{
		"already compiled format",
		&Format{FormatString: "test", compiled: true},
		nil,
		true,
	},
}

func TestFormat_Compile(t *testing.T) {
	for _, tt := range formatCompileTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test", nil, tt.args...); (err != nil) != tt.wantErr {
				t.Errorf("Format.Compile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkFormat_Compile(b *testing.B) {
	for _, tt := range formatCompileTests {
		tt := tt
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tt.f.compiled = false
				_ = tt.f.Compile(tt.name, nil, tt.args...)
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
	{
		"chars should not be escaped",
		&Format{FormatString: "test of $b escaping"},
		"",
		"test of $b escaping",
		false,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test_"+tt.name, nil, nil); err != nil {
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
	cleanExecuteTests() // TODO: use testing.cleanup when its a thing

	for _, tt := range testsForExecute {
		tt := tt
		b.Run(tt.name, func(b *testing.B) {
			b.StopTimer()
			if !tt.f.compiled {
				_ = tt.f.Compile("test_"+tt.name, nil, nil)
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.f.Compile("test_"+tt.name, nil, nil); err != nil {
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
		tt := tt
		b.Run(tt.name, func(b *testing.B) {
			b.StopTimer()
			if !tt.f.compiled {
				_ = tt.f.Compile("test_"+tt.name, nil, nil)
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				_, _ = tt.f.Execute(tt.args)
			}
		})
	}
}
