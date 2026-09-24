
package application_example

import (
	"context"
	"errors"
	"fmt"
	"io"
	"golang-template/version"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"

	"github.com/wrouesnel/kongutil"

	"github.com/chigopher/pathlib"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.yaml.in/yaml/v4"

	"github.com/alecthomas/kong"
)

// Name is the name of this subapplication.
const Name = "application"

//nolint:gochecknoglobals
type CLIConfig struct {
	Version kong.VersionFlag `help:"Show version number"`

	CpuProfile bool `help:"Enable CPU Profiling"`
	MemProfile bool `help:"Enable Memory Profiling"`

	CpuProfileOutputPath string `default:"${application_name}.cpu.pprof" help:"CPU Profiling Output Path"`
	MemProfileOutputPath string `default:"${application_name}.mem.pprof" help:"Memory Profiling Output Path"`
	GoRoutineOutputPath  string `default:"${application_name}.goroutines.txt" help:"Go Routine Dump Output Path"`

	Logging struct {
		Level       string   `default:"info"    help:"logging level"`
		Format      string   `default:"console" enum:"console,json"  help:"logging format (${enum})"`
		OutputPaths []string `default:"stderr" help:"Paths to log to"`
	} `embed:"" prefix:"log-"`

	ConfigFile *pathlib.Path `default:"${default_configpath}" help:"Path to top-level configuration file"`

	//Assets assets.Config `embed:"" prefix:"assets."`

	//Debug struct {
	//	Assets struct {
	//		List struct {
	//		} `cmd:"" help:"list embedded files in the binary"`
	//		Cat struct {
	//			Filename string `arg:"" help:"embedded file to emit to stdout" name:"filename"`
	//		} `cmd:"" help:"output the specified file to stdout"`
	//	} `cmd:""`
	//} `cmd:""`

	// config tracks the actual loaded config nodes
	config *yaml.Node
}

func (c *CLIConfig) BeforeReset() error {
	c.config = new(yaml.Node)
	return nil
}

// Entrypoint is the real application entrypoint. This structure allows test packages to E2E-style tests invoking commmands
// as though they are on the command line, but using built-in coverage tools. Stub-main under the `cmd` package calls this
// function.
func Entrypoint(ctx context.Context) int {
	appCtx, appCancel := context.WithCancel(ctx)
	defer appCancel()

	deferredLogs := []string{}

	// Command line parsing can now happen
	vars := kong.Vars{
		"version":            version.Version,
		"default_configpath": fmt.Sprintf("%s.yml", version.Name),
		"application_name": fmt.Sprintf("%s-%s", version.Name, Name)
	}

	// cli is the root entrypoint config parsed by kong and made available to subcommands via
	// a binding.
	var cli EntrypointConfig

	kongCtx := kong.Parse(&cli,
		kong.Description(version.Description),
		kong.DefaultEnvars(version.Name),
		kongutil.PathlibMapper(),
		vars)

	// Initialize logging as soon as possible
	logConfig := zap.NewProductionConfig()
	if err := logConfig.Level.UnmarshalText([]byte(CLI.Logging.Level)); err != nil {
		deferredLogs = append(deferredLogs, err.Error())
	}
	logConfig.Encoding = CLI.Logging.Format
	logConfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	if CLI.Logging.Format == "console" {
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	logConfig.OutputPaths = CLI.Logging.OutputPaths

	logger, err := logConfig.Build()
	if err != nil {
		// Error unhandled since this is a very early failure
		_, _ = io.WriteString(stdErr, "Failure while building logger")
		return 1
	}

	collectMemoryProfile := func() {
		if CLI.MemProfileOutputPath == "" {
			logger.Warn("Memory profile requested but writing output disabled")
			return
		}
		idx := 0
		outputPath := fmt.Sprintf("%v.%v", CLI.MemProfileOutputPath, idx)
		for {
			outputPath = fmt.Sprintf("%v.%v", CLI.MemProfileOutputPath, idx)
			if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
				break
			}
			idx += 1
			// Failsafe
			if idx > 1000000 {
				logger.Error("Didn't find a usable file to write the dump too!")
				return
			}
		}
		logger.Info("Writing Memory Profile: Started", zap.String("filename", outputPath))
		f, err := os.Create(outputPath)
		if err != nil {
			logger.Error("Could not create memory profile output", zap.String("filename", outputPath))
			return
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
			logger.Error("Writing Memory Profile: Error", zap.Error(err))
			return
		}
		logger.Info("Writing Memory Profile: Success", zap.String("filename", outputPath))
	}

	logGoRoutineProfile := func() {
		if CLI.GoRoutineOutputPath == "" {
			logger.Warn("Go routine profile requested but writing output disabled")
			return
		}
		idx := 0
		outputPath := fmt.Sprintf("%v.%v", CLI.GoRoutineOutputPath, idx)
		for {
			outputPath = fmt.Sprintf("%v.%v", CLI.GoRoutineOutputPath, idx)
			if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
				break
			}
			idx += 1
			// Failsafe
			if idx > 1000000 {
				logger.Error("Didn't find a usable file to write the dump too!")
				return
			}
		}
		logger.Info("Writing Go Routine Stack Dump: Start", zap.String("filename", outputPath))
		f, err := os.Create(outputPath)
		if err != nil {
			logger.Error("Could not create goroutines output", zap.String("filename", outputPath))
			return
		}
		defer f.Close()

		if err := pprof.Lookup("goroutine").WriteTo(f, 1); err != nil {
			logger.Error("Writing Go Routine Stack Dump: Error", zap.Error(err))
		} else {
			logger.Info("Writing Go Routine Stack Dump: Success", zap.String("filename", outputPath))
		}
	}
	// Install as the global logger
	zap.ReplaceGlobals(logger)

	logger.Debug("Configuring signal handling")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGUSR2)
	sigCtx, cancelFn := context.WithCancel(appCtx)
	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGUSR1:
				logger.Info("Caught signal - collecting memory profile", zap.String("signal", sig.String()))
				collectMemoryProfile()
			case syscall.SIGUSR2:
				logger.Info("Caught signal - dumping current goroutines", zap.String("signal", sig.String()))
				logGoRoutineProfile()
			default:
				logger.Info("Caught signal - exiting", zap.String("signal", sig.String()))
				cancelFn()
			}
		}
		logger.Info("Signal handling channel closed")
	}()

	// Emit deferred logs
	for _, line := range deferredLogs {
		logger.Error(line)
	}

	//logger.Info("Using config paths", zap.Strings("configDirs", configDirs))

	//logger.Info("Configuring asset handling", zap.Bool("use-filesystem", CLI.Assets.UseFilesystem))
	//assets.UseFilesystem(CLI.Assets.UseFilesystem)

	// config is the main configuration file.
	config := EntrypointConfig{}
	err := yaml.Load([]byte(config), &config, commonproviders.YamlOption())
	if err != nil {
		logger.Error("Could not create YAML parser", zap.Error(err))
		return 1
	}

	if err := UnmarshalConfig(CLI.ConfigFile, &config); err != nil {
		logger.Error("Could parse configuration file", zap.Error(err))
		return 1
	}

	// Main application context
	ctx.BindTo(sigCtx, (*context.Context)(nil))
	// Main command line 
	ctx.Bind(&cli)
	// Main config file
	ctx.Bind(&config)
	// kongCtx if needed.
	ctx.Bind(kongCtx
	if CLI.CpuProfile {
		logger.Info("Writing CPU profile", zap.String("filename", CLI.CpuProfileOutputPath))
		f, err := os.Create(CLI.CpuProfileOutputPath)
		if err != nil {
			logger.Fatal("Could not create CPU profile output", zap.String("filename", CLI.CpuProfileOutputPath))
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			logger.Fatal("Could not start CPU profile: ", zap.Error(err))
		}
		defer pprof.StopCPUProfile()
	}

	if err := ctx.Run(); err != nil {
		logger.Error("Error from command", zap.Error(err))
	}

	if CLI.MemProfile {
		collectMemoryProfile()
	}

	logger.Info("Exiting normally")
	return 0
}
