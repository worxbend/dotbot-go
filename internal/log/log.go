package log

import (
	"fmt"
	"io"
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
	return &Logger{out: out, level: Action, color: false}
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
	fmt.Fprintf(l.out, "%s%s%s\n", l.colorFor(level), msg, l.reset())
}

func (l *Logger) colorFor(level Level) string {
	if !l.color || level < Debug {
		return ""
	}
	switch {
	case level < Info:
		return "\033[33m"
	case level < Action:
		return "\033[34m"
	case level < Warning:
		return "\033[32m"
	case level < Error:
		return "\033[35m"
	default:
		return "\033[31m"
	}
}

func (l *Logger) reset() string {
	if !l.color {
		return ""
	}
	return "\033[0m"
}
