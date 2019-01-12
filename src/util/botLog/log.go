package botLog

import (
    "fmt"
    "io"
    "log"
)
// TODO: Maybe switch this to direct output? I'd get a good chunk more control over how output happens etc. Would let me
//       control levels etc more specifically. Will want calldepth stuff though.
var internalCopy *log.Logger

func InitLogger(writer io.Writer, flags int) {
    internalCopy = log.New(writer, "", flags)
}

type Logger struct {
    prefix string
    logger *log.Logger
}

func NewLogger(prefix string, logger *log.Logger) *Logger {
    l := logger
    if l == nil {
        l = internalCopy
    }

    return &Logger{prefix: prefix, logger: l}
}

func (l *Logger) Prefix() string {
    return l.prefix
}

func (l *Logger) SetPrefix(prefix string) {
    l.prefix = prefix
}
func (l *Logger) internalLog(level string, depth int, args ...interface{}) {
    toPrint := fmt.Sprintf("[%s] [%s] %s", level, l.prefix, fmt.Sprint(args...))
    _ = l.logger.Output(depth, toPrint)
}

func (l *Logger) internalLogf(level, format string, args ...interface{}) {
    l.internalLog(level, 4, fmt.Sprintf(format, args...))
}

func (l *Logger) Info(args ...interface{}) {
    l.internalLog("INFO", 3, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
    l.internalLogf("INFO", format, args...)
}

func (l *Logger) Warn(args ...interface{}) {
    l.internalLog("WARN", 3, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
    l.internalLogf("WARN", format, args...)
}
