package log

import (
	"io/ioutil"
	"log"
	"sync"
	"testing"
)

func BenchmarkLogger_Info(b *testing.B) {
	l := Logger{
		flags:    FTimestamp,
		output:   ioutil.Discard,
		prefix:   "test",
		wMutex:   sync.Mutex{},
		minLevel: 0,
	}
	for i := 0; i < b.N; i++ {
		l.Info("test")
	}
}

func BenchmarkStdlogger(b *testing.B) {
	l := log.New(ioutil.Discard, "test", log.Ltime)
	for i := 0; i < b.N; i++ {
		l.Print("test")
	}
}

/*
func Test_levelToString(t *testing.T) {
	type args struct {
		level int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"testInfo",
			args{INFO},
			"INFO ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := levelToString(tt.args.level); got != tt.want {
				t.Errorf("levelToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_writeOut(t *testing.T) {
	type args struct {
		msg   string
		level int
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		{

		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.writeOut(tt.args.msg, tt.args.level)
		})
	}
}

func TestLogger_Trace(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Trace(tt.args.args...)
		})
	}
}

func TestLogger_Tracef(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Tracef(tt.args.format, tt.args.args...)
		})
	}
}

func TestLogger_Debug(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Debug(tt.args.format, tt.args.args...)
		})
	}
}

func TestLogger_Debugf(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Debugf(tt.args.format, tt.args.args...)
		})
	}
}

func TestLogger_Info(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Info(tt.args.args...)
		})
	}
}

func TestLogger_Infof(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Infof(tt.args.format, tt.args.args...)
		})
	}
}

func TestLogger_Warn(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Warn(tt.args.args...)
		})
	}
}

func TestLogger_Warnf(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Warnf(tt.args.format, tt.args.args...)
		})
	}
}

func TestLogger_Crit(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Crit(tt.args.args...)
		})
	}
}

func TestLogger_Critf(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Critf(tt.args.format, tt.args.args...)
		})
	}
}

func TestLogger_Panic(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Panic(tt.args.args...)
		})
	}
}

func TestLogger_Panicf(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}
	tests := []struct {
		name string
		l    *Logger
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Panicf(tt.args.format, tt.args.args...)
		})
	}
}
*/
