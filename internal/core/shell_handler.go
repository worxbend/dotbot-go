package core

import (
	"fmt"

	sh "dotbot-go/internal/shell"
)

type ShellHandler struct{}

func (ShellHandler) CanHandle(directive string) bool { return directive == "shell" }
func (ShellHandler) SupportsDryRun() bool            { return true }

func (ShellHandler) Validate(ctx *Context, directive string, data any) error {
	items, ok := asList(data)
	if !ok {
		return fmt.Errorf("shell directive must be a list")
	}
	for _, item := range items {
		if _, _, ok := shellCommandSpec(item); !ok {
			return fmt.Errorf("shell directive item must include a command")
		}
	}
	return nil
}

func (h ShellHandler) Plan(ctx *Context, directive string, data any) ([]Operation, error) {
	if err := h.Validate(ctx, directive, data); err != nil {
		return nil, err
	}
	items, _ := asList(data)
	operations := []Operation{}
	for _, item := range items {
		command, description, _ := shellCommandSpec(item)
		operations = append(operations, Operation{
			Directive: directive,
			Target:    command,
			Detail:    description,
		})
	}
	return operations, nil
}

func (ShellHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	items, ok := asList(data)
	if !ok {
		return false, fmt.Errorf("shell directive must be a list")
	}
	success := true
	defaults, _ := asMap(ctx.Defaults["shell"])
	for _, item := range items {
		stdin := false
		stdout := false
		stderr := false
		quiet := false
		if defaults != nil {
			stdin = boolValue(defaults, "stdin", stdin)
			stdout = boolValue(defaults, "stdout", stdout)
			stderr = boolValue(defaults, "stderr", stderr)
			quiet = boolValue(defaults, "quiet", quiet)
		}
		cmd, msg, itemOK := shellCommandSpec(item)
		if !itemOK {
			success = false
			continue
		}
		if m, ok := asMap(item); ok {
			stdin = boolValue(m, "stdin", stdin)
			stdout = boolValue(m, "stdout", stdout)
			stderr = boolValue(m, "stderr", stderr)
			quiet = boolValue(m, "quiet", quiet)
		}
		prefix := ""
		if ctx.Options.DryRun {
			prefix = "Would run command "
		}
		if quiet {
			if msg != "" {
				ctx.Log.Info(prefix + msg)
			}
		} else if msg == "" {
			ctx.Log.Action(prefix + cmd)
		} else {
			ctx.Log.Action(fmt.Sprintf("%s%s [%s]", prefix, msg, cmd))
		}
		if ctx.Options.DryRun {
			continue
		}
		if ctx.Options.Verbose > 1 {
			stdout = true
			stderr = true
		}
		ret := ctx.Shell.Run(ctx.RunContext, cmd, sh.Options{
			CWD:          ctx.BaseDirectory,
			EnableStdin:  stdin,
			EnableStdout: stdout,
			EnableStderr: stderr,
			Timeout:      defaultTimeout(ctx.Options.ShellTimeout),
		})
		if ret != 0 {
			success = false
			ctx.Log.Warning(fmt.Sprintf("Command [%s] failed", cmd))
		}
	}
	return finish(ctx, success, "All commands have been executed", "Some commands were not successfully executed"), nil
}

func shellCommandSpec(item any) (string, string, bool) {
	if s, ok := asString(item); ok {
		return s, "", true
	}
	if m, ok := asMap(item); ok {
		cmd, ok := asString(m["command"])
		if !ok {
			return "", "", false
		}
		msg, _ := asString(m["description"])
		return cmd, msg, true
	}
	if list, ok := asList(item); ok && len(list) > 0 {
		cmd, ok := asString(list[0])
		if !ok {
			return "", "", false
		}
		msg := ""
		if len(list) > 1 {
			msg, _ = asString(list[1])
		}
		return cmd, msg, true
	}
	return "", "", false
}
