package util

import (
	"reflect"
	"testing"
)

func Test_zip(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want []string
	}{
		{
			"empty map",
			map[string]string{},
			[]string(nil),
		},
		{
			"2 long map",
			map[string]string{"test": "case"},
			[]string{"test", "case"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zip(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("zip() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Benchmark_zip(b *testing.B) {
	ttZip := map[string]string{"test": "test2", "test3": "test4"}
	for i := 0; i < b.N; i++ {
		zip(ttZip)
	}
}
