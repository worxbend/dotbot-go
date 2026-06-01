package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dotbot-go/internal/config"
	"dotbot-go/internal/core"
	"dotbot-go/internal/fsops"
	"dotbot-go/internal/log"
	"dotbot-go/internal/shell"
)

const Version = "0.1.0"

var ErrExit = errors.New("app: exit")

type Options struct {
	SuperQuiet            bool
	Quiet                 bool
	Verbose               int
	BaseDirectory         string
	ConfigFiles           []string
	Plugins               []string
	PluginDirs            []string
	DisableBuiltInPlugins bool
	Only                  []string
	Skip                  []string
	DryRun                bool
	ForceColor            bool
	NoColor               bool
	ExitOnFailure         bool
	ShowVersion           bool
}

type Dependencies struct {
	ConfigReader func([]string) ([]config.Task, error)
	FS           fsops.FS
	Shell        shell.Runner
	Clock        func() time.Time
	Chdir        func(string) error
	Handlers     []core.Handler
}

func Run(ctx context.Context, opts Options, stdout io.Writer, deps Dependencies) error {
	if stdout == nil {
		stdout = io.Discard
	}
	deps = deps.withDefaults()
	logger := log.New(stdout)
	if opts.ShowVersion {
		fmt.Fprintf(stdout, "Dotbot-Go version %s\n", Version)
		return nil
	}
	configureLogger(logger, opts)
	if opts.ForceColor && opts.NoColor {
		logger.Error("`--force-color` and `--no-color` cannot both be provided")
		return ErrExit
	}
	if opts.ForceColor {
		logger.UseColor(true)
	} else if opts.NoColor {
		logger.UseColor(false)
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
	if len(tasks) == 0 {
		logger.Warning("No tasks given in configuration, no work to do")
	}
	base := opts.BaseDirectory
	if base == "" {
		base = filepath.Dir(abs(opts.ConfigFiles[0]))
	} else {
		base = abs(base)
	}
	if err := deps.Chdir(base); err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	logger.Action(runSummary(len(tasks), len(opts.ConfigFiles), base, opts.DryRun))
	handlers := deps.Handlers
	if opts.DisableBuiltInPlugins {
		handlers = []core.Handler{}
	}
	coreOpts := core.Options{
		Only:                  opts.Only,
		Skip:                  opts.Skip,
		ExitOnFailure:         opts.ExitOnFailure,
		DryRun:                opts.DryRun,
		Verbose:               opts.Verbose,
		DisableBuiltInPlugins: opts.DisableBuiltInPlugins,
	}
	dispatcher, err := core.NewDispatcher(core.DispatcherConfig{
		BaseDirectory: base,
		Options:       coreOpts,
		Logger:        logger,
		Handlers:      handlers,
		FS:            deps.FS,
		Shell:         deps.Shell,
		Clock:         deps.Clock,
	})
	if err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	if len(opts.Plugins) > 0 || len(opts.PluginDirs) > 0 {
		logger.Warning("Go migration does not support Python plugin loading")
		logger.Debug("Unsupported plugin inputs: " + strings.Join(append(opts.Plugins, opts.PluginDirs...), ", "))
	}
	success, err := dispatcher.Dispatch(ctx, tasks)
	if err != nil {
		logger.Error(err.Error())
		return ErrExit
	}
	if success && len(opts.Plugins) == 0 && len(opts.PluginDirs) == 0 {
		logger.Info("All tasks executed successfully")
		return nil
	}
	logger.Error("Some tasks were not executed successfully")
	return ErrExit
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
