package irc

import (
	"testing"
)

var intermedTests = []struct {
	name string
	in   string
	want string
}{
	{
		name: "bolded chars",
		in:   "this\x02 is a test with bolds\x02",
		want: "this$b is a test with bolds$b",
	},
	{
		name: "rainbow",
		in:   "\x0304h\x0307e\x0308r\x0309e \x0310h\x0312a\x0302v\x0306e \x0305a \x0307t\x0308e\x0309s\x0303t \x0312s\x0302t\x0306r\x0313i\x0305n\x0304g",
		want: "$cFF0000h$cFC7F00e$cFFFF00r$c00FC00e $c009393h$c0000FCa$c00007Fv$c9C009Ce $c7F0000a $cFC7F00t$cFFFF00e$c00FC00s$c009300t $c0000FCs$c00007Ft$c9C009Cr$cFF00FFi$c7F0000n$cFF0000g",
	},
	{
		name: "fg and bg",
		in:   "this \x0301,00 is \x0302,01 a test",
		want: "this $c000000 is $c00007F a test",
	},
	{
		name: "sneaky extra numbers",
		in:   "this\x030021 is a test",
		want: "this$cFFFFFF21 is a test",
	},
	{
		name: "trailing comma",
		in:   "this \x0300, is a test",
		want: "this $cFFFFFF, is a test",
	},
	{
		name: "bare colour code",
		in:   "test \x03 string",
		want: "test  string",
	},
	{
		name: "test with sentinels",
		in:   "this i$ a te$t",
		want: "this i$$ a te$$t",
	},
}

func TestIRCTransformer_MakeIntermediate(t *testing.T) {
	for _, tt := range intermedTests {
		t.Run(tt.name, func(t *testing.T) {
			ir := Transformer{}
			if got := ir.MakeIntermediate(tt.in); got != tt.want {
				t.Errorf("MakeIntermediate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIRCTransformer_Transform(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "encode black with bold",
			in:   "$c000000this is a test message with $bbold characters",
			want: "\x0301this is a test message with \x02bold characters",
		},
		{
			name: "all chars, no colour",
			in:   "$b$i$u$s$r",
			want: "\x02\x1d\x1F\x0F", // Note the skipped strikethrough
		},
		{
			name: "colour correction",
			in:   "this is a test of $cF0F0F0colour$r stuff",
			want: "this is a test of \x0300colour\x0F stuff",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := Transformer{}
			if got := ir.Transform(tt.in); got != tt.want {
				t.Errorf("Transform() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChainTransformer(t *testing.T) {
	for _, tt := range intermedTests {
		t.Run(tt.name, func(t *testing.T) {
			chained := tt.in
			ir := Transformer{}
			for i := 0; i < 20; i++ {
				chained = ir.MakeIntermediate(ir.Transform(chained))
			}
			if chained != tt.want {
				t.Errorf("Chained tranformer = %q, want %q", chained, tt.want)
			}
		})
	}
}
