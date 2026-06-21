package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"dotbot-go/internal/app"
)

type options struct {
	superQuiet    bool
	quiet         bool
	verbose       int
	baseDirectory string
	configFiles   []string
	only          []string
	skip          []string
	dryRun        bool
	forceColor    bool
	noColor       bool
	exitOnFailure bool
	showVersion   bool
	validate      bool
	plan          bool
	output        string
}

func Execute(args []string, stdout, stderr io.Writer) int {
	return ExecuteContext(context.Background(), args, stdout, stderr)
}

func ExecuteContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	opts := &options{}
	cmd := newRootCommand(ctx, opts, stdout)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		if errors.Is(err, app.ErrExit) {
			return 1
		}
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func newRootCommand(ctx context.Context, opts *options, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "dotbot-go",
		Short:         "A Go port of Dotbot for bootstrapping dotfiles",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(ctx, opts.appOptions(), stdout, app.Dependencies{})
		},
	}
	registerCommonFlags(cmd, opts)
	registerApplyFlags(cmd, opts)
	cmd.AddCommand(newValidateCommand(ctx, opts, stdout))
	cmd.AddCommand(newPlanCommand(ctx, opts, stdout))
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStdout(), renderHelp(cmd, stdout))
	})
	return cmd
}

func newValidateCommand(ctx context.Context, opts *options, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "validate",
		Short:         "Validate configuration without applying changes",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.validate = true
			return app.Run(ctx, opts.appOptions(), stdout, app.Dependencies{})
		},
	}
	registerCommonFlags(cmd, opts)
	return cmd
}

func newPlanCommand(ctx context.Context, opts *options, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plan",
		Short:         "Print planned operations without applying changes",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.plan = true
			return app.Run(ctx, opts.appOptions(), stdout, app.Dependencies{})
		},
	}
	registerCommonFlags(cmd, opts)
	cmd.Flags().StringVar(&opts.output, "output", "text", "output format: text or json")
	return cmd
}

func registerCommonFlags(cmd *cobra.Command, opts *options) {
	cmd.Flags().BoolVarP(&opts.superQuiet, "super-quiet", "Q", false, "deprecated quiet mode")
	mustMarkHidden(cmd, "super-quiet")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "suppress most output")
	cmd.Flags().CountVarP(&opts.verbose, "verbose", "v", "enable verbose output\n-v: show informational messages\n-vv: also, set shell commands stderr/stdout to true")
	cmd.Flags().StringVarP(&opts.baseDirectory, "base-directory", "d", "", "execute commands from within BASE_DIR")
	cmd.Flags().StringArrayVarP(&opts.configFiles, "config-file", "c", nil, "run commands given in CONFIG_FILE")
	cmd.Flags().StringSliceVar(&opts.only, "only", nil, "only run specified directives")
	cmd.Flags().StringSliceVar(&opts.skip, "except", nil, "skip specified directives")
	cmd.Flags().BoolVar(&opts.forceColor, "force-color", false, "force color output")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "disable color output")
	cmd.Flags().BoolVar(&opts.showVersion, "version", false, "show program's version number and exit")
}

func registerApplyFlags(cmd *cobra.Command, opts *options) {
	cmd.Flags().BoolVarP(&opts.dryRun, "dry-run", "n", false, "print what would be done, without doing it")
	cmd.Flags().BoolVarP(&opts.exitOnFailure, "exit-on-failure", "x", false, "exit after first failed directive")
}

func mustMarkHidden(cmd *cobra.Command, name string) {
	if err := cmd.Flags().MarkHidden(name); err != nil {
		panic(fmt.Sprintf("hide flag %s: %v", name, err))
	}
}

func ExecuteOS() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return ExecuteContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
}

func (o *options) appOptions() app.Options {
	return app.Options{
		SuperQuiet:    o.superQuiet,
		Quiet:         o.quiet,
		Verbose:       o.verbose,
		BaseDirectory: o.baseDirectory,
		ConfigFiles:   o.configFiles,
		Only:          o.only,
		Skip:          o.skip,
		DryRun:        o.dryRun,
		ForceColor:    o.forceColor,
		NoColor:       o.noColor,
		ExitOnFailure: o.exitOnFailure,
		ShowVersion:   o.showVersion,
		Validate:      o.validate,
		Plan:          o.plan,
		Output:        o.output,
	}
}

func renderHelp(cmd *cobra.Command, stdout io.Writer) string {
	color := helpColorEnabled(cmd, stdout)
	var b strings.Builder

	helpTitle(&b, color, cmd)

	helpSection(&b, color, "Usage")
	fmt.Fprintf(&b, "  %s\n\n", cmd.UseLine())

	helpSection(&b, color, "Examples")
	if len(cmd.Commands()) > 0 {
		fmt.Fprintln(&b, "  dotbot-go -c install.conf.yaml")
		fmt.Fprintln(&b, "  dotbot-go validate -c install.conf.yaml")
		fmt.Fprintln(&b, "  dotbot-go plan -c install.conf.yaml --output json")
		fmt.Fprintln(&b, "  dotbot-go -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml --dry-run")
		fmt.Fprintln(&b, "  dotbot-go -c install.conf.yaml --only link -vv")
	} else if cmd.Name() == "plan" {
		fmt.Fprintln(&b, "  dotbot-go plan -c install.conf.yaml")
		fmt.Fprintln(&b, "  dotbot-go plan -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml")
		fmt.Fprintln(&b, "  dotbot-go plan -c install.conf.yaml --output json")
	} else {
		fmt.Fprintln(&b, "  dotbot-go validate -c install.conf.yaml")
		fmt.Fprintln(&b, "  dotbot-go validate -d ~/.dotfiles -c ~/.dotfiles/install.conf.yaml")
		fmt.Fprintln(&b, "  dotbot-go validate -c install.conf.yaml --only link")
	}
	fmt.Fprintln(&b)

	if len(cmd.Commands()) > 0 {
		helpSection(&b, color, "Commands")
		fmt.Fprintln(&b, "  validate   validate configuration without applying changes")
		fmt.Fprintln(&b, "  plan       print planned operations without applying changes")
		fmt.Fprintln(&b)
	}

	helpSection(&b, color, "Built-In Directives")
	fmt.Fprintln(&b, "  defaults   set directive defaults")
	fmt.Fprintln(&b, "  link       create symlinks or hardlinks")
	fmt.Fprintln(&b, "  create     create directories")
	fmt.Fprintln(&b, "  shell      run setup commands")
	fmt.Fprintln(&b, "  clean      remove broken links")
	fmt.Fprintln(&b)

	helpSection(&b, color, "Flags")
	helpFlagGroup(&b, color, cmd, "Workflow", []string{
		"config-file",
		"base-directory",
		"dry-run",
		"exit-on-failure",
	})
	helpFlagGroup(&b, color, cmd, "Filtering", []string{
		"only",
		"except",
	})
	helpFlagGroup(&b, color, cmd, "Output", []string{
		"output",
		"quiet",
		"verbose",
		"force-color",
		"no-color",
		"version",
		"help",
	})
	helpSection(&b, color, "Color")
	fmt.Fprintln(&b, "  Colors are automatic for terminals.")
	fmt.Fprintln(&b, "  Use --force-color for redirected color, or --no-color for plain output.")
	return b.String()
}

func helpTitle(b *strings.Builder, color bool, cmd *cobra.Command) {
	border := strings.Repeat("-", 64)
	fmt.Fprintf(b, "%s\n", helpColor(color, "\033[36m", "+"+border+"+"))
	helpTitleLine(b, color, "\033[1;36m", "dotbot-go")
	helpTitleLine(b, color, "", cmd.Short)
	fmt.Fprintf(b, "%s\n\n", helpColor(color, "\033[36m", "+"+border+"+"))
}

func helpTitleLine(b *strings.Builder, color bool, code, text string) {
	width := 62
	if len(text) > width {
		text = text[:width]
	}
	padding := strings.Repeat(" ", width-len(text))
	fmt.Fprintf(b, "| %s%s |\n", helpColor(color, code, text), padding)
}

func helpSection(b *strings.Builder, color bool, title string) {
	fmt.Fprintf(b, "%s\n", helpColor(color, "\033[1;37m", title))
}

func helpFlagGroup(b *strings.Builder, color bool, cmd *cobra.Command, title string, names []string) {
	rows := flagRows(cmd, names)
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(b, "%s\n", helpColor(color, "\033[1;34m", title))
	width := 0
	for _, row := range rows {
		if len(row.name) > width {
			width = len(row.name)
		}
	}
	for _, row := range rows {
		padding := strings.Repeat(" ", width-len(row.name))
		descriptionWidth := 76 - width - 4
		lines := wrapWords(row.usage, descriptionWidth)
		if len(lines) == 0 {
			lines = []string{""}
		}
		fmt.Fprintf(b, "  %s%s  %s\n", helpColor(color, "\033[36m", row.name), padding, lines[0])
		for _, line := range lines[1:] {
			fmt.Fprintf(b, "  %s  %s\n", strings.Repeat(" ", width), line)
		}
	}
	fmt.Fprintln(b)
}

type flagRow struct {
	name  string
	usage string
}

func flagRows(cmd *cobra.Command, names []string) []flagRow {
	rows := []flagRow{}
	for _, flagName := range names {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			continue
		}
		if flag.Hidden {
			continue
		}
		name := flagDisplayName(flag.Name)
		if flag.Shorthand != "" {
			name = "-" + flag.Shorthand + ", " + name
		}
		rows = append(rows, flagRow{name: name, usage: singleLine(flag.Usage)})
	}
	return rows
}

func flagDisplayName(name string) string {
	switch name {
	case "base-directory":
		return "--base-directory <dir>"
	case "config-file":
		return "--config-file <file>"
	case "except":
		return "--except <directive>"
	case "only":
		return "--only <directive>"
	case "output":
		return "--output <format>"
	default:
		return "--" + name
	}
}

func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func wrapWords(s string, width int) []string {
	if width < 20 {
		width = 20
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := []string{}
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	return append(lines, line)
}

func helpColorEnabled(cmd *cobra.Command, stdout io.Writer) bool {
	if flag := cmd.Flags().Lookup("no-color"); flag != nil && flag.Value.String() == "true" {
		return false
	}
	if flag := cmd.Flags().Lookup("force-color"); flag != nil && flag.Value.String() == "true" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func helpColor(enabled bool, code, text string) string {
	if !enabled || code == "" {
		return text
	}
	return code + text + "\033[0m"
}
