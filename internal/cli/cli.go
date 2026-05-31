package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"dotbot-go/internal/config"
	"dotbot-go/internal/core"
	"dotbot-go/internal/log"
)

const version = "0.1.0"

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

func Execute() int {
	opts := &options{}
	cmd := newRootCommand(opts)
	if err := cmd.Execute(); err != nil {
		if _, ok := err.(exitError); ok {
			return 1
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func newRootCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "dotbot-go",
		Short:         "A Go port of Dotbot for bootstrapping dotfiles",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.superQuiet, "super-quiet", "Q", false, "deprecated quiet mode")
	_ = cmd.Flags().MarkHidden("super-quiet")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "suppress most output")
	cmd.Flags().CountVarP(&opts.verbose, "verbose", "v", "enable verbose output\n-v: show informational messages\n-vv: also, set shell commands stderr/stdout to true")
	cmd.Flags().StringVarP(&opts.baseDirectory, "base-directory", "d", "", "execute commands from within BASE_DIR")
	cmd.Flags().StringArrayVarP(&opts.configFiles, "config-file", "c", nil, "run commands given in CONFIG_FILE")
	cmd.Flags().StringArrayVarP(&opts.plugins, "plugin", "p", nil, "load PLUGIN as a plugin")
	cmd.Flags().BoolVar(&opts.disableBuiltInPlugins, "disable-built-in-plugins", false, "disable built-in plugins")
	cmd.Flags().StringArrayVar(&opts.pluginDirs, "plugin-dir", nil, "deprecated plugin directory")
	_ = cmd.Flags().MarkHidden("plugin-dir")
	cmd.Flags().StringSliceVar(&opts.only, "only", nil, "only run specified directives")
	cmd.Flags().StringSliceVar(&opts.skip, "except", nil, "skip specified directives")
	cmd.Flags().BoolVarP(&opts.dryRun, "dry-run", "n", false, "print what would be done, without doing it")
	cmd.Flags().BoolVar(&opts.forceColor, "force-color", false, "force color output")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "disable color output")
	cmd.Flags().BoolVar(&opts.showVersion, "version", false, "show program's version number and exit")
	cmd.Flags().BoolVarP(&opts.exitOnFailure, "exit-on-failure", "x", false, "exit after first failed directive")
	return cmd
}

func run(opts *options) error {
	logger := log.New(os.Stdout)
	if opts.showVersion {
		fmt.Fprintf(os.Stdout, "Dotbot-Go version %s\n", version)
		return nil
	}
	if opts.superQuiet || opts.quiet {
		logger.SetLevel(log.Warning)
	}
	if opts.verbose > 0 {
		if opts.verbose == 1 {
			logger.SetLevel(log.Info)
		} else {
			logger.SetLevel(log.Debug)
		}
	}
	if opts.forceColor && opts.noColor {
		logger.Error("`--force-color` and `--no-color` cannot both be provided")
		return exitError{}
	}
	if opts.forceColor {
		logger.UseColor(true)
	} else if opts.noColor {
		logger.UseColor(false)
	}
	if len(opts.configFiles) == 0 {
		logger.Error("No configuration file specified")
		return exitError{}
	}
	tasks, err := config.Read(opts.configFiles)
	if err != nil {
		logger.Error(err.Error())
		return exitError{}
	}
	if len(tasks) == 0 {
		logger.Warning("No tasks given in configuration, no work to do")
	}
	base := opts.baseDirectory
	if base == "" {
		base = filepath.Dir(abs(opts.configFiles[0]))
	} else {
		base = abs(base)
	}
	if err := os.Chdir(base); err != nil {
		logger.Error(err.Error())
		return exitError{}
	}
	handlers := core.BuiltIns()
	if opts.disableBuiltInPlugins {
		handlers = []core.Handler{}
	}
	coreOpts := core.Options{
		Only:                  opts.only,
		Skip:                  opts.skip,
		ExitOnFailure:         opts.exitOnFailure,
		DryRun:                opts.dryRun,
		Verbose:               opts.verbose,
		DisableBuiltInPlugins: opts.disableBuiltInPlugins,
	}
	dispatcher, err := core.NewDispatcher(base, coreOpts, logger, handlers)
	if err != nil {
		logger.Error(err.Error())
		return exitError{}
	}
	if len(opts.plugins) > 0 || len(opts.pluginDirs) > 0 {
		logger.Warning("Go migration does not support Python plugin loading")
		logger.Debug("Unsupported plugin inputs: " + strings.Join(append(opts.plugins, opts.pluginDirs...), ", "))
	}
	success, err := dispatcher.Dispatch(tasks)
	if err != nil {
		logger.Error(err.Error())
		return exitError{}
	}
	if success && len(opts.plugins) == 0 && len(opts.pluginDirs) == 0 {
		logger.Info("All tasks executed successfully")
		return nil
	}
	logger.Error("Some tasks were not executed successfully")
	return exitError{}
}

type exitError struct{}

func (exitError) Error() string { return "" }

func abs(path string) string {
	out, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return out
}
