package core

import (
	"fmt"

	sh "dotbot-go/internal/shell"
)

// ShellHandler implements the shell directive.
type ShellHandler struct{}

// CanHandle reports whether directive is shell.
func (ShellHandler) CanHandle(directive string) bool { return directive == "shell" }

// SupportsDryRun reports that shell can preview commands without running them.
func (ShellHandler) SupportsDryRun() bool { return true }

// Validate checks shell directive data without running commands.
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

// Plan expands shell directive data into command operations.
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

// Handle runs commands requested by the shell directive.
func (ShellHandler) Handle(ctx *Context, directive string, data any) (bool, error) {
	items, ok := asList(data)
	if !ok {
		return false, fmt.Errorf("shell directive must be a list")
	}
	success := true
	defaults, _ := asMap(ctx.Defaults["shell"])
	for _, item := range items {
		cmd, msg, itemOK := shellCommandSpec(item)
		if !itemOK {
			success = false
			continue
		}
		opts := shellDirectiveOptionsFor(defaults, item)
		logShellCommand(ctx, opts, cmd, msg)
		if ctx.Options.DryRun {
			continue
		}
		ret := ctx.Shell.Run(ctx.RunContext, cmd, opts.runnerOptions(ctx))
		if ret != 0 {
			success = false
			ctx.Log.Warning(fmt.Sprintf("Command [%s] failed", cmd))
		}
	}
	return finish(ctx, success, "All commands have been executed", "Some commands were not successfully executed"), nil
}

type shellDirectiveOptions struct {
	stdin  bool
	stdout bool
	stderr bool
	quiet  bool
}

func shellDirectiveOptionsFor(defaults map[string]any, item any) shellDirectiveOptions {
	opts := shellDirectiveOptions{}.withMap(defaults)
	if values, ok := asMap(item); ok {
		opts = opts.withMap(values)
	}
	return opts
}

func (opts shellDirectiveOptions) withMap(values map[string]any) shellDirectiveOptions {
	if values == nil {
		return opts
	}
	return shellDirectiveOptions{
		stdin:  boolValue(values, "stdin", opts.stdin),
		stdout: boolValue(values, "stdout", opts.stdout),
		stderr: boolValue(values, "stderr", opts.stderr),
		quiet:  boolValue(values, "quiet", opts.quiet),
	}
}

func (opts shellDirectiveOptions) runnerOptions(ctx *Context) sh.Options {
	stdout := opts.stdout
	stderr := opts.stderr
	if ctx.Options.Verbose > 1 {
		stdout = true
		stderr = true
	}
	return sh.Options{
		CWD:          ctx.BaseDirectory,
		EnableStdin:  opts.stdin,
		EnableStdout: stdout,
		EnableStderr: stderr,
		Timeout:      defaultTimeout(ctx.Options.ShellTimeout),
	}
}

func logShellCommand(ctx *Context, opts shellDirectiveOptions, command, description string) {
	prefix := ""
	if ctx.Options.DryRun {
		prefix = "Would run command "
	}
	if opts.quiet {
		if description != "" {
			ctx.Log.Info(prefix + description)
		}
		return
	}
	if description == "" {
		ctx.Log.Action(prefix + command)
		return
	}
	ctx.Log.Action(fmt.Sprintf("%s%s [%s]", prefix, description, command))
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
