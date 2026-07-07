package log

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Level controls the minimum message severity written by Logger.
type Level int

const (
	// Debug includes low-level diagnostic messages.
	Debug Level = iota
	// Info includes informational messages that are hidden by default.
	Info
	// Action includes user-visible steps and is the default level.
	Action
	// Warning includes recoverable problems.
	Warning
	// Error includes failed operations.
	Error
)

// Logger writes labeled progress messages to an output stream.
type Logger struct {
	out   io.Writer
	level Level
	color bool
}

// New creates a Logger that writes action-and-above messages to out.
func New(out io.Writer) *Logger {
	return &Logger{out: out, level: Action, color: supportsColor(out)}
}

// SetLevel changes the minimum level written by the logger.
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// UseColor enables or disables ANSI color output.
func (l *Logger) UseColor(color bool) {
	l.color = color
}

// Debug writes a diagnostic message.
func (l *Logger) Debug(msg string) { l.log(Debug, msg) }

// Info writes an informational message.
func (l *Logger) Info(msg string) { l.log(Info, msg) }

// Action writes a user-visible action message.
func (l *Logger) Action(msg string) { l.log(Action, msg) }

// Warning writes a recoverable problem message.
func (l *Logger) Warning(msg string) { l.log(Warning, msg) }

// Error writes a failed operation message.
func (l *Logger) Error(msg string) { l.log(Error, msg) }

func (l *Logger) log(level Level, msg string) {
	if level < l.level {
		return
	}
	_, _ = fmt.Fprintf(
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
		return "step "
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
