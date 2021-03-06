// Package log provides a level based logger in place of go's standard logger
package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Log flags
const (
	FTimestamp = 1 << iota
	FShowFile  // Maybe some other time.
)

// Log levels
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
	parent   *Logger
}

// Flags is a getter for the flags the logger currently has
func (l *Logger) Flags() int {
	return l.flags
}

// SetFlags is a setter for the logger's flags. It returns the same Logger it was run on for use in call chains
func (l *Logger) SetFlags(flags int) *Logger {
	l.flags = flags
	return l
}

// Prefix is a getter for the prefix the logger currently has
func (l *Logger) Prefix() string {
	return l.prefix
}

// SetPrefix is a setter for the logger's flags. It returns the same Logger it was run on for use in call chains
func (l *Logger) SetPrefix(prefix string) *Logger {
	l.prefix = prefix
	return l
}

// MinLevel is a getter for the minimum log level on the logger
func (l *Logger) MinLevel() int {
	return l.minLevel
}

// New creates a new logger with the set options
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

func (l *Logger) writeMsg(msg string, level int) {
	if level < l.minLevel {
		return
	}

	outStr := bytes.Buffer{}

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

	if _, err := l.write(outStr.Bytes()); err != nil {
		fmt.Fprintf(os.Stderr, "Could not write log entry: %s", err)
	}
}

// Trace logs the passed data at the Trace level. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Trace(args ...interface{}) {
	l.writeMsg(fmt.Sprint(args...), TRACE)
}

// Tracef logs the passed data at the Tracef level using the format string passed as the first argument to
// format the message. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Tracef(format string, args ...interface{}) {
	l.writeMsg(fmt.Sprintf(format, args...), TRACE)
}

// Debug logs the passed data at the Debug level. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Debug(args ...interface{}) {
	l.writeMsg(fmt.Sprint(args...), DEBUG)
}

// Debugf logs the passed data at the Debugf level using the format string passed as the first argument to
// format the message. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.writeMsg(fmt.Sprintf(format, args...), DEBUG)
}

// Info logs the passed data at the Info level. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Info(args ...interface{}) {
	l.writeMsg(fmt.Sprint(args...), INFO)
}

// Infof logs the passed data at the Infof level using the format string passed as the first argument to
// format the message. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Infof(format string, args ...interface{}) {
	l.writeMsg(fmt.Sprintf(format, args...), INFO)
}

// Warn logs the passed data at the Warn level. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Warn(args ...interface{}) {
	l.writeMsg(fmt.Sprint(args...), WARN)
}

// Warnf logs the passed data at the Warnf level using the format string passed as the first argument
// to format the message. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.writeMsg(fmt.Sprintf(format, args...), WARN)
}

// Crit logs the passed data at the Crit level. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Crit(args ...interface{}) {
	l.writeMsg(fmt.Sprint(args...), CRIT)
	os.Exit(1)
}

// Critf logs the passed data at the Critf level using the format string passed as the first argument to
// format the message. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Critf(format string, args ...interface{}) {
	l.writeMsg(fmt.Sprintf(format, args...), CRIT)
	os.Exit(1)
}

// Panic logs the passed data at the Panic level. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Panic(args ...interface{}) {
	msg := fmt.Sprint(args...)
	l.writeMsg(msg, PANIC)
	panic(msg)
}

// Panicf logs the passed data at the Panicf level using the format string passed as the first argument to
// format the message. The passed arguments are run through fmt.Sprintf before logging
func (l *Logger) Panicf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.writeMsg(msg, PANIC)
	panic(msg)
}

func (l *Logger) write(msg []byte) (int, error) {
	if l.parent != nil {
		return l.parent.write(msg)
	}

	l.wMutex.Lock()
	defer l.wMutex.Unlock()

	return l.output.Write(msg)
}

// Clone duplicates the logger it is run on, returning a clean version that can be modified
func (l *Logger) Clone() *Logger {
	return &Logger{
		flags:    l.flags,
		output:   l.output,
		prefix:   l.prefix,
		minLevel: l.minLevel,
		parent:   l,
	}
}
