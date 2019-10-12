package transformer

import (
	"image/color"
	"reflect"
	"testing"
)

func TestNewSimpleTransformer(t *testing.T) {
	type args struct {
		replaceMap map[rune]string
		colourMap  map[color.Color]string
	}
	tests := []struct {
		name string
		args args
		want *SimpleTransformer
	}{
		{
			name: "normal setup",
			args: args{
				replaceMap: map[rune]string{Bold: "BOLD", Italic: "ITALIC", Underline: "UNDERLINE", Strikethrough: "STRIKETHROUGH", Reset: "RESET"},
				colourMap:  map[color.Color]string{color.Gray{Y: 42}: "GREY", color.Black: "BLACK", color.White: "WHITE"},
			},
			want: &SimpleTransformer{
				rplMap:  map[rune]string{Bold: "BOLD", Italic: "ITALIC", Underline: "UNDERLINE", Strikethrough: "STRIKETHROUGH", Reset: "RESET"},
				palette: []color.Color{color.Gray{Y: 42}, color.Black, color.White},
				colMap:  map[color.Color]string{color.Gray{Y: 42}: "GREY", color.Black: "BLACK", color.White: "WHITE"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := NewSimpleTransformer(tt.args.replaceMap, tt.args.colourMap)
			switch {
			case !reflect.DeepEqual(tt.want.rplMap, tf.rplMap):
				t.Errorf("NewSimpleTransformer().rplMap = %v, want %v", tf.rplMap, tt.want.rplMap)
			case !reflect.DeepEqual(tt.want.colMap, tf.colMap):
				t.Errorf("NewSimpleTransformer.colMap = %v, want %v", tf.colMap, tt.want.colMap)
			case !cmpSliceNoOrder(tf.palette, tt.want.palette):
				t.Errorf("NewSimpleTransformer.palette = %#v, want %#v", tf.palette, tt.want.palette)
			}
		})
	}
}

func TestSimpleTransformer_MakeIntermediate(t *testing.T) {
	var constructorArgs = struct {
		replaceMap map[rune]string
		colourMap  map[color.Color]string
	}{
		replaceMap: map[rune]string{Bold: "bold", Italic: "italic", Underline: "underline", Strikethrough: "strikethrough", Reset: "reset"},
		colourMap: map[color.Color]string{
			color.RGBA{R: 0xF9, G: 0x4B, B: 0xA3, A: 0xFF}: "ONE",
			color.RGBA{R: 0x91, G: 0x82, B: 0xBA, A: 0xFF}: "TWO",
			color.RGBA{R: 0x7a, G: 0x88, B: 0xc9, A: 0xFF}: "THREE",
		},
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple",
			in:   "this boldis a test message with italic reset",
			want: "this $bis a test message with $i $r",
		},
		{
			name: "colour spam",
			in:   "thisONE has some colourTWOs in THREEit",
			want: "this$cF94BA3 has some colour$c9182BAs in $c7A88C9it",
		},
		{
			name: "mixed",
			in:   "thisONE has some bold and some THREE colours in reset it",
			want: "this$cF94BA3 has some $b and some $c7A88C9 colours in $r it",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSimpleTransformer(constructorArgs.replaceMap, constructorArgs.colourMap)
			if got := s.MakeIntermediate(tt.in); got != tt.want {
				t.Errorf("MakeIntermediate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSimpleTransformer_Transform(t *testing.T) {
	var constructorArgs = struct {
		replaceMap map[rune]string
		colourMap  map[color.Color]string
	}{
		replaceMap: map[rune]string{Bold: "bold", Italic: "italic", Underline: "underline", Strikethrough: "strikethrough", Reset: "reset"},
		colourMap: map[color.Color]string{
			color.RGBA{R: 0xF9, G: 0x4B, B: 0xA3, A: 0xFF}: "ONE",
			color.RGBA{R: 0x91, G: 0x82, B: 0xBA, A: 0xFF}: "TWO",
			color.RGBA{R: 0x7a, G: 0x88, B: 0xc9, A: 0xFF}: "THREE",
		},
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple",
			in:   "this $bis a test message with $i $r",
			want: "this boldis a test message with italic reset",
		},
		{
			name: "colour spam",
			in:   "this$cF94BA3 has some colour$c9182BAs in $c7A88C9it",
			want: "thisONE has some colourTWOs in THREEit",
		},
		{
			name: "mixed",
			in:   "this$cF94BA3 has some $b and some $c7A88C9 colours in $r it",
			want: "thisONE has some bold and some THREE colours in reset it",
		},
		{
			name: "colour coercion",
			in:   "this $cF75BA3 has some weird $c424242 colours that are not dead on in it",
			want: "this ONE has some weird TWO colours that are not dead on in it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSimpleTransformer(constructorArgs.replaceMap, constructorArgs.colourMap)
			if got := s.Transform(tt.in); got != tt.want {
				t.Errorf("Transform() = %v, want %v", got, tt.want)
			}
		})
	}
}
