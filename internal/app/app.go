package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"dotbot-go/internal/config"
	"dotbot-go/internal/core"
	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	"dotbot-go/internal/shell"
)

// Version is overridden by release builds with -ldflags "-X ...Version=<version>".
var Version = "0.2.1"

// ErrExit reports a controlled application failure whose user-facing message has
// already been written to the configured output stream.
var ErrExit = errors.New("app: exit")

// Options describes one dotbot-go invocation after CLI parsing.
type Options struct {
	// SuperQuiet preserves the deprecated Dotbot-compatible -Q/--super-quiet flag.
	SuperQuiet bool
	// Quiet suppresses informational and action output while preserving warnings.
	Quiet bool
	// Verbose increases log detail; values greater than one also expose shell output.
	Verbose int
	// BaseDirectory is the repository root used to resolve relative config paths.
	BaseDirectory string
	// ConfigFiles lists configuration files to read in order.
	ConfigFiles []string
	// Only limits execution and planning to the named directives.
	Only []string
	// Skip excludes the named directives from execution and planning.
	Skip []string
	// DryRun previews filesystem and shell actions without applying them.
	DryRun bool
	// ForceColor enables ANSI color even when stdout is not a terminal.
	ForceColor bool
	// NoColor disables ANSI color regardless of terminal capabilities.
	NoColor bool
	// ExitOnFailure stops directive dispatch after the first failed action.
	ExitOnFailure bool
	// ShowVersion prints Version and exits without reading configuration.
	ShowVersion bool
	// Validate checks configuration and planned operations without applying them.
	Validate bool
	// Plan prints planned operations without applying them.
	Plan bool
	// Output selects the plan output format.
	Output string
}

// Dependencies groups side-effecting collaborators so tests and adapters can run
// the application without touching the host filesystem or shell.
type Dependencies struct {
	// ConfigReader reads all config files into task directives.
	ConfigReader func([]string) ([]config.Task, error)
	// FS performs filesystem operations for directive handlers.
	FS fsops.FS
	// Shell runs shell directive commands.
	Shell shell.Runner
	// Clock supplies timestamps for backup names.
	Clock func() time.Time
	// Chdir changes the process working directory for apply runs.
	Chdir func(string) error
	// Handlers override the built-in directive handlers when provided.
	Handlers []core.Handler
}

// Run executes one application invocation and returns ErrExit for expected
// user-facing failures.
func Run(ctx context.Context, opts Options, stdout io.Writer, deps Dependencies) error {
	if stdout == nil {
		stdout = io.Discard
	}
	deps = deps.withDefaults()
	logger := log.New(stdout)
	if opts.ShowVersion {
		if _, err := fmt.Fprintf(stdout, "Dotbot-Go version %s\n", Version); err != nil {
			return err
		}
		return nil
	}
	configureLogger(logger, opts)
	if err := configureColor(logger, opts); err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	if len(opts.ConfigFiles) == 0 {
		logger.Error("No configuration file specified")
		return ErrExit
	}
	tasks, err := deps.ConfigReader(opts.ConfigFiles)
	if err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	if len(tasks) == 0 && (!opts.Plan || !isStructuredOutput(opts.Output)) {
		logger.Warning("No tasks given in configuration, no work to do")
	}
	base, err := baseDirectory(opts)
	if err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	coreOpts := coreOptions(opts)
	if opts.Validate || opts.Plan {
		dispatcher, err := newDispatcher(base, coreOpts, logger, deps)
		if err != nil {
			logger.Error(err.Error())
			return ErrExit
		}
		plan, err := dispatcher.Plan(tasks)
		if err != nil {
			logger.Error(err.Error())
			return ErrExit
		}
		if opts.Plan {
			if err := writePlanOutput(stdout, opts.Output, plan, len(tasks), len(opts.ConfigFiles), base); err != nil {
				logger.Error(err.Error())
				return ErrExit
			}
			return nil
		}
		logger.Action(validateSummary(len(tasks), len(opts.ConfigFiles), len(plan.Operations), base))
		return nil
	}
	if err := deps.Chdir(base); err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	logger.Action(runSummary(len(tasks), len(opts.ConfigFiles), base, opts.DryRun))
	dispatcher, err := newDispatcher(base, coreOpts, logger, deps)
	if err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	success, err := dispatcher.Dispatch(ctx, tasks)
	if err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	if success {
		logger.Info("All tasks executed successfully")
		return nil
	}
	logger.Error("Some tasks were not executed successfully")
	return ErrExit
}

func configureColor(logger *log.Logger, opts Options) error {
	if opts.ForceColor && opts.NoColor {
		return fmt.Errorf("`--force-color` and `--no-color` cannot both be provided")
	}
	if opts.ForceColor {
		logger.UseColor(true)
	} else if opts.NoColor {
		logger.UseColor(false)
	}
	return nil
}

func baseDirectory(opts Options) (string, error) {
	if opts.BaseDirectory == "" {
		if len(opts.ConfigFiles) == 0 {
			return "", fmt.Errorf("no configuration file specified")
		}
		return filepath.Dir(abs(opts.ConfigFiles[0])), nil
	}
	return abs(opts.BaseDirectory), nil
}

func coreOptions(opts Options) core.Options {
	return core.Options{
		Only:          opts.Only,
		Skip:          opts.Skip,
		ExitOnFailure: opts.ExitOnFailure,
		DryRun:        opts.DryRun,
		Verbose:       opts.Verbose,
	}
}

func newDispatcher(base string, coreOpts core.Options, logger *log.Logger, deps Dependencies) (*core.Dispatcher, error) {
	return core.NewDispatcher(core.DispatcherConfig{
		BaseDirectory: base,
		Options:       coreOpts,
		Logger:        logger,
		Handlers:      deps.Handlers,
		FS:            deps.FS,
		Shell:         deps.Shell,
		Clock:         deps.Clock,
	})
}

func runSummary(taskCount, configCount int, base string, dryRun bool) string {
	mode := "apply"
	if dryRun {
		mode = "dry-run"
	}
	return fmt.Sprintf(
		"Starting %s with %d task(s), %d config file(s), base %s",
		mode,
		taskCount,
		configCount,
		base,
	)
}

func validateSummary(taskCount, configCount, operationCount int, base string) string {
	return fmt.Sprintf(
		"Configuration is valid: %d task(s), %d config file(s), %d planned operation(s), base %s",
		taskCount,
		configCount,
		operationCount,
		base,
	)
}

func (d Dependencies) withDefaults() Dependencies {
	if d.ConfigReader == nil {
		d.ConfigReader = config.Read
	}
	if d.FS == nil {
		d.FS = fsops.OSFS{}
	}
	if d.Shell == nil {
		d.Shell = shell.OSRunner{}
	}
	if d.Clock == nil {
		d.Clock = time.Now
	}
	if d.Chdir == nil {
		d.Chdir = os.Chdir
	}
	if d.Handlers == nil {
		d.Handlers = core.BuiltIns()
	}
	return d
}

func configureLogger(logger *log.Logger, opts Options) {
	if opts.SuperQuiet || opts.Quiet {
		logger.SetLevel(log.Warning)
	}
	if opts.Verbose > 0 {
		if opts.Verbose == 1 {
			logger.SetLevel(log.Info)
			return
		}
		logger.SetLevel(log.Debug)
	}
}

func abs(path string) string {
	out, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return out
}
