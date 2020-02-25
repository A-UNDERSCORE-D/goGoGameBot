package format

import (
	"testing"
)

// duplicate the map so individual tests dont mess with things
func getTestMap() map[string]interface{} {
	m := map[string]interface{}{
		"int":            1,
		"negativeInt":    -2,
		"answer to life": 42,
		"string":         "test",
		"emptyString":    "",
		"true":           true,
		"false":          false,
	}
	nm := make(map[string]interface{})

	for k, v := range m {
		nm[k] = v
	}

	return nm
}

func makeTestStorage() *Storage {
	return &Storage{
		data: getTestMap(),
	}
}

func TestStorage_Delete(t *testing.T) {
	storage := makeTestStorage()
	tests := []struct {
		testName   string
		targetName string
	}{
		{"int", "int"},
		{"bool1", "true"},
		{"bool2", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			storage.Delete(tt.targetName)
			if _, ok := storage.get(tt.targetName); ok {
				t.Errorf("Delete(): expected %q not to exist in %#v", tt.targetName, storage)
			}
		})
	}
}

func TestStorage_GetBool(t *testing.T) {
	s := makeTestStorage()
	tests := []struct {
		testName   string
		targetName string
		default_   bool
		want       bool
	}{
		{
			"get true",
			"true",
			false,
			true,
		},
		{
			"get false",
			"false",
			true,
			false,
		},
		{
			"get nonexistent false default",
			"thisDoesntExist",
			false,
			false,
		},
		{
			"get nonexistent true default",
			"thisDoesntExist",
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := s.GetBool(tt.targetName, tt.default_); got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorage_GetInt(t *testing.T) {
	s := makeTestStorage()
	tests := []struct {
		testName   string
		targetName string
		default_   int
		want       int
	}{
		{
			"get positive",
			"int",
			1337,
			1,
		},
		{
			"get negative",
			"negativeInt",
			1337,
			-2,
		},
		{
			"life",
			"answer to life",
			-1337,
			42,
		},
		{
			"default",
			"noexist",
			1337,
			1337,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := s.GetInt(tt.targetName, tt.default_); got != tt.want {
				t.Errorf("GetInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorage_GetString(t *testing.T) {
	s := makeTestStorage()
	tests := []struct {
		testName   string
		targetName string
		default_   string
		want       string
	}{
		{
			"get",
			"string",
			"asd",
			"test",
		},
		{
			"get empty string",
			"emptyString",
			"this isnt empty",
			"",
		},
		{
			"get not exist",
			"not exist",
			"this doesnt exist",
			"this doesnt exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			if got := s.GetString(tt.targetName, tt.default_); got != tt.want {
				t.Errorf("GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStorage_SetBool(t *testing.T) {
	s := new(Storage)
	tests := []struct {
		testName   string
		targetName string
		setting    bool
	}{
		{
			"true",
			"true",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			s.SetBool(tt.targetName, tt.setting)

			if res, exists := s.get(tt.targetName); exists {
				if b, ok := res.(bool); !ok {
					t.Errorf("s.SetBool() set a type that was not a bool: %t", res)
				} else if ok && (b != tt.setting) {
					t.Errorf("s.setbool set an incorrect value. got %v, want %v", b, tt.setting)
				}
			}
		})
	}
}

func TestStorage_SetInt(t *testing.T) {
	s := new(Storage)
	tests := []struct {
		testName   string
		targetName string
		setting    int
	}{
		{
			"true",
			"true",
			1337,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			s.SetInt(tt.targetName, tt.setting)

			if res, exists := s.get(tt.targetName); exists {
				if b, ok := res.(int); !ok {
					t.Errorf("s.SetInt() set an invalid type: got %T, want Int", res)
				} else if ok && (b != tt.setting) {
					t.Errorf("s.SetInt() set an incorrect value. got %v, want %v", b, tt.setting)
				}
			}
		})
	}
}

func TestStorage_SetString(t *testing.T) {
	s := new(Storage)
	tests := []struct {
		testName   string
		targetName string
		setting    string
	}{
		{
			"true",
			"true",
			"so I heard ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			s.SetString(tt.targetName, tt.setting)

			if res, exists := s.get(tt.targetName); exists {
				if b, ok := res.(string); !ok {
					t.Errorf("s.SetString() set an invalid type: got %T, want Int", res)
				} else if ok && (b != tt.setting) {
					t.Errorf("s.SetString() set an incorrect value. got %v, want %v", b, tt.setting)
				}
			}
		})
	}
}
