package log

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Level int

const (
	Debug Level = iota
	Info
	Action
	Warning
	Error
)

type Logger struct {
	out   io.Writer
	level Level
	color bool
}

func New(out io.Writer) *Logger {
	return &Logger{out: out, level: Action, color: supportsColor(out)}
}

func (l *Logger) SetLevel(level Level) {
	l.level = level
}

func (l *Logger) UseColor(color bool) {
	l.color = color
}

func (l *Logger) Debug(msg string)   { l.log(Debug, msg) }
func (l *Logger) Info(msg string)    { l.log(Info, msg) }
func (l *Logger) Action(msg string)  { l.log(Action, msg) }
func (l *Logger) Warning(msg string) { l.log(Warning, msg) }
func (l *Logger) Error(msg string)   { l.log(Error, msg) }

func (l *Logger) log(level Level, msg string) {
	if level < l.level {
		return
	}
	fmt.Fprintf(
		l.out,
		"%s%s%s %s%s%s\n",
		l.colorFor(level),
		labelFor(level),
		l.reset(),
		l.colorForMessage(level),
		msg,
		l.reset(),
	)
}

func (l *Logger) colorFor(level Level) string {
	if !l.color {
		return ""
	}
	switch level {
	case Debug:
		return "\033[2;33m"
	case Info:
		return "\033[34m"
	case Action:
		return "\033[32m"
	case Warning:
		return "\033[33m"
	default:
		return "\033[1;31m"
	}
}

func (l *Logger) colorForMessage(level Level) string {
	if !l.color || level != Error {
		return ""
	}
	return "\033[31m"
}

func (l *Logger) reset() string {
	if !l.color {
		return ""
	}
	return "\033[0m"
}

func labelFor(level Level) string {
	switch level {
	case Debug:
		return "debug"
	case Info:
		return "info "
	case Action:
		return "done "
	case Warning:
		return "warn "
	default:
		return "error"
	}
}

func supportsColor(out io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
