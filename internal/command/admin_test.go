package command

import "testing"

func TestAdmin_CheckMask(t *testing.T) {
	type fields struct {
		Level int
		Mask  string
	}

	type args struct {
		mask string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "basic test",
			fields: fields{1, "*!*@*"},
			args:   args{"test!test@test.test"},
			want:   true,
		},
		{
			name:   "basic test",
			fields: fields{1, "*!*@someHost"},
			args:   args{"test!test@test.test"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Admin{
				Level: tt.fields.Level,
				Mask:  tt.fields.Mask,
			}
			if got := a.MatchesMask(tt.args.mask); got != tt.want {
				t.Errorf("Admin.MatchesMask() = %v, want %v", got, tt.want)
			}
		})
	}
}
