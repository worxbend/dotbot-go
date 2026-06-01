package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"dotbot-go/internal/app"
)

type options struct {
	superQuiet            bool
	quiet                 bool
	verbose               int
	baseDirectory         string
	configFiles           []string
	plugins               []string
	pluginDirs            []string
	disableBuiltInPlugins bool
	only                  []string
	skip                  []string
	dryRun                bool
	forceColor            bool
	noColor               bool
	exitOnFailure         bool
	showVersion           bool
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
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run(ctx, opts.appOptions(), stdout, app.Dependencies{})
		},
	}
	cmd.Flags().BoolVarP(&opts.superQuiet, "super-quiet", "Q", false, "deprecated quiet mode")
	mustMarkHidden(cmd, "super-quiet")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "suppress most output")
	cmd.Flags().CountVarP(&opts.verbose, "verbose", "v", "enable verbose output\n-v: show informational messages\n-vv: also, set shell commands stderr/stdout to true")
	cmd.Flags().StringVarP(&opts.baseDirectory, "base-directory", "d", "", "execute commands from within BASE_DIR")
	cmd.Flags().StringArrayVarP(&opts.configFiles, "config-file", "c", nil, "run commands given in CONFIG_FILE")
	cmd.Flags().StringArrayVarP(&opts.plugins, "plugin", "p", nil, "load PLUGIN as a plugin")
	cmd.Flags().BoolVar(&opts.disableBuiltInPlugins, "disable-built-in-plugins", false, "disable built-in plugins")
	cmd.Flags().StringArrayVar(&opts.pluginDirs, "plugin-dir", nil, "deprecated plugin directory")
	mustMarkHidden(cmd, "plugin-dir")
	cmd.Flags().StringSliceVar(&opts.only, "only", nil, "only run specified directives")
	cmd.Flags().StringSliceVar(&opts.skip, "except", nil, "skip specified directives")
	cmd.Flags().BoolVarP(&opts.dryRun, "dry-run", "n", false, "print what would be done, without doing it")
	cmd.Flags().BoolVar(&opts.forceColor, "force-color", false, "force color output")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "disable color output")
	cmd.Flags().BoolVar(&opts.showVersion, "version", false, "show program's version number and exit")
	cmd.Flags().BoolVarP(&opts.exitOnFailure, "exit-on-failure", "x", false, "exit after first failed directive")
	return cmd
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
		SuperQuiet:            o.superQuiet,
		Quiet:                 o.quiet,
		Verbose:               o.verbose,
		BaseDirectory:         o.baseDirectory,
		ConfigFiles:           o.configFiles,
		Plugins:               o.plugins,
		PluginDirs:            o.pluginDirs,
		DisableBuiltInPlugins: o.disableBuiltInPlugins,
		Only:                  o.only,
		Skip:                  o.skip,
		DryRun:                o.dryRun,
		ForceColor:            o.forceColor,
		NoColor:               o.noColor,
		ExitOnFailure:         o.exitOnFailure,
		ShowVersion:           o.showVersion,
	}
}
