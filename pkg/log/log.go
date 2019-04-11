package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	FTimestamp = 1 << iota
	FShowFile  // Maybe some other time.
)

const (
	TRACE = 10 * iota
	DEBUG
	INFO
	WARN
	ERROR
	CRIT
	PANIC
)

func levelToString(level int) string {
	switch level {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO "
	case WARN:
		return "WARN "
	case ERROR:
		return "ERROR"
	case CRIT:
		return "CRIT "
	case PANIC:
		return "PANIC"
	}
	return "?????"
}

// Logger is a level based logging engine
type Logger struct {
	flags    int
	output   io.Writer
	prefix   string
	wMutex   sync.Mutex
	minLevel int
}

func (l *Logger) Flags() int {
	return l.flags
}

func (l *Logger) SetFlags(flags int) *Logger {
	l.flags = flags
	return l
}

func (l *Logger) Prefix() string {
	return l.prefix
}

func (l *Logger) SetPrefix(prefix string) *Logger {
	l.prefix = prefix
	return l
}

func (l *Logger) MinLevel() int {
	return l.minLevel
}

func New(flags int, output io.Writer, prefix string, minLevel int) *Logger {
	return &Logger{flags: flags, output: output, prefix: prefix, minLevel: minLevel, wMutex: sync.Mutex{}}
}

func shortenFilename(filename string) string {
	out := filename
	for i := len(filename) - 1; i > 0; i-- {
		if filename[i] == '/' {
			out = filename[i+1:]
			break
		}
	}
	return out
}

const openBrace = '['
const closeBrace = ']'
const space = ' '

func (l *Logger) writeOut(msg string, level int) {
	if level < l.minLevel {
		return
	}

	outStr := strings.Builder{}
	if l.flags&FTimestamp != 0 {
		outStr.WriteRune(openBrace)
		outStr.WriteString(time.Now().Format("15:04:05.000"))
		outStr.WriteRune(closeBrace)
		outStr.WriteRune(space)
	}

	outStr.WriteRune(openBrace)
	outStr.WriteString(levelToString(level))
	outStr.WriteRune(closeBrace)
	outStr.WriteRune(space)

	if l.flags&FShowFile != 0 {
		outStr.WriteRune(openBrace)
		_, file, line, ok := runtime.Caller(2)
		if !ok {
			outStr.WriteString("???")
		} else {
			outStr.WriteString(shortenFilename(file))
			outStr.WriteRune(':')
			outStr.WriteString(strconv.Itoa(line))
		}

		outStr.WriteRune(closeBrace)
		outStr.WriteRune(space)
	}

	if l.prefix != "" {
		outStr.WriteRune(openBrace)
		outStr.WriteString(l.prefix)
		outStr.WriteRune(closeBrace)
		outStr.WriteRune(space)
	}

	outStr.WriteString(strings.TrimRight(msg, "\r\n"))
	outStr.WriteRune('\n')

	l.wMutex.Lock()
	defer l.wMutex.Unlock()
	_, _ = l.output.Write([]byte(outStr.String()))
}

func (l *Logger) Trace(args ...interface{}) {
	l.writeOut(fmt.Sprint(args...), TRACE)
}

func (l *Logger) Tracef(format string, args ...interface{}) {
	l.writeOut(fmt.Sprintf(format, args...), TRACE)
}

func (l *Logger) Debug(args ...interface{}) {
	l.writeOut(fmt.Sprint(args...), DEBUG)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.writeOut(fmt.Sprintf(format, args...), DEBUG)
}

func (l *Logger) Info(args ...interface{}) {
	l.writeOut(fmt.Sprint(args...), INFO)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.writeOut(fmt.Sprintf(format, args...), INFO)
}

func (l *Logger) Warn(args ...interface{}) {
	l.writeOut(fmt.Sprint(args...), WARN)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.writeOut(fmt.Sprintf(format, args...), WARN)
}

func (l *Logger) Crit(args ...interface{}) {
	l.writeOut(fmt.Sprint(args...), CRIT)
	os.Exit(1)
}

func (l *Logger) Critf(format string, args ...interface{}) {
	l.writeOut(fmt.Sprintf(format, args...), CRIT)
	os.Exit(1)
}

func (l *Logger) Panic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	l.writeOut(msg, PANIC)
	panic(msg)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.writeOut(msg, PANIC)
	panic(msg)
}

func (l Logger) Clone() *Logger {
	return &l
}
